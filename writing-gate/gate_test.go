package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// writeTestFile はテスト用のファイルを作る｡
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// toolUseLine は tool_use 1 件だけを含む transcript 行を作る｡
func toolUseLine(tool, filePath string) string {
	return `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"tool_use","name":"` + tool + `","input":{"file_path":"` + filePath + `"}}]}}`
}

// toolUseInputLine は任意の入力を持つ tool_use 1 件の transcript 行を作る｡
// new_string や content は改行や引用符を含むため、JSON へは Marshal で埋める｡
func toolUseInputLine(t *testing.T, tool string, input map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return `{"type":"assistant","message":{"role":"assistant","content":[` +
		`{"type":"tool_use","name":"` + tool + `","input":` + string(raw) + `}]}}`
}

func TestEditedMarkdown(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTestFile(t, dir, "session.jsonl", strings.Join([]string{
		toolUseLine("Write", "/repo/a.md"),
		toolUseLine("Read", "/repo/never.md"),
		toolUseLine("Edit", "/repo/b.md"),
		toolUseLine("Write", "/repo/a.md"), // 重複は 1 件に畳む
		toolUseLine("Write", "/repo/main.go"),
		`{"type":"user","message":{"role":"user","content":[]}}`,
		`これは JSON ではない行`,
		toolUseLine("NotebookEdit", "/repo/c.md"),
	}, "\n")+"\n")

	tr, err := ReadTranscript(transcript)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}
	paths := make([]string, 0, len(tr.Edited))
	for _, e := range tr.Edited {
		paths = append(paths, e.Path)
	}
	want := []string{"/repo/a.md", "/repo/b.md", "/repo/c.md"}
	if !slices.Equal(paths, want) {
		t.Errorf("Edited のパス = %v, want %v", paths, want)
	}
}

// userLine は文字列本文のユーザー発言 1 件の transcript 行を作る｡
// ユーザーが打った slash command はこの形で残る｡
func userLine(t *testing.T, content string, meta bool) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"type":    "user",
		"isMeta":  meta,
		"message": map[string]any{"role": "user", "content": content},
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return string(raw)
}

const (
	gateOffCommand = "<command-message>gate-off</command-message>\n<command-name>/japanese-writer:gate-off</command-name>"
	gateOnCommand  = "<command-message>gate-on</command-message>\n<command-name>/japanese-writer:gate-on</command-name>"
)

func TestReadTranscriptSeen(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTestFile(t, dir, "session.jsonl", strings.Join([]string{
		userLine(t, "Stop hook feedback:\n/repo/a.md\n  error:3 [process-leak] 痕跡 (wg:0123abcd)\n", true),
		// 2 件目の block｡1 件目と同じ ID が再び載ることもある
		userLine(t, "Stop hook feedback:\n  error:3 [process-leak] 痕跡 (wg:0123abcd)\n  error:9 [style-mix] 混在 (wg:ffff0000)\n", true),
		userLine(t, "これは本文に wg: とだけ書いたユーザー発言", false),
	}, "\n")+"\n")

	tr, err := ReadTranscript(transcript)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}
	want := map[string]bool{"0123abcd": true, "ffff0000": true}
	if !maps.Equal(tr.Seen, want) {
		t.Errorf("Seen = %v, want %v", tr.Seen, want)
	}
}

func TestReadTranscriptGateOff(t *testing.T) {
	dir := t.TempDir()

	read := func(t *testing.T, name string, lines ...string) *Transcript {
		t.Helper()
		path := writeTestFile(t, dir, name, strings.Join(lines, "\n")+"\n")
		tr, err := ReadTranscript(path)
		if err != nil {
			t.Fatalf("ReadTranscript: %v", err)
		}
		return tr
	}

	t.Run("何も打っていなければ有効", func(t *testing.T) {
		if tr := read(t, "none.jsonl", toolUseLine("Write", "/repo/a.md")); tr.GateOff {
			t.Error("GateOff = true, want false")
		}
	})

	t.Run("ユーザーが gate-off を打てば止まる", func(t *testing.T) {
		if tr := read(t, "off.jsonl", userLine(t, gateOffCommand, false)); !tr.GateOff {
			t.Error("GateOff = false, want true")
		}
	})

	t.Run("後に gate-on を打てば戻る", func(t *testing.T) {
		tr := read(t, "on.jsonl", userLine(t, gateOffCommand, false), userLine(t, gateOnCommand, false))
		if tr.GateOff {
			t.Error("GateOff = true, want false")
		}
	})

	t.Run("gate-on の後にまた gate-off を打てば止まる", func(t *testing.T) {
		tr := read(t, "off-again.jsonl",
			userLine(t, gateOffCommand, false), userLine(t, gateOnCommand, false), userLine(t, gateOffCommand, false))
		if !tr.GateOff {
			t.Error("GateOff = false, want true")
		}
	})

	t.Run("本文が text ブロックの配列でも読む", func(t *testing.T) {
		line := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":` +
			strconv.Quote(gateOffCommand) + `}]}}`
		if tr := read(t, "array.jsonl", line); !tr.GateOff {
			t.Error("GateOff = false, want true")
		}
	})

	t.Run("Claude が Skill ツールで呼んでも止まらない", func(t *testing.T) {
		tr := read(t, "skill.jsonl",
			toolUseInputLine(t, "Skill", map[string]any{"skill": "japanese-writer:gate-off"}),
			`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"Launching skill: japanese-writer:gate-off"}]}}`)
		if tr.GateOff {
			t.Error("GateOff = true, want false")
		}
	})

	t.Run("hook の feedback に書式が引用されていても止まらない", func(t *testing.T) {
		tr := read(t, "meta.jsonl", userLine(t, "Stop hook feedback:\n止めるには "+gateOffCommand, true))
		if tr.GateOff {
			t.Error("GateOff = true, want false")
		}
	})
}

func TestWrittenFragments(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
		want []string
	}{
		{"old_string に無い行だけを残す", "a\nb\nc", "a\nB\nc", []string{"B"}},
		{"連続する書いた行は 1 つの断片にまとめる", "a\nb", "a\nX\nY", []string{"X\nY"}},
		{"離れた書いた行は別の断片にする", "a\nb\nc", "X\nb\nY", []string{"X", "Y"}},
		{"全部書き換えた", "x", "y\nz", []string{"y\nz"}},
		{"行が変わらなければ何も書いていない", "a\nb", "a\nb", nil},
		{"空行は数えない", "a", "\nB\n", []string{"B"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writtenFragments(tt.old, tt.new)
			if !slices.Equal(got, tt.want) {
				t.Errorf("writtenFragments(%q, %q) = %q, want %q", tt.old, tt.new, got, tt.want)
			}
		})
	}
}

func TestEditedMarkdownFragments(t *testing.T) {
	dir := t.TempDir()
	transcript := writeTestFile(t, dir, "session.jsonl", strings.Join([]string{
		toolUseInputLine(t, "Edit", map[string]any{
			"file_path": "/repo/a.md", "old_string": "x", "new_string": "一行目｡",
		}),
		toolUseInputLine(t, "MultiEdit", map[string]any{
			"file_path": "/repo/a.md",
			"edits": []map[string]any{
				{"old_string": "y", "new_string": "二行目｡"},
				{"old_string": "z", "new_string": "三行目｡"},
			},
		}),
		toolUseInputLine(t, "Edit", map[string]any{
			"file_path": "/repo/b.md", "old_string": "x", "new_string": "追記｡",
		}),
		toolUseInputLine(t, "Write", map[string]any{
			"file_path": "/repo/b.md", "content": "全体｡\n",
		}),
		toolUseInputLine(t, "NotebookEdit", map[string]any{
			"file_path": "/repo/c.md", "new_source": "セル｡",
		}),
		toolUseInputLine(t, "Write", map[string]any{
			"file_path": "/repo/d.md", "content": "全体｡\n",
		}),
		toolUseInputLine(t, "Edit", map[string]any{
			"file_path": "/repo/d.md", "old_string": "全体", "new_string": "差し替え",
		}),
	}, "\n")+"\n")

	tr, err := ReadTranscript(transcript)
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}
	got := tr.Edited
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %+v", len(got), got)
	}
	// Write の後の Edit で Whole が外れてはいけない｡
	if !got[3].Whole {
		t.Errorf("d.md = %+v, want Whole", got[3])
	}
	if got[0].Whole || !slices.Equal(got[0].Fragments, []string{"一行目｡", "二行目｡", "三行目｡"}) {
		t.Errorf("a.md = %+v, want Edit と MultiEdit の new_string", got[0])
	}
	// Write はファイル全体を書いたことになる｡
	if !got[1].Whole {
		t.Errorf("b.md = %+v, want Whole", got[1])
	}
	// 書いた本文を取り出せないツールは、ファイル全体を書いたものとして扱う｡
	if !got[2].Whole {
		t.Errorf("c.md = %+v, want Whole", got[2])
	}
}

func TestFocusLines(t *testing.T) {
	src := "# 見出し\n\n既存の行｡\n追記した行｡\n続けた行｡\n\n既存の行｡\n"

	t.Run("書いた断片が現れる行を返す", func(t *testing.T) {
		got := FocusLines(src, EditedFile{Fragments: []string{"追記した行｡\n続けた行｡"}})
		want := map[int]bool{4: true, 5: true}
		if !maps.Equal(got, want) {
			t.Errorf("FocusLines = %v, want %v", got, want)
		}
	})

	t.Run("同じ断片が複数回現れればすべて返す", func(t *testing.T) {
		got := FocusLines(src, EditedFile{Fragments: []string{"既存の行｡"}})
		want := map[int]bool{3: true, 7: true}
		if !maps.Equal(got, want) {
			t.Errorf("FocusLines = %v, want %v", got, want)
		}
	})

	t.Run("後の編集で消えた断片は数えない", func(t *testing.T) {
		got := FocusLines(src, EditedFile{Fragments: []string{"消した行｡", ""}})
		if len(got) != 0 {
			t.Errorf("FocusLines = %v, want 空", got)
		}
	})

	t.Run("Whole なら nil を返して文書全体を見る", func(t *testing.T) {
		if got := FocusLines(src, EditedFile{Whole: true}); got != nil {
			t.Errorf("FocusLines = %v, want nil", got)
		}
	})
}

func TestReadTranscriptMissingFile(t *testing.T) {
	if _, err := ReadTranscript(filepath.Join(t.TempDir(), "none.jsonl")); err == nil {
		t.Error("transcript が無ければエラーを返すはず")
	}
}

func TestGateConfigIsExcluded(t *testing.T) {
	cfg := &GateConfig{Exclude: defaultExcludes}

	tests := []struct {
		path string
		want bool
	}{
		{"/repo/tmp/draft.md", true},
		{"/repo/PLAN.md", true},
		{"/repo/CHANGELOG.md", true},
		{"/home/u/obsidian/2026-08-23.md", true},
		{"/repo/docs/minutes/kickoff.md", true},
		{"/home/u/.claude/projects/x/notes.md", true},
		{"/repo/docs/design.md", false},
		{"/repo/README.md", false},
		{"/repo/tmpfile.md", false}, // tmp を含むが tmp ディレクトリではない
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := cfg.IsExcluded(tt.path); got != tt.want {
				t.Errorf("IsExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadGateConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("ファイルが無ければ既定値", func(t *testing.T) {
		cfg, err := LoadGateConfig(filepath.Join(dir, "none.json"))
		if err != nil {
			t.Fatalf("LoadGateConfig: %v", err)
		}
		if !slices.Equal(cfg.Exclude, defaultExcludes) {
			t.Errorf("Exclude = %v, want 既定値", cfg.Exclude)
		}
	})

	t.Run("exclude_extra は既定へ追記する", func(t *testing.T) {
		path := writeTestFile(t, dir, "extra.json", `{"exclude_extra":["**/scratch/**"]}`)
		cfg, err := LoadGateConfig(path)
		if err != nil {
			t.Fatalf("LoadGateConfig: %v", err)
		}
		if !cfg.IsExcluded("/repo/scratch/a.md") {
			t.Error("追記した除外が効いていない")
		}
		if !cfg.IsExcluded("/repo/PLAN.md") {
			t.Error("既定の除外が消えている")
		}
	})

	t.Run("exclude は既定を置き換える", func(t *testing.T) {
		path := writeTestFile(t, dir, "replace.json", `{"exclude":["**/only/**"]}`)
		cfg, err := LoadGateConfig(path)
		if err != nil {
			t.Fatalf("LoadGateConfig: %v", err)
		}
		if cfg.IsExcluded("/repo/PLAN.md") {
			t.Error("既定の除外が残っている")
		}
		if !cfg.IsExcluded("/repo/only/a.md") {
			t.Error("指定した除外が効いていない")
		}
	})

	t.Run("壊れた JSON はエラー", func(t *testing.T) {
		path := writeTestFile(t, dir, "broken.json", `{`)
		if _, err := LoadGateConfig(path); err == nil {
			t.Error("エラーを返すはず")
		}
	})
}

func TestSelectTargets(t *testing.T) {
	dir := t.TempDir()
	a := writeTestFile(t, dir, "a.md", "本文｡\n")
	b := writeTestFile(t, dir, "b.md", "本文｡\n")

	// 既定の除外は使わない。t.TempDir() は Linux では /tmp 配下に作られ、
	// 既定の `**/tmp/**` に当たって全件が外れる。除外そのものの検証は
	// TestGateConfigIsExcluded が実パスを組み立てて行う。
	cfg := &GateConfig{}

	t.Run("存在しないファイルは外す", func(t *testing.T) {
		targets, _ := selectTargets([]string{a, filepath.Join(dir, "gone.md")}, cfg, 0)
		if !slices.Equal(targets, []string{a}) {
			t.Errorf("targets = %v, want %v", targets, []string{a})
		}
	})

	t.Run("上限を超えた分は skipped に残す", func(t *testing.T) {
		targets, skipped := selectTargets([]string{a, b}, cfg, 1)
		if !slices.Equal(targets, []string{a}) {
			t.Errorf("targets = %v, want %v", targets, []string{a})
		}
		if !slices.Equal(skipped, []string{b}) {
			t.Errorf("skipped = %v, want %v", skipped, []string{b})
		}
	})

	t.Run("除外パターンに当たるファイルは外す", func(t *testing.T) {
		plan := writeTestFile(t, dir, "PLAN.md", "本文｡\n")
		excluded := &GateConfig{Exclude: []string{"**/PLAN.md"}}
		targets, _ := selectTargets([]string{a, plan}, excluded, 0)
		if !slices.Equal(targets, []string{a}) {
			t.Errorf("targets = %v, want %v", targets, []string{a})
		}
	})
}

func TestRunGate(t *testing.T) {
	dir := t.TempDir()

	// 既定の除外に頼らない。t.TempDir() は Linux では /tmp 配下に作られ、
	// 既定の `**/tmp/**` が全件を外してしまう。
	gateConfig := writeTestFile(t, dir, "gate-config.json", `{"exclude":["**/PLAN.md"]}`)

	run := func(t *testing.T, transcript string) hookOutput {
		t.Helper()
		stdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Pipe: %v", err)
		}
		os.Stdout = w
		runErr := runGate([]string{
			"--transcript", transcript,
			"--config", gateConfig,
			"--rules", filepath.Join(dir, "no-rules.json"),
		})
		_ = w.Close()
		os.Stdout = stdout
		if runErr != nil {
			t.Fatalf("runGate: %v", runErr)
		}

		var out hookOutput
		if err := json.NewDecoder(r).Decode(&out); err != nil {
			t.Fatalf("出力が JSON ではありません: %v", err)
		}
		return out
	}

	t.Run("違反があれば block する", func(t *testing.T) {
		bad := writeTestFile(t, dir, "bad.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "bad.jsonl", toolUseLine("Write", bad)+"\n")
		out := run(t, transcript)
		if out.Decision != "block" {
			t.Fatalf("Decision = %q, want block", out.Decision)
		}
		if !strings.Contains(out.Reason, "process-leak") {
			t.Errorf("reason にルール ID が無い: %s", out.Reason)
		}
		if !strings.Contains(out.Reason, "まるごと書き直す") {
			t.Errorf("reason に直し方の原則が無い: %s", out.Reason)
		}
	})

	t.Run("違反が無ければ通す", func(t *testing.T) {
		good := writeTestFile(t, dir, "good.md", "設定の原本は config/settings/ に置く｡\n")
		transcript := writeTestFile(t, dir, "good.jsonl", toolUseLine("Write", good)+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空", out.Decision)
		}
	})

	t.Run("warn だけなら通す", func(t *testing.T) {
		warn := writeTestFile(t, dir, "warn.md",
			"いわば設定の原本である｡\nたとえるなら金型に近い｡\n言ってみれば工場に近い｡\n")
		transcript := writeTestFile(t, dir, "warn.jsonl", toolUseLine("Write", warn)+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空", out.Decision)
		}
	})

	// issue #53: 既存の md に 1 行足しただけで、触っていない行の指摘で block されていた｡
	t.Run("Edit で書いていない既存行の指摘では block しない", func(t *testing.T) {
		existing := writeTestFile(t, dir, "existing.md",
			"# 配置\n\nご指示のとおり、設定を分割した｡\n\n拡張子は .md と .bpmn を置く｡\n")
		transcript := writeTestFile(t, dir, "existing.jsonl", toolUseInputLine(t, "Edit", map[string]any{
			"file_path":  existing,
			"old_string": "拡張子は .md を置く｡",
			"new_string": "拡張子は .md と .bpmn を置く｡",
		})+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空: %s", out.Decision, out.Reason)
		}
	})

	t.Run("Edit で書いた行の指摘は block する", func(t *testing.T) {
		edited := writeTestFile(t, dir, "edited.md",
			"# 配置\n\n設定の原本は config/ に置く｡\n\nご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "edited.jsonl", toolUseInputLine(t, "Edit", map[string]any{
			"file_path":  edited,
			"old_string": "設定を分割した｡",
			"new_string": "ご指示のとおり、設定を分割した｡",
		})+"\n")
		out := run(t, transcript)
		if out.Decision != "block" {
			t.Fatalf("Decision = %q, want block", out.Decision)
		}
		if !strings.Contains(out.Reason, "edited.md\n  error:5 [process-leak]") {
			t.Errorf("書いた行 (5 行目) の指摘が無い: %s", out.Reason)
		}
	})

	t.Run("MultiEdit で書いていない既存行の指摘では block しない", func(t *testing.T) {
		multi := writeTestFile(t, dir, "multi.md",
			"ご指示のとおり、設定を分割した｡\n\n原本は config/ に置く｡\n")
		transcript := writeTestFile(t, dir, "multi.jsonl", toolUseInputLine(t, "MultiEdit", map[string]any{
			"file_path": multi,
			"edits": []map[string]any{
				{"old_string": "原本は", "new_string": "原本は config/ に置く｡"},
			},
		})+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空: %s", out.Decision, out.Reason)
		}
	})

	t.Run("block の reason に無効化コメントの書式がある", func(t *testing.T) {
		bad := writeTestFile(t, dir, "bad2.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "bad2.jsonl", toolUseLine("Write", bad)+"\n")
		out := run(t, transcript)
		if !strings.Contains(out.Reason, "<!-- writing-gate-disable process-leak -->") {
			t.Errorf("reason に無効化コメントの書式が無い: %s", out.Reason)
		}
	})

	t.Run("除外対象しか編集していなければ通す", func(t *testing.T) {
		plan := writeTestFile(t, dir, "PLAN.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "plan.jsonl", toolUseLine("Write", plan)+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空", out.Decision)
		}
	})

	// issue #65: 直さないと決めた指摘で、ユーザーのメッセージごとに block され続けていた｡
	t.Run("前回の reason に載せた指摘は block しない", func(t *testing.T) {
		bad := writeTestFile(t, dir, "seen.md", "ご指示のとおり、設定を分割した｡\n")
		first := writeTestFile(t, dir, "seen1.jsonl", toolUseLine("Write", bad)+"\n")
		out := run(t, first)
		if out.Decision != "block" {
			t.Fatalf("Decision = %q, want block", out.Decision)
		}
		ids := regexp.MustCompile(`\(wg:[0-9a-f]{8}\)`).FindAllString(out.Reason, -1)
		if len(ids) != 1 {
			t.Fatalf("reason の指摘 ID = %v, want 1 件: %s", ids, out.Reason)
		}

		// hook が block すると、reason は user (isMeta) の発言として transcript に残る
		second := writeTestFile(t, dir, "seen2.jsonl",
			toolUseLine("Write", bad)+"\n"+userLine(t, "Stop hook feedback:\n"+out.Reason, true)+"\n")
		if out := run(t, second); out.Decision != "" {
			t.Errorf("Decision = %q, want 空: %s", out.Decision, out.Reason)
		}
	})

	t.Run("前回に載せていない指摘があれば block し、載せた指摘は reason から消す", func(t *testing.T) {
		bad := writeTestFile(t, dir, "seen-partial.md", "ご指示のとおり、設定を分割した｡\n")
		first := writeTestFile(t, dir, "seen-partial1.jsonl", toolUseLine("Write", bad)+"\n")
		out := run(t, first)
		if out.Decision != "block" {
			t.Fatalf("Decision = %q, want block", out.Decision)
		}

		// 新しい漏出を 1 行足した
		writeTestFile(t, dir, "seen-partial.md", "ご指示のとおり、設定を分割した｡\n\n当初の案では分割しなかった｡\n")
		second := writeTestFile(t, dir, "seen-partial2.jsonl",
			toolUseLine("Write", bad)+"\n"+userLine(t, "Stop hook feedback:\n"+out.Reason, true)+"\n")
		out = run(t, second)
		if out.Decision != "block" {
			t.Fatalf("Decision = %q, want block: %s", out.Decision, out.Reason)
		}
		if strings.Contains(out.Reason, "error:1 ") {
			t.Errorf("前回に載せた 1 行目の指摘が reason に残っている: %s", out.Reason)
		}
		if !strings.Contains(out.Reason, "error:3 ") {
			t.Errorf("新しい 3 行目の指摘が reason に無い: %s", out.Reason)
		}
	})

	t.Run("ユーザーが gate-off を打っていれば block しない", func(t *testing.T) {
		bad := writeTestFile(t, dir, "off.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "off.jsonl",
			userLine(t, gateOffCommand, false)+"\n"+toolUseLine("Write", bad)+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空: %s", out.Decision, out.Reason)
		}
	})

	t.Run("gate-on を打ち直していれば block する", func(t *testing.T) {
		bad := writeTestFile(t, dir, "on.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "on.jsonl",
			userLine(t, gateOffCommand, false)+"\n"+toolUseLine("Write", bad)+"\n"+userLine(t, gateOnCommand, false)+"\n")
		if out := run(t, transcript); out.Decision != "block" {
			t.Errorf("Decision = %q, want block", out.Decision)
		}
	})

	t.Run("Claude が Skill ツールで gate-off を呼んでも block する", func(t *testing.T) {
		bad := writeTestFile(t, dir, "skill-off.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "skill-off.jsonl",
			toolUseInputLine(t, "Skill", map[string]any{"skill": "japanese-writer:gate-off"})+"\n"+
				toolUseLine("Write", bad)+"\n")
		if out := run(t, transcript); out.Decision != "block" {
			t.Errorf("Decision = %q, want block", out.Decision)
		}
	})

	t.Run("block の reason に指摘 ID と gate-off の案内がある", func(t *testing.T) {
		bad := writeTestFile(t, dir, "guide.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "guide.jsonl", toolUseLine("Write", bad)+"\n")
		out := run(t, transcript)
		if !regexp.MustCompile(`\[process-leak\] .*\(wg:[0-9a-f]{8}\)\n`).MatchString(out.Reason) {
			t.Errorf("reason の指摘行に ID が無い: %s", out.Reason)
		}
		if !strings.Contains(out.Reason, "/japanese-writer:gate-off") {
			t.Errorf("reason に gate-off の案内が無い: %s", out.Reason)
		}
		if !strings.Contains(out.Reason, "/japanese-writer:gate-on") {
			t.Errorf("reason に gate-on の案内が無い: %s", out.Reason)
		}
	})

	// issue #65: 一意にするため old_string に含めた前後の行の既存の指摘で block されていた｡
	t.Run("old_string にそのまま含めた未変更行の指摘では block しない", func(t *testing.T) {
		existing := writeTestFile(t, dir, "neighbor.md",
			"ご指示のとおり、設定を分割した｡\n拡張子は .md と .bpmn を置く｡\n")
		transcript := writeTestFile(t, dir, "neighbor.jsonl", toolUseInputLine(t, "Edit", map[string]any{
			"file_path":  existing,
			"old_string": "ご指示のとおり、設定を分割した｡\n拡張子は .md を置く｡",
			"new_string": "ご指示のとおり、設定を分割した｡\n拡張子は .md と .bpmn を置く｡",
		})+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空: %s", out.Decision, out.Reason)
		}
	})
}
