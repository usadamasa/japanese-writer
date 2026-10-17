package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
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

	got, err := EditedMarkdown(transcript)
	if err != nil {
		t.Fatalf("EditedMarkdown: %v", err)
	}
	paths := make([]string, 0, len(got))
	for _, e := range got {
		paths = append(paths, e.Path)
	}
	want := []string{"/repo/a.md", "/repo/b.md", "/repo/c.md"}
	if !slices.Equal(paths, want) {
		t.Errorf("EditedMarkdown のパス = %v, want %v", paths, want)
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

	got, err := EditedMarkdown(transcript)
	if err != nil {
		t.Fatalf("EditedMarkdown: %v", err)
	}
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

func TestEditedMarkdownMissingFile(t *testing.T) {
	if _, err := EditedMarkdown(filepath.Join(t.TempDir(), "none.jsonl")); err == nil {
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
}
