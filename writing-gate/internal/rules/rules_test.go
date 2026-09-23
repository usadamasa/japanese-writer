package rules

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// writeTestFile はテスト用のファイルを作る｡
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// findRule は id でルールを引く｡
func findRule(rs *RuleSet, id string) *PhraseRule {
	for _, r := range rs.PhraseRules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func TestLoadRulesEmbedded(t *testing.T) {
	rs, err := LoadRules("")
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}

	ids := make([]string, 0, len(rs.PhraseRules))
	for _, r := range rs.PhraseRules {
		ids = append(ids, r.ID)
		if r.Severity == "" {
			t.Errorf("%s: severity が空", r.ID)
		}
		if r.Message == "" {
			t.Errorf("%s: message が空", r.ID)
		}
		if len(r.compiled) != len(r.Regexps) {
			t.Errorf("%s: 正規表現が compile されていない", r.ID)
		}
	}
	for _, want := range []string{"process-leak", "pink-elephant"} {
		if !slices.Contains(ids, want) {
			t.Errorf("phrase_rule %s が無い (ids=%v)", want, ids)
		}
	}
	if !rs.Aggregates.SentenceEndingRepeat.Enabled {
		t.Error("sentence_ending_repeat が無効になっている")
	}
}

func TestPhraseRuleMatches(t *testing.T) {
	rs, err := LoadRules("")
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}

	tests := []struct {
		name string
		id   string
		text string
		want bool
	}{
		{"部分一致で拾う", "process-leak", "ご指示のとおり、設定を分割した｡", true},
		{"正規表現で拾う", "pink-elephant", "以前は symlink だったが、今は不要になった｡", true},
		{"当たらない文は空", "process-leak", "設定の原本は config/settings/ に置く｡", false},
		{"句点の後に主題句だけ残す行を拾う", "wrap-orphan-topic", "案を並べてレビューを通してから送る｡agy は", true},
		{"句点で切った行は拾わない", "wrap-orphan-topic", "案を並べてレビューを通してから送る｡", false},
		{"inline code の除去跡は拾わない", "wrap-orphan-topic", "有効化はユーザーが手で行う｡ が", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := findRule(rs, tt.id)
			if rule == nil {
				t.Fatalf("ルール %s が無い", tt.id)
			}
			got := rule.Matches(tt.text)
			if (len(got) > 0) != tt.want {
				t.Errorf("Matches(%q) = %v, want hit=%v", tt.text, got, tt.want)
			}
		})
	}
}

func TestLoadRulesOverride(t *testing.T) {
	dir := t.TempDir()

	t.Run("既存 id へ追記する", func(t *testing.T) {
		path := writeTestFile(t, dir, "append.json",
			`{"phrase_rules":[{"id":"process-leak","patterns":["社内の合意により"]}]}`)
		rs, err := LoadRules(path)
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		if got := findRule(rs, "process-leak").Matches("社内の合意により手順を変えた｡"); len(got) == 0 {
			t.Error("追記したパターンが効いていない")
		}
		if got := findRule(rs, "process-leak").Matches("ご指示のとおり直した｡"); len(got) == 0 {
			t.Error("既存のパターンが消えている")
		}
	})

	t.Run("新しい id を足す", func(t *testing.T) {
		path := writeTestFile(t, dir, "new.json",
			`{"phrase_rules":[{"id":"team-jargon","severity":"warn","message":"社内語","patterns":["よしなに"]}]}`)
		rs, err := LoadRules(path)
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		rule := findRule(rs, "team-jargon")
		if rule == nil {
			t.Fatal("追加したルールが無い")
		}
		if len(rule.Matches("よしなに調整する｡")) == 0 {
			t.Error("追加したルールが効いていない")
		}
	})

	t.Run("message と severity は非空なら差し替える", func(t *testing.T) {
		path := writeTestFile(t, dir, "replace.json",
			`{"phrase_rules":[{"id":"process-leak","severity":"warn","message":"差し替えた"}]}`)
		rs, err := LoadRules(path)
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		rule := findRule(rs, "process-leak")
		if rule.Severity != "warn" || rule.Message != "差し替えた" {
			t.Errorf("severity=%q message=%q", rule.Severity, rule.Message)
		}
	})

	t.Run("集計ルールの閾値を差し替える", func(t *testing.T) {
		path := writeTestFile(t, dir, "aggr.json",
			`{"aggregates":{"metaphor_repeat":{"enabled":true,"severity":"error","min_count":1}}}`)
		rs, err := LoadRules(path)
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		if rs.Aggregates.MetaphorRepeat.MinCount != 1 {
			t.Errorf("MinCount = %d, want 1", rs.Aggregates.MetaphorRepeat.MinCount)
		}
		if !rs.Aggregates.SentenceEndingRepeat.Enabled {
			t.Error("指定していないブロックまで差し替わっている")
		}
	})

	t.Run("比喩の目印を追記する", func(t *testing.T) {
		path := writeTestFile(t, dir, "metaphor.json", `{"metaphor_markers":["まるで"]}`)
		rs, err := LoadRules(path)
		if err != nil {
			t.Fatalf("LoadRules: %v", err)
		}
		if !slices.Contains(rs.MetaphorMarkers, "まるで") {
			t.Error("追記した目印が入っていない")
		}
		if !slices.Contains(rs.MetaphorMarkers, "いわば") {
			t.Error("既定の目印が消えている")
		}
	})

	t.Run("id の無い要素はエラー", func(t *testing.T) {
		path := writeTestFile(t, dir, "noid.json", `{"phrase_rules":[{"patterns":["x"]}]}`)
		if _, err := LoadRules(path); err == nil {
			t.Error("エラーを返すはず")
		}
	})

	t.Run("不正な正規表現はエラー", func(t *testing.T) {
		path := writeTestFile(t, dir, "badre.json",
			`{"phrase_rules":[{"id":"x","severity":"warn","message":"m","regexps":["("]}]}`)
		if _, err := LoadRules(path); err == nil {
			t.Error("エラーを返すはず")
		}
	})

	t.Run("壊れた JSON はエラー", func(t *testing.T) {
		path := writeTestFile(t, dir, "broken.json", `{`)
		if _, err := LoadRules(path); err == nil {
			t.Error("エラーを返すはず")
		}
	})

	t.Run("ファイルが無くても既定で動く", func(t *testing.T) {
		if _, err := LoadRules(filepath.Join(dir, "none.json")); err != nil {
			t.Errorf("LoadRules: %v", err)
		}
	})
}
