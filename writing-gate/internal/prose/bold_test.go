package prose

// TestBoldProblemsRegressions の表は nanaism/yomiyasu の
// tests/fixtures/bold_regressions.json を Go の table test へ移したもの｡
//
// Copyright (c) 2026 nanaism
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"slices"
	"strings"
	"testing"
)

func TestBoldProblemsRegressions(t *testing.T) {
	// boundary はブロック境界の安全性だけを見るケース｡件数は問わず、
	// ブロックをまたぐ修正案を出さないことだけを確かめる｡
	tests := []struct {
		name     string
		text     string
		want     int
		lines    []int
		boundary bool
		note     string
	}{
		{name: "issue2", text: "**太字は1行目から始まり、\n2行目で閉じる。** 続きの文には**別の太字**もある。", want: 0, note: "Core reproduction: closing delimiter plus later opening on line 2 must not be paired together."},
		{name: "simple_two_lines", text: "**太字は行1から\n行2まで**。", want: 0, note: "Strong emphasis can span a soft line break inside one paragraph."},
		{name: "simple_three_lines", text: "前置き。**行1から\n行2を通って\n行3まで**。", want: 0, note: "Carry delimiters across all lines of the same paragraph."},
		{name: "crlf", text: "**行1\r\n行2** 続き**次**。", want: 0, note: "CRLF handling must preserve original source line numbers."},
		{name: "multiple_following_pairs", text: "**行1から\n行2まで** 続き**第2**、さらに**第3**。", want: 0, note: "An odd local delimiter count cannot shift later valid pairs."},
		{name: "two_multiline_spans", text: "**最初は\nここまで** 続き**次は\n最後まで**。", want: 0, note: "Two multiline spans in the same paragraph."},
		{name: "hard_line_break", text: "**行1  \n行2** 続き**次**。", want: 0, note: "Two spaces before newline are a hard break, not a paragraph boundary."},
		{name: "offset_lines", text: "# 見出し\n\n前置き。\n\n**行5から\n行6まで** 続き**次**。", want: 0, note: "Do not renumber findings by a stripped or joined block representation."},
		{name: "list_continuation", text: "- **同じ項目の行1\n  行2まで** 続き**次**。", want: 0, note: "Continuation lines of one list item paragraph may share emphasis."},
		{name: "ordered_list_continuation", text: "1. **同じ項目の行1\n   行2まで** 続き**次**。", want: 0, note: "Ordered-list marker width changes continuation indentation."},
		{name: "quote_continuation", text: "> **引用の行1\n> 行2まで** 続き**次**。", want: 0, note: "Strip container markers semantically, while keeping source offsets."},
		{name: "quote_lazy_continuation", text: "> **引用の行1\n行2まで** 続き**次**。", want: 0, note: "Lazy continuation remains in the same quoted paragraph."},
		{name: "list_quote_continuation", text: "- > **行1\n  > 行2まで** 続き**次**。", want: 0, note: "Nested list and quote container continuation."},
		{name: "escaped_delimiters", text: "\\*\\*字面だけ\\*\\* **本当の\n太字** 続き**次**。", want: 0, note: "Escaped asterisks must not enter the delimiter stack."},
		{name: "even_backslash_escape", text: "\\\\**本当の\n太字** 続き**次**。", want: 0, note: "Two backslashes escape each other; the following asterisks are active."},
		{name: "odd_backslash_escape", text: "\\\\\\*\\*字面だけ **本当の\n太字** 続き**次**。", want: 0, note: "Escape classification must count backslash parity."},
		{name: "inline_code_same_line", text: "`**コード**` **本当の\n太字** 続き**次**。", want: 0, note: "Inline code has priority over emphasis delimiters."},
		{name: "inline_code_multiline", text: "`**コードの行1\n行2**` 続き**太字**。", want: 0, note: "Inline code itself can span a newline; inner asterisks remain literal."},
		{name: "inline_code_double_tick", text: "``**コード`行1\n行2**`` 続き**太字**。", want: 0, note: "Match equal-length backtick runs across a soft break."},
		{name: "inline_html", text: "**行1 <span>補足</span>\n行2** 続き**次**。", want: 0, note: "Inline HTML is not a leaf-block boundary."},
		{name: "true_broken_open_close", text: "次は**「行1から\n行2まで」**を読む。", want: 1, lines: []int{1}, note: "Both Japanese brackets fail flanking; repair keeps brackets, newline and emphasized text."},
		{name: "true_broken_close", text: "**行1から\n行2まで。**続き。", want: 1, lines: []int{1}, note: "Punctuation before closing delimiter and following letter breaks strong emphasis; move punctuation outside."},
		{name: "broken_with_later_valid", text: "次は**「行1から\n行2まで」**を読む。続き**次**。", want: 1, lines: []int{1}, note: "Fix only the intended first span and retain the independently valid source pair **次**. The original broken delimiters accidentally render を読む。続き as strong; that incidental broken range is an observation, not a preservation obligation."},
		{name: "singleline_bracket_regression", text: "次は**「立場」**を読む。", want: 1, lines: []int{1}, note: "Existing single-line bracket relocation must remain supported."},
		{name: "singleline_punctuation_regression", text: "これは**必須です。**詳しくは下。", want: 1, lines: []int{1}, note: "Existing single-line punctuation relocation must remain supported."},
		{name: "paragraph_boundary", text: "**開始\n\n終了** 続き**次**。", boundary: true, note: "Blank line ends the paragraph; no repair may pair the first and second blocks."},
		{name: "heading_boundary", text: "**開始\n## 見出し\n終了** 続き**次**。", boundary: true, note: "ATX heading interrupts a paragraph; later valid strong must survive."},
		{name: "list_item_boundary", text: "- **開始\n- 終了** 続き**次**。", boundary: true, note: "Separate list items must never share a strong delimiter pair."},
		{name: "quote_boundary", text: "> **開始\n\n終了** 続き**次**。", boundary: true, note: "A new paragraph outside a quote cannot close emphasis inside the quote."},
		{name: "fence_boundary", text: "**開始\n```text\nコード ** 字面\n```\n終了** 続き**次**。", boundary: true, note: "A fenced code block is a block boundary and ignores contained asterisks."},
		{name: "tilde_fence_boundary", text: "**開始\n~~~text\nコード ** 字面\n~~~\n終了** 続き**次**。", boundary: true, note: "Tilde fences require the same protection as backtick fences."},
		{name: "indented_code_boundary", text: "**開始\n\n    コード ** 字面\n\n終了** 続き**次**。", boundary: true, note: "Indented code block follows a blank line; its asterisks are not active."},
		{name: "thematic_break_boundary", text: "**開始\n\n---\n\n終了** 続き**次**。", boundary: true, note: "Thematic break creates separate leaf blocks."},
		{name: "heading_then_multiline", text: "# **独立見出し**\n\n**本文の行1\n行2** 続き**次**。", want: 0, note: "Heading strong and body strong must use separate inline contexts."},
		{name: "broken_offset_lines", text: "# 見出し\n\n前置き。\n\n次は**「行5から\n行6まで」**を読む。", want: 1, lines: []int{5}, note: "Opening delimiter is on original line 5, not line 1 after block normalization."},
		{name: "broken_quote_multiline", text: "> 次は**「行1から\n> 行2まで」**を読む。", want: 1, lines: []int{1}, note: "Repair multiline brackets without removing or moving quote prefixes."},
		{name: "broken_list_multiline", text: "- 次は**「行1から\n  行2まで」**を読む。", want: 1, lines: []int{1}, note: "Repair multiline brackets without removing list marker or continuation indentation."},
		{name: "quote_start_destructive", text: "**「始まり\n> 終わり」**続き**次**。", boundary: true, note: "Unclosed marks in the first paragraph cannot pair with marks in the new blockquote."},
		{name: "setext_start_destructive", text: "**「始まり\n===\n終わり」**続き**次**。", boundary: true, note: "A Setext heading and following paragraph are separate inline blocks."},
		{name: "table_start_destructive", text: "**「始まり\n| 終わり」**続き**次**。 | 後 |\n| -- | -- |\n| 行 | 後 |", boundary: true, note: "A table header, cells and preceding text must not form a single emphasis pair."},
		{name: "quote_depth_destructive", text: "> **「始まり\n>> 終わり」**続き**次**。", boundary: true, note: "A nested quote creates a separate paragraph; depth changes cannot carry unmatched delimiters."},
		{name: "heading_inside_quote_destructive", text: "> **「始まり\n> ## 見出し\n> 終わり」**続き**次**。", boundary: true, note: "The blockquote contains different leaf blocks separated by a heading."},
		{name: "list_inside_quote_destructive", text: "> **「始まり\n> - 終わり」**続き**次**。", boundary: true, note: "A new list item inside a quote is a separate inline block."},
		{name: "quote_depth_spaced_marker_destructive", text: "> **「始まり\n>  > 終わり」**続き**次**。", boundary: true, note: "A nested quote marker may be preceded by up to three spaces inside its parent container."},
		{name: "table_without_outer_pipes_destructive", text: "**「始まり\n終わり」**続き**次**。 | 後\n-- | --\n行 | 後", boundary: true, note: "Outer pipes are optional in GFM tables; the separator creates a real block boundary."},
		{name: "issue2_comment_even_delimiters", text: "**太字は1行目から始まり、\n2行目で閉じる。** 続きの文では**次の太字が始まり、\n3行目で閉じる。**", want: 0, note: "Exact additional public Issue #2 example, issuecomment-5970836936, with two local delimiters on line 2."},
		{name: "inner_whitespace_1", text: "** 重要 **", want: 1, lines: []int{1}, note: "Both sides have inner whitespace."},
		{name: "inner_whitespace_2", text: "次は** 重要 **です", want: 1, lines: []int{1}, note: "Both sides have inner whitespace in sentence."},
		{name: "inner_whitespace_3", text: "次は**重要 **です", want: 1, lines: []int{1}, note: "Closing delimiter has leading inner whitespace."},
		{name: "inner_whitespace_4", text: "次は** 重要**です", want: 1, lines: []int{1}, note: "Opening delimiter has trailing inner whitespace."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boldProblems(tt.text)
			if tt.boundary {
				for _, p := range got {
					if p.Suggest != "" {
						t.Errorf("ブロックをまたぐ修正案を出している: %+v (%s)", p, tt.note)
					}
				}
				return
			}
			if len(got) != tt.want {
				t.Fatalf("件数 = %d, want %d: %+v (%s)", len(got), tt.want, got, tt.note)
			}
			if len(tt.lines) > 0 {
				lines := make([]int, 0, len(got))
				for _, p := range got {
					lines = append(lines, p.Line)
				}
				if !slices.Equal(lines, tt.lines) {
					t.Errorf("行番号 = %v, want %v", lines, tt.lines)
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
