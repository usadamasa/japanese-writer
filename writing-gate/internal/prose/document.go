package prose

import (
	"regexp"
	"strings"
)

// LineKind は行の種別｡集計ルールを当てる範囲を決める｡
type LineKind int

const (
	// KindBody は通常の段落行｡
	KindBody LineKind = iota
	// KindHeading は見出し行｡
	KindHeading
	// KindList は箇条書き・番号付きリストの項目｡語尾の並列は規範上むしろ推奨されるため、
	// 語尾の連続判定からは外す｡
	KindList
	// KindTable は表の行｡
	KindTable
	// KindQuote は引用行｡他人の文なので文体判定から外す｡
	KindQuote
)

// Line は解析対象の 1 行｡コードブロックとフロントマターは除去済み｡
type Line struct {
	Num  int
	Text string
	Kind LineKind
}

// Sentence は 1 文｡行番号は文の開始行を指す｡
type Sentence struct {
	Line  int
	Text  string
	Kind  LineKind
	Block int // 同じ段落 (空行と非本文行で区切る) に属する文は同じ番号を持つ
}

// Document は markdown を解析した結果｡
type Document struct {
	Path      string
	Lines     []Line
	Sentences []Sentence
	// DisabledRules は `<!-- writing-gate-disable [rule-id ...] -->` で無効化された
	// ルール ID の集合｡ID を伴わない場合は全ルールを無効化するため All が true になる｡
	DisabledRules map[string]bool
	DisabledAll   bool
}

var (
	fenceRe      = regexp.MustCompile("^\\s{0,3}(```|~~~)")
	headingRe    = regexp.MustCompile(`^\s{0,3}#{1,6}\s`)
	listRe       = regexp.MustCompile(`^\s*([-*+]|\d+[.)])\s`)
	quoteRe      = regexp.MustCompile(`^\s{0,3}>`)
	inlineCodeRe = regexp.MustCompile("`[^`]*`")
	htmlTagRe    = regexp.MustCompile(`<[^>]+>`)
	linkURLRe    = regexp.MustCompile(`\]\([^)]*\)`)
	commentRe    = regexp.MustCompile(`(?s)<!--(.*?)-->`)
	disableRe    = regexp.MustCompile(`writing-gate-disable([^\n]*)`)
	sentenceSep  = regexp.MustCompile(`[｡。．]`)
)

// ParseDocument は markdown 本文を解析する｡
// コードブロック・フロントマター・インラインコード・HTML コメントは除去する｡
// これらを残すと、サンプルコードやルールの説明そのものが検出対象になる｡
func ParseDocument(path, src string) *Document {
	doc := &Document{Path: path, DisabledRules: map[string]bool{}}
	doc.readDisableMarkers(src)

	raw := strings.Split(src, "\n")
	start := skipFrontMatter(raw)

	inFence := false
	for i := start; i < len(raw); i++ {
		text := raw[i]
		if fenceRe.MatchString(text) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		doc.Lines = append(doc.Lines, Line{Num: i + 1, Text: text, Kind: classify(text)})
	}

	doc.Sentences = extractSentences(doc.Lines)
	return doc
}

// readDisableMarkers は HTML コメント中の無効化指定を読む｡
func (d *Document) readDisableMarkers(src string) {
	for _, m := range commentRe.FindAllStringSubmatch(src, -1) {
		body := m[1]
		hit := disableRe.FindStringSubmatch(body)
		if hit == nil {
			continue
		}
		ids := strings.Fields(hit[1])
		if len(ids) == 0 {
			d.DisabledAll = true
			continue
		}
		for _, id := range ids {
			d.DisabledRules[id] = true
		}
	}
}

// IsDisabled は指定ルールが無効化されているかを返す｡
func (d *Document) IsDisabled(ruleID string) bool {
	return d.DisabledAll || d.DisabledRules[ruleID]
}

func skipFrontMatter(raw []string) int {
	if len(raw) == 0 || strings.TrimSpace(raw[0]) != "---" {
		return 0
	}
	for i := 1; i < len(raw); i++ {
		if strings.TrimSpace(raw[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

func classify(text string) LineKind {
	switch {
	case headingRe.MatchString(text):
		return KindHeading
	case quoteRe.MatchString(text):
		return KindQuote
	case listRe.MatchString(text):
		return KindList
	case strings.HasPrefix(strings.TrimSpace(text), "|"):
		return KindTable
	default:
		return KindBody
	}
}

// cleanText は文の判定に使わない要素を落とす｡
func cleanText(text string) string {
	text = inlineCodeRe.ReplaceAllString(text, "")
	text = linkURLRe.ReplaceAllString(text, "]")
	text = htmlTagRe.ReplaceAllString(text, "")
	text = listRe.ReplaceAllString(text, "")
	text = headingRe.ReplaceAllString(text, "")
	text = strings.TrimPrefix(strings.TrimSpace(text), ">")
	return strings.TrimSpace(text)
}

// extractSentences は行を文へ割る｡
//
// 段落 (Block) は空行と非本文行で区切る｡語尾の連続はこの単位で見る｡
// 見出しをまたいだ「連続」は読み手に連続として届かないため｡
func extractSentences(lines []Line) []Sentence {
	var out []Sentence
	block := 0
	prevKind := KindBody
	prevBlank := true

	for _, ln := range lines {
		if strings.TrimSpace(ln.Text) == "" {
			if !prevBlank {
				block++
			}
			prevBlank = true
			continue
		}
		if ln.Kind != prevKind && !prevBlank {
			block++
		}
		prevBlank = false
		prevKind = ln.Kind

		body := cleanText(ln.Text)
		if body == "" {
			continue
		}
		for _, s := range splitSentences(body) {
			out = append(out, Sentence{Line: ln.Num, Text: s, Kind: ln.Kind, Block: block})
		}
	}
	return out
}

// splitSentences は句点で文へ割る｡句点で終わらない末尾の断片も 1 文として扱う｡
// 一文改行で書かれた行は句点を持たないことがあるため｡
func splitSentences(text string) []string {
	parts := sentenceSep.Split(text, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
