// Package rules は writing-gate が当てるルールの定義と読み込みを持つ｡
// 文章の解析には関わらない｡
package rules

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

//go:embed rules.json
var embeddedRules []byte

// PhraseRule は語・句レベルの検出ルール｡置換先を持たないため prh には載せられない｡
type PhraseRule struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity"`
	Message  string   `json:"message"`
	Guidance string   `json:"guidance"`
	Patterns []string `json:"patterns"`
	Regexps  []string `json:"regexps"`

	compiled []*regexp.Regexp
}

// AggregateRule は文書全体の集計ルールの設定｡
type AggregateRule struct {
	Enabled  bool   `json:"enabled"`
	Severity string `json:"severity"`

	// sentence_ending_repeat
	MinRun           int `json:"min_run"`
	SuffixRunes      int `json:"suffix_runes"`
	MinSentenceRunes int `json:"min_sentence_runes"`

	// style_mix
	MinSentences int `json:"min_sentences"`
	MaxExamples  int `json:"max_examples"`

	// metaphor_repeat
	MinCount int `json:"min_count"`
}

// Aggregates は集計ルールの束｡
type Aggregates struct {
	SentenceEndingRepeat AggregateRule `json:"sentence_ending_repeat"`
	StyleMix             AggregateRule `json:"style_mix"`
	MetaphorRepeat       AggregateRule `json:"metaphor_repeat"`
}

// RuleSet は writing-gate が適用するルール一式｡
type RuleSet struct {
	Version         int           `json:"version"`
	PhraseRules     []*PhraseRule `json:"phrase_rules"`
	MetaphorMarkers []string      `json:"metaphor_markers"`
	Aggregates      Aggregates    `json:"aggregates"`
}

// LoadRules は埋め込みルールを読み、overridePath があればマージする｡
//
// マージの規則:
//   - 同じ id の phrase_rule は patterns / regexps を追記し、message と guidance は
//     override 側が非空なら差し替える｡
//   - 未知の id は追加する｡
//   - metaphor_markers は追記する｡
//   - aggregates は override 側に現れたブロックだけ差し替える｡
func LoadRules(overridePath string) (*RuleSet, error) {
	var rs RuleSet
	if err := json.Unmarshal(embeddedRules, &rs); err != nil {
		return nil, fmt.Errorf("埋め込みルールの解析に失敗しました: %w", err)
	}

	if overridePath != "" {
		raw, err := os.ReadFile(overridePath) // #nosec G304 -- CLI ツール: パスは利用者指定
		switch {
		case err == nil:
			if err := mergeOverride(&rs, raw); err != nil {
				return nil, fmt.Errorf("%s: %w", overridePath, err)
			}
		case os.IsNotExist(err):
			// 追加ルールは任意｡無ければ埋め込みだけで動く｡
		default:
			return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", overridePath, err)
		}
	}

	if err := rs.compile(); err != nil {
		return nil, err
	}
	return &rs, nil
}

// overrideFile は追加ルールファイルの形｡aggregates はブロック単位で差し替えるため
// ポインタで受けて、指定の有無を区別する｡
type overrideFile struct {
	PhraseRules     []*PhraseRule `json:"phrase_rules"`
	MetaphorMarkers []string      `json:"metaphor_markers"`
	Aggregates      *struct {
		SentenceEndingRepeat *AggregateRule `json:"sentence_ending_repeat"`
		StyleMix             *AggregateRule `json:"style_mix"`
		MetaphorRepeat       *AggregateRule `json:"metaphor_repeat"`
	} `json:"aggregates"`
}

func mergeOverride(rs *RuleSet, raw []byte) error {
	var ov overrideFile
	if err := json.Unmarshal(raw, &ov); err != nil {
		return fmt.Errorf("追加ルールの解析に失敗しました: %w", err)
	}

	index := make(map[string]*PhraseRule, len(rs.PhraseRules))
	for _, r := range rs.PhraseRules {
		index[r.ID] = r
	}
	for _, add := range ov.PhraseRules {
		if add.ID == "" {
			return fmt.Errorf("phrase_rules に id のない要素があります")
		}
		base, ok := index[add.ID]
		if !ok {
			rs.PhraseRules = append(rs.PhraseRules, add)
			index[add.ID] = add
			continue
		}
		base.Patterns = append(base.Patterns, add.Patterns...)
		base.Regexps = append(base.Regexps, add.Regexps...)
		if add.Message != "" {
			base.Message = add.Message
		}
		if add.Guidance != "" {
			base.Guidance = add.Guidance
		}
		if add.Severity != "" {
			base.Severity = add.Severity
		}
	}

	rs.MetaphorMarkers = append(rs.MetaphorMarkers, ov.MetaphorMarkers...)

	if ov.Aggregates != nil {
		if a := ov.Aggregates.SentenceEndingRepeat; a != nil {
			rs.Aggregates.SentenceEndingRepeat = *a
		}
		if a := ov.Aggregates.StyleMix; a != nil {
			rs.Aggregates.StyleMix = *a
		}
		if a := ov.Aggregates.MetaphorRepeat; a != nil {
			rs.Aggregates.MetaphorRepeat = *a
		}
	}
	return nil
}

// Matches は text に当たったパターンを、部分一致・正規表現の順で全て返す｡
// compile 済みの正規表現をパッケージの外へ出さないため、判定はここに閉じる｡
func (r *PhraseRule) Matches(text string) []string {
	var hits []string
	for _, pat := range r.Patterns {
		if pat != "" && strings.Contains(text, pat) {
			hits = append(hits, pat)
		}
	}
	for _, re := range r.compiled {
		if hit := re.FindString(text); hit != "" {
			hits = append(hits, hit)
		}
	}
	return hits
}

func (rs *RuleSet) compile() error {
	for _, r := range rs.PhraseRules {
		r.compiled = r.compiled[:0]
		for _, expr := range r.Regexps {
			re, err := regexp.Compile(expr)
			if err != nil {
				return fmt.Errorf("phrase_rule %s の正規表現が不正です (%s): %w", r.ID, expr, err)
			}
			r.compiled = append(r.compiled, re)
		}
	}
	return nil
}
