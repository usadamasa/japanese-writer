package main

import (
	"encoding/json"
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
	want := []string{"/repo/a.md", "/repo/b.md", "/repo/c.md"}
	if !slices.Equal(got, want) {
		t.Errorf("EditedMarkdown = %v, want %v", got, want)
	}
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

	t.Run("除外対象しか編集していなければ通す", func(t *testing.T) {
		plan := writeTestFile(t, dir, "PLAN.md", "ご指示のとおり、設定を分割した｡\n")
		transcript := writeTestFile(t, dir, "plan.jsonl", toolUseLine("Write", plan)+"\n")
		if out := run(t, transcript); out.Decision != "" {
			t.Errorf("Decision = %q, want 空", out.Decision)
		}
	})
}
