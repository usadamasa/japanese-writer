package prose

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// boldCase は testdata/bold_regressions.json の 1 件｡
// expected_bold_problems が null のケースは境界の安全性だけを見る (件数は問わない)｡
type boldCase struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Expected *int   `json:"expected_bold_problems"`
	Lines    []int  `json:"expected_line_numbers"`
	Note     string `json:"note"`
}

func TestBoldProblemsRegressions(t *testing.T) {
	raw, err := os.ReadFile("testdata/bold_regressions.json")
	if err != nil {
		t.Fatalf("fixture を読めない: %v", err)
	}
	var cases []boldCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("fixture の解析に失敗: %v", err)
	}
	if len(cases) != 50 {
		t.Fatalf("fixture の件数 = %d, want 50", len(cases))
	}

	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			got := boldProblems(c.Text)
			if c.Expected == nil {
				for _, p := range got {
					if p.Suggest != "" {
						t.Errorf("ブロックをまたぐ修正案を出している: %+v (%s)", p, c.Note)
					}
				}
				return
			}
			if len(got) != *c.Expected {
				t.Fatalf("件数 = %d, want %d: %+v (%s)", len(got), *c.Expected, got, c.Note)
			}
			if *c.Expected > 0 && len(c.Lines) > 0 {
				lines := make([]int, 0, len(got))
				for _, p := range got {
					lines = append(lines, p.Line)
				}
				if !slices.Equal(lines, c.Lines) {
					t.Errorf("行番号 = %v, want %v", lines, c.Lines)
				}
			}
		})
	}
}

func TestBoldProblemsFix(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		how     string
		suggest string // 部分一致
		exact   bool
	}{
		{"かっこの内側だけを太字にする", "次に**「文書の立場」**を決めます。",
			"かっこの内側だけを太字にする", "「**文書の立場**」", false},
		{"複数行のかっこ", "次に**「太字は1行目から始まり、\n2行目で閉じる。」**を決めます。",
			"かっこの内側だけを太字にする", "「**太字は1行目から始まり、\n2行目で閉じる。**」", false},
		{"句点を外へ出す", "これは**必須です。**詳しくは下に書きます。",
			"句読点を太字の外に出す", "**必須です**。", false},
		{"半角句点を外へ出す", "これは**必須です｡**詳しくは下に書きます｡",
			"句読点を太字の外に出す", "**必須です**｡", false},
		{"半角読点を外へ出す", "これは**必須で､**詳しくは下に書きます｡",
			"句読点を太字の外に出す", "**必須で**､", false},
		{"半角スペースを入れる", "立場は**「勧め」か「決まり」**で決めます。",
			"文字に接する側に半角スペースを入れる", " **「勧め」か「決まり」** ", false},
		{"内側の空白 (両側)", "** 重要 **", "太字の内側の空白を取る", "**重要**", true},
		{"内側の空白 (閉じ側)", "次は**重要 **です", "太字の内側の空白を取る", "次は**重要**です", true},
		{"内側の空白 (開き側)", "次は** 重要**です", "太字の内側の空白を取る", "次は**重要**です", true},
		{"内側の空白 (複数行)", "** 重要\n重要 **", "太字の内側の空白を取る", "**重要\n重要**", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boldProblems(tt.text)
			if len(got) != 1 {
				t.Fatalf("件数 = %d, want 1: %+v", len(got), got)
			}
			p := got[0]
			if p.How != tt.how {
				t.Errorf("How = %q, want %q", p.How, tt.how)
			}
			if tt.exact && p.Suggest != tt.suggest {
				t.Errorf("Suggest = %q, want %q", p.Suggest, tt.suggest)
			}
			if !tt.exact && !strings.Contains(p.Suggest, tt.suggest) {
				t.Errorf("Suggest = %q, want contains %q", p.Suggest, tt.suggest)
			}
		})
	}
}

func TestBoldProblemsLineNumber(t *testing.T) {
	text := "1行目\n2行目\n3行目で**「太字が始まり、\n4行目で閉じる。」**のを決めます。"
	got := boldProblems(text)
	if len(got) != 1 || got[0].Line != 3 {
		t.Fatalf("got %+v, want 1 件 (3 行目)", got)
	}
}

func TestBoldProblemsSkips(t *testing.T) {
	for _, text := range []string{
		"---\ntitle: **「x」**です\n---\n本文",
		"```\n次に**「立場」**を決める\n```",
		"`次に**「立場」**を決める`",
		"<p>次に**「立場」**を決める</p>",
		`\*\*これは太字ではない\*\*`,
		"**先行** 通常テキスト **2行に\nまたがる太字** 通常テキスト **後続**",
	} {
		if got := boldProblems(text); len(got) != 0 {
			t.Errorf("boldProblems(%q) = %+v, want none", text, got)
		}
	}
}

func TestCheckBoldNotRendered(t *testing.T) {
	rs := loadTestRules(t)

	t.Run("表示されない太字を拾う", func(t *testing.T) {
		doc := ParseDocument("a.md", "次に**「文書の立場」**を決める｡\n")
		findings := Check(doc, rs)
		var hit *Finding
		for i := range findings {
			if findings[i].RuleID == "bold-not-rendered" {
				hit = &findings[i]
			}
		}
		if hit == nil {
			t.Fatalf("bold-not-rendered が無い: %v", findingIDs(findings))
		}
		if hit.Line != 1 || hit.Severity != rs.Aggregates.BoldNotRendered.Severity {
			t.Errorf("Line=%d Severity=%q", hit.Line, hit.Severity)
		}
		if !strings.Contains(hit.Guidance, "「**文書の立場**」") {
			t.Errorf("Guidance に修正案が無い: %q", hit.Guidance)
		}
	})

	t.Run("複数行の太字は改行を記号へ置き換えて出す", func(t *testing.T) {
		doc := ParseDocument("a.md", "次は**「行1から\n行2まで」**を読む｡\n")
		for _, f := range Check(doc, rs) {
			if f.RuleID != "bold-not-rendered" {
				continue
			}
			if strings.Contains(f.Excerpt, "\n") || strings.Contains(f.Guidance, "\n") {
				t.Errorf("改行が残っている: excerpt=%q guidance=%q", f.Excerpt, f.Guidance)
			}
			return
		}
		t.Fatal("bold-not-rendered が無い")
	})

	t.Run("無効化マーカーで止まる", func(t *testing.T) {
		doc := ParseDocument("a.md", "<!-- writing-gate-disable bold-not-rendered -->\n次に**「文書の立場」**を決める｡\n")
		if hasRule(Check(doc, rs), "bold-not-rendered") {
			t.Error("無効化したのに拾っている")
		}
	})

	t.Run("正しい太字は拾わない", func(t *testing.T) {
		doc := ParseDocument("a.md", "次に「**文書の立場**」を決める｡\n")
		if hasRule(Check(doc, rs), "bold-not-rendered") {
			t.Error("正しい太字を拾っている")
		}
	})
}
