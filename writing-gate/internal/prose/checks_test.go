package prose

import (
	"strings"
	"testing"

	"github.com/usadamasa/japanese-writer/writing-gate/internal/rules"
)

// loadTestRules は埋め込みルールだけを読み込む｡
func loadTestRules(t *testing.T) *rules.RuleSet {
	t.Helper()
	rs, err := rules.LoadRules("")
	if err != nil {
		t.Fatalf("LoadRules: %v", err)
	}
	return rs
}

// findingIDs は検出結果のルール ID を出現順に返す｡
func findingIDs(findings []Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.RuleID)
	}
	return ids
}

func hasRule(findings []Finding, ruleID string) bool {
	for _, f := range findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}

func TestCheckPhrases(t *testing.T) {
	rs := loadTestRules(t)

	tests := []struct {
		name   string
		src    string
		wantID string
		want   bool
	}{
		{
			name:   "制作過程の痕跡を拾う",
			src:    "ご指示のとおり、設定を分割した｡\n",
			wantID: "process-leak",
			want:   true,
		},
		{
			name:   "節の追加理由の説明を拾う",
			src:    "この節は読み手の理解を助けるために追加した｡\n",
			wantID: "process-leak",
			want:   true,
		},
		{
			name:   "方針転換の経緯注記を拾う",
			src:    "以前は symlink だったが、現在は不要になった｡\n",
			wantID: "pink-elephant",
			want:   true,
		},
		{
			name:   "却下案の供養を拾う",
			src:    "この手順では Homebrew は採用していません｡\n",
			wantID: "pink-elephant",
			want:   true,
		},
		{
			name:   "コードブロック内は対象外",
			src:    "説明の文｡\n\n```\nご指示のとおり\n```\n",
			wantID: "process-leak",
			want:   false,
		},
		{
			name:   "インラインコード内は対象外",
			src:    "`ご指示のとおり` という文字列を検出する｡\n",
			wantID: "process-leak",
			want:   false,
		},
		{
			name:   "引用行は対象外",
			src:    "> ご指示のとおり対応しました\n",
			wantID: "process-leak",
			want:   false,
		},
		{
			name:   "無関係な文は拾わない",
			src:    "設定の原本は config/settings/ に置く｡\n",
			wantID: "process-leak",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(ParseDocument("t.md", tt.src), rs)
			if hasRule(got, tt.wantID) != tt.want {
				t.Errorf("hasRule(%s) = %v, want %v (findings=%v)",
					tt.wantID, !tt.want, tt.want, findingIDs(got))
			}
		})
	}
}

func TestDisableMarker(t *testing.T) {
	rs := loadTestRules(t)

	t.Run("全ルールを無効化する", func(t *testing.T) {
		src := "<!-- writing-gate-disable -->\n\nご指示のとおり対応した｡\n"
		if got := Check(ParseDocument("t.md", src), rs); len(got) != 0 {
			t.Errorf("検出 0 件のはずが %v", findingIDs(got))
		}
	})

	t.Run("ルールを指定して無効化する", func(t *testing.T) {
		src := "<!-- writing-gate-disable pink-elephant -->\n\n" +
			"以前は symlink だったが、現在は不要になった｡\nご指示のとおり対応した｡\n"
		got := Check(ParseDocument("t.md", src), rs)
		if hasRule(got, "pink-elephant") {
			t.Errorf("pink-elephant は無効化されているはず: %v", findingIDs(got))
		}
		if !hasRule(got, "process-leak") {
			t.Errorf("process-leak は生きているはず: %v", findingIDs(got))
		}
	})
}

func TestCheckSentenceEndingRepeat(t *testing.T) {
	rs := loadTestRules(t)

	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "同じ語尾が 3 文続いたら拾う",
			src: "設定は原本から生成している｡\n" +
				"生成物はコミットしている｡\n" +
				"検証は pre-commit で実施している｡\n",
			want: true,
		},
		{
			name: "2 文なら拾わない",
			src: "設定は原本から生成している｡\n" +
				"生成物はコミットしている｡\n",
			want: false,
		},
		{
			name: "段落が変われば連続とみなさない",
			src: "設定は原本から生成している｡\n生成物はコミットしている｡\n\n" +
				"検証は pre-commit で実施している｡\n",
			want: false,
		},
		{
			name: "箇条書きの並列は拾わない",
			src: "- 設定は原本から生成している\n" +
				"- 生成物はコミットしている\n" +
				"- 検証は pre-commit で実施している\n",
			want: false,
		},
		{
			name: "ひらがなで終わらない行の並びは拾わない",
			src: "えらい(大変) / おそがい(恐ろしい) /\n" +
				"いごく(動く) / ちゃっと(すぐ) /\n" +
				"やぐい(出来が悪い) / こすい(ずるい) /\n",
			want: false,
		},
		{
			name: "英文が続いても拾わない",
			src: "This is the first line of the note\n" +
				"That was the second line of the note\n" +
				"Here comes the third line of the note\n",
			want: false,
		},
		{
			name: "語尾が違えば拾わない",
			src: "設定は原本から生成する｡\n" +
				"生成物はコミットした｡\n" +
				"検証は pre-commit が担う｡\n",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(ParseDocument("t.md", tt.src), rs)
			if hasRule(got, "sentence-ending-repeat") != tt.want {
				t.Errorf("sentence-ending-repeat = %v, want %v (findings=%v)",
					!tt.want, tt.want, findingIDs(got))
			}
		})
	}
}

func TestClassifyStyle(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"設定を生成する", "plain"},
		{"設定を生成した", "plain"},
		{"設定は要らない", "plain"},
		{"設定を生成します", "desumasu"},
		{"設定を生成しました", "desumasu"},
		{"これは設定です", "desumasu"},
		{"確認だけで済ます", "plain"}, // 本動詞の「ます」は丁寧語ではない
		{"部下を励ます", "plain"},   // 同上
		{"湯を冷ます", "plain"},    // 同上
		{"人を だます", "plain"},   // ひらがなでも連用形でなければ本動詞
		{"設定を書き換えます", "desumasu"},
		// 一段動詞は連用形に送り仮名が無く、漢字の直後に「ます」が付く｡
		{"オプションの一覧は --help に出ます", "desumasu"},
		{"ここに結果を見ます", "desumasu"},
		{"あとで通知が来ます", "desumasu"},
		{"生成物は build 配下", ""}, // 体言止めは判定しない
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := classifyStyle(tt.text); got != tt.want {
				t.Errorf("classifyStyle(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestCheckStyleMix(t *testing.T) {
	rs := loadTestRules(t)

	t.Run("少数派を指摘する", func(t *testing.T) {
		src := "設定を生成する｡\n原本を編集した｡\n検証は pre-commit が担う｡\n" +
			"生成物はコミットした｡\nこの節は補足です｡\n"
		got := Check(ParseDocument("t.md", src), rs)
		if !hasRule(got, "style-mix") {
			t.Fatalf("style-mix を検出するはず: %v", findingIDs(got))
		}
		for _, f := range got {
			if f.RuleID == "style-mix" && !strings.Contains(f.Excerpt, "補足") {
				t.Errorf("少数派の文を指すはず: %q", f.Excerpt)
			}
		}
	})

	t.Run("統一されていれば拾わない", func(t *testing.T) {
		src := "設定を生成する｡\n原本を編集した｡\n検証は pre-commit が担う｡\n" +
			"生成物はコミットした｡\nこの節は補足である｡\n"
		if got := Check(ParseDocument("t.md", src), rs); hasRule(got, "style-mix") {
			t.Errorf("style-mix は出ないはず: %v", findingIDs(got))
		}
	})

	t.Run("文数が少なければ拾わない", func(t *testing.T) {
		src := "設定を生成する｡\nこの節は補足です｡\n"
		if got := Check(ParseDocument("t.md", src), rs); hasRule(got, "style-mix") {
			t.Errorf("style-mix は出ないはず: %v", findingIDs(got))
		}
	})
}

func TestCheckMetaphorRepeat(t *testing.T) {
	rs := loadTestRules(t)

	t.Run("目印が閾値を超えたら拾う", func(t *testing.T) {
		src := "いわば設定の原本である｡\nたとえるなら金型に近い｡\n言ってみれば工場のようなものだ｡\n"
		got := Check(ParseDocument("t.md", src), rs)
		if !hasRule(got, "metaphor-repeat") {
			t.Errorf("metaphor-repeat を検出するはず: %v", findingIDs(got))
		}
		for _, f := range got {
			if f.RuleID == "metaphor-repeat" && f.Severity != SeverityWarn {
				t.Errorf("severity = %q, want %q", f.Severity, SeverityWarn)
			}
		}
	})

	t.Run("1 箇所なら拾わない", func(t *testing.T) {
		src := "いわば設定の原本である｡\n"
		if got := Check(ParseDocument("t.md", src), rs); hasRule(got, "metaphor-repeat") {
			t.Errorf("metaphor-repeat は出ないはず: %v", findingIDs(got))
		}
	})
}
