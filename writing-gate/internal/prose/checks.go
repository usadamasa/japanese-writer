// Package prose は markdown の文章そのものを解析して点検する｡
// CLI の引数処理・transcript の読み取り・hook との受け渡しは呼び出し側が持つ｡
package prose

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/usadamasa/japanese-writer/writing-gate/internal/rules"
)

// Finding は 1 件の検出結果｡
type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Guidance string `json:"guidance,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
}

const (
	SeverityError = "error"
	SeverityWarn  = "warn"
)

// Check は解析済み文書へルールを当て、検出結果を行番号順で返す｡
func Check(doc *Document, rs *rules.RuleSet) []Finding {
	if doc.DisabledAll {
		return nil
	}

	var findings []Finding
	findings = append(findings, checkPhrases(doc, rs)...)
	findings = append(findings, checkSentenceEndingRepeat(doc, rs)...)
	findings = append(findings, checkStyleMix(doc, rs)...)
	findings = append(findings, checkMetaphorRepeat(doc, rs)...)

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	return findings
}

// checkPhrases は語・句レベルの漏出パターンを当てる｡見出し・リスト・表も対象にする｡
// 漏出は本文以外にも出るため｡引用行だけは他人の文なので外す｡
func checkPhrases(doc *Document, rs *rules.RuleSet) []Finding {
	var findings []Finding
	for _, rule := range rs.PhraseRules {
		if doc.IsDisabled(rule.ID) {
			continue
		}
		for _, ln := range doc.Lines {
			if ln.Kind == KindQuote {
				continue
			}
			text := cleanText(ln.Text)
			if text == "" {
				continue
			}
			for _, hit := range rule.Matches(text) {
				findings = append(findings, Finding{
					File: doc.Path, Line: ln.Num, RuleID: rule.ID,
					Severity: rule.Severity, Message: rule.Message,
					Guidance: rule.Guidance, Excerpt: hit,
				})
			}
		}
	}
	return findings
}

// checkSentenceEndingRepeat は同じ語尾の文が段落内で連続していないかを見る｡
// リスト項目は並列性が規範なので対象外｡
func checkSentenceEndingRepeat(doc *Document, rs *rules.RuleSet) []Finding {
	cfg := rs.Aggregates.SentenceEndingRepeat
	const ruleID = "sentence-ending-repeat"
	if !cfg.Enabled || doc.IsDisabled(ruleID) {
		return nil
	}

	var findings []Finding
	var runSuffix string
	var runStart int
	runLen := 0

	flush := func() {
		if runLen >= cfg.MinRun {
			findings = append(findings, Finding{
				File: doc.Path, Line: runStart, RuleID: ruleID, Severity: cfg.Severity,
				Message:  fmt.Sprintf("同じ語尾 (「%s」) の文が %d 文続いている｡", runSuffix, runLen),
				Guidance: "語尾だけを言い換えても単調さは残る｡述部の組み立てを変えるか、文をまとめる｡",
				Excerpt:  runSuffix,
			})
		}
		runLen = 0
		runSuffix = ""
	}

	prevBlock := -1
	for _, s := range doc.Sentences {
		if s.Kind != KindBody {
			flush()
			prevBlock = -1
			continue
		}
		if s.Block != prevBlock {
			flush()
			prevBlock = s.Block
		}

		suffix := endingSuffix(s.Text, cfg.SuffixRunes, cfg.MinSentenceRunes)
		if suffix == "" {
			flush()
			continue
		}
		if suffix == runSuffix {
			runLen++
			continue
		}
		flush()
		runSuffix = suffix
		runStart = s.Line
		runLen = 1
	}
	flush()
	return findings
}

// endingSuffix は文末の判定に使う末尾 n 文字を返す｡短すぎる文は判定しない｡
func endingSuffix(text string, n, minRunes int) string {
	if !hasJapanese(text) {
		return ""
	}
	trimmed := strings.TrimRight(text, "」』）)】〉>\"'　 ")
	r := []rune(trimmed)
	if len(r) < minRunes || len(r) < n {
		return ""
	}
	// 日本語の述部はひらがなで終わる｡そうでないものは語彙の列挙や記号の並びで、
	// 「同じ語尾が続いている」とは読まれない｡
	if !unicode.In(r[len(r)-1], unicode.Hiragana) {
		return ""
	}
	return string(r[len(r)-n:])
}

// hasJapanese は日本語の文字を含むかを返す｡
// 語尾と文体の判定は日本語にしか意味がないため、英文をふるい落とす｡
func hasJapanese(text string) bool {
	for _, r := range text {
		if unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) {
			return true
		}
	}
	return false
}

var (
	desumasuSuffixes = []string{"ませんでした", "ましょう", "でしょう", "ません", "ました", "でした", "です", "ます"}

	// plainEndRunes は である体の文末に立つ文字｡五段・一段の終止形、過去の「た」「だ」、
	// 形容詞の「い」を含む｡体言止めはどれにも当たらず、判定対象から外れる｡
	plainEndRunes = "うくぐすずつぬぶむるたいだ"

	// renyoukeiRunes は連用形の末尾に立つひらがな (い段・え段)｡
	// 丁寧語の「ます」は連用形に付くため、ひらがなでこれ以外が直前に来る形
	// (「だます」「かます」) は本動詞であって丁寧語ではない｡
	renyoukeiRunes = "いきしちにひみりぎじぢびぴえけせてねへめれげぜでべぺ"

	// honDoushiMasu は「ます」で終わるが丁寧語ではない本動詞｡
	// 一段動詞は連用形が送り仮名を持たないため、「出ます」「見ます」は漢字の
	// 直後に「ます」が付く｡「済ます」「励ます」と字面が同じで、直前の 1 文字では
	// 区別できない｡丁寧語の側は開いた集合なので、閉じているこちらを列挙して
	// 残りを丁寧語として扱う｡
	honDoushiMasu = []string{
		"済ます", "澄ます", "励ます", "冷ます", "覚ます", "醒ます", "悩ます", "眩ます",
	}
)

// checkStyleMix は である体とですます体の混在を文書単位で見る｡
// textlint の no-mix-dearu-desumasu は文ごとに指摘するが、ここでは少数派を数えて
// 1 件にまとめる｡完了ゲートで並べても読めるようにするため｡
func checkStyleMix(doc *Document, rs *rules.RuleSet) []Finding {
	cfg := rs.Aggregates.StyleMix
	const ruleID = "style-mix"
	if !cfg.Enabled || doc.IsDisabled(ruleID) {
		return nil
	}

	var desumasu, plain []Sentence
	for _, s := range doc.Sentences {
		if s.Kind == KindQuote || s.Kind == KindHeading || s.Kind == KindTable {
			continue
		}
		switch classifyStyle(s.Text) {
		case "desumasu":
			desumasu = append(desumasu, s)
		case "plain":
			plain = append(plain, s)
		}
	}

	total := len(desumasu) + len(plain)
	if total < cfg.MinSentences || len(desumasu) == 0 || len(plain) == 0 {
		return nil
	}

	minority, majorityName := desumasu, "である体"
	if len(plain) < len(desumasu) {
		minority, majorityName = plain, "ですます体"
	}

	limit := cfg.MaxExamples
	if limit <= 0 || limit > len(minority) {
		limit = len(minority)
	}

	findings := make([]Finding, 0, limit)
	for _, s := range minority[:limit] {
		findings = append(findings, Finding{
			File: doc.Path, Line: s.Line, RuleID: ruleID, Severity: cfg.Severity,
			Message: fmt.Sprintf("文体が混ざっている｡文書全体は %s が多数 (である体 %d 文 / ですます体 %d 文)｡",
				majorityName, len(plain), len(desumasu)),
			Guidance: "多数派へ寄せる｡語尾だけ差し替えず、文全体を書き直す｡",
			Excerpt:  s.Text,
		})
	}
	return findings
}

// classifyStyle は文末から文体を判定する｡体言止めなど判定できないものは空文字を返す｡
func classifyStyle(text string) string {
	if !hasJapanese(text) {
		return ""
	}
	trimmed := strings.TrimRight(text, "」』）)】〉>\"'　 ")
	r := []rune(trimmed)
	if len(r) == 0 {
		return ""
	}
	for _, suf := range desumasuSuffixes {
		if !strings.HasSuffix(trimmed, suf) {
			continue
		}
		if suf == "ます" && !isPoliteMasu(trimmed, r) {
			break
		}
		return "desumasu"
	}
	if strings.ContainsRune(plainEndRunes, r[len(r)-1]) {
		return "plain"
	}
	return ""
}

// isPoliteMasu は「ます」で終わる文の「ます」が丁寧語かを返す｡
// 直前がひらがななら連用形かどうかで決まる｡漢字・カタカナのときは
// 一段動詞の連用形 (「出ます」) と本動詞 (「済ます」) が同じ字面になるため、
// 列挙できる本動詞だけを外す｡
func isPoliteMasu(trimmed string, r []rune) bool {
	if len(r) < 3 {
		return false
	}

	prev := r[len(r)-3]
	if unicode.In(prev, unicode.Hiragana) {
		return strings.ContainsRune(renyoukeiRunes, prev)
	}

	for _, verb := range honDoushiMasu {
		if strings.HasSuffix(trimmed, verb) {
			return false
		}
	}
	return true
}

// checkMetaphorRepeat は比喩の目印が文書内で何度も出ていないかを見る｡
func checkMetaphorRepeat(doc *Document, rs *rules.RuleSet) []Finding {
	cfg := rs.Aggregates.MetaphorRepeat
	const ruleID = "metaphor-repeat"
	if !cfg.Enabled || doc.IsDisabled(ruleID) {
		return nil
	}

	count := 0
	first := 0
	for _, ln := range doc.Lines {
		if ln.Kind == KindQuote {
			continue
		}
		text := cleanText(ln.Text)
		for _, marker := range rs.MetaphorMarkers {
			if marker == "" {
				continue
			}
			n := strings.Count(text, marker)
			if n > 0 && first == 0 {
				first = ln.Num
			}
			count += n
		}
	}
	if count < cfg.MinCount {
		return nil
	}
	return []Finding{{
		File: doc.Path, Line: first, RuleID: ruleID, Severity: cfg.Severity,
		Message:  fmt.Sprintf("比喩の目印が %d 箇所ある｡同じ比喩を使い回していないか確認する｡", count),
		Guidance: "説明の主軸を比喩に預けている箇所は、対象そのものの記述へ置き換える｡",
	}}
}
