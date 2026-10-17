package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/usadamasa/japanese-writer/internal/jsonlscan"
	"github.com/usadamasa/japanese-writer/writing-gate/internal/prose"
	"github.com/usadamasa/japanese-writer/writing-gate/internal/rules"
)

// defaultExcludes は既定の除外パターン｡
//
// 会話の記録そのものが成果物である面 (議事録・日誌・対話ログ) と、
// 生成物・作業用ファイルを外す｡sanitize-artifacts が「適用しない面」として
// 挙げているものに揃えてある｡
var defaultExcludes = []string{
	"**/tmp/**",
	"**/node_modules/**",
	"**/.git/**",
	"**/.claude/projects/**",
	"**/.claude/worktrees/**",
	"**/obsidian/**",
	"**/daily/**",
	"**/diary/**",
	"**/minutes/**",
	"**/PLAN.md",
	"**/CHANGELOG.md",
}

// GateConfig は除外設定｡
type GateConfig struct {
	// Exclude は既定の除外パターンを置き換える｡
	Exclude []string `json:"exclude"`
	// ExcludeExtra は既定へ追記する除外パターン｡
	ExcludeExtra []string `json:"exclude_extra"`
}

// LoadGateConfig は設定ファイルを読む｡無ければ既定値を返す｡
func LoadGateConfig(path string) (*GateConfig, error) {
	cfg := &GateConfig{Exclude: defaultExcludes}
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- CLI ツール: パスは利用者指定
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", path, err)
	}

	var loaded GateConfig
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, fmt.Errorf("%s の解析に失敗しました: %w", path, err)
	}
	if len(loaded.Exclude) > 0 {
		cfg.Exclude = loaded.Exclude
	}
	cfg.Exclude = append(cfg.Exclude, loaded.ExcludeExtra...)
	return cfg, nil
}

// IsExcluded はパスが除外対象かを返す｡
//
// filepath.Match は `**` を扱えないため、2 つの形だけを自前で解釈する｡
//   - `**/dir/**` … パスに `/dir/` を含む
//   - `**/name.md` … basename が一致する
//
// それ以外はそのまま filepath.Match へ渡す｡
func (c *GateConfig) IsExcluded(path string) bool {
	base := filepath.Base(path)
	for _, pat := range c.Exclude {
		switch {
		case strings.HasPrefix(pat, "**/") && strings.HasSuffix(pat, "/**"):
			if strings.Contains(path, "/"+strings.Trim(pat, "*/")+"/") {
				return true
			}
		case strings.HasPrefix(pat, "**/"):
			if ok, _ := filepath.Match(strings.TrimPrefix(pat, "**/"), base); ok {
				return true
			}
		default:
			if ok, _ := filepath.Match(pat, path); ok {
				return true
			}
		}
	}
	return false
}

// editInput は Write / Edit 系 tool_use の入力から、書いたファイルと本文を取る｡
type editInput struct {
	FilePath  string  `json:"file_path"`
	NewString *string `json:"new_string"`
	Edits     []struct {
		NewString string `json:"new_string"`
	} `json:"edits"`
}

// editTools は「文書を書いた」とみなすツール｡
var editTools = map[string]bool{
	"Write":        true,
	"Edit":         true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

// EditedFile はセッション中に書いた .md 1 件｡
type EditedFile struct {
	Path string
	// Fragments は Edit / MultiEdit の new_string｡
	Fragments []string
	// Whole はファイル全体を書いたことを表す｡Write のほか、書いた本文を
	// 取り出せないツールもこちらに倒す｡点検の漏れより、点検しすぎを選ぶ｡
	Whole bool
}

// add は tool_use 1 件分の書き込みを積む｡
func (e *EditedFile) add(tool string, in editInput) {
	switch {
	case tool == "Edit" && in.NewString != nil:
		e.Fragments = append(e.Fragments, *in.NewString)
	case tool == "MultiEdit" && len(in.Edits) > 0:
		for _, ed := range in.Edits {
			e.Fragments = append(e.Fragments, ed.NewString)
		}
	default:
		e.Whole = true
	}
}

// EditedMarkdown は transcript から編集された .md を重複なく拾う｡
// 出現順を保つので、報告の並びがセッションの作業順になる｡
func EditedMarkdown(transcriptPath string) ([]EditedFile, error) {
	f, err := os.Open(transcriptPath) // #nosec G304 -- CLI ツール: パスは hook 由来
	if err != nil {
		return nil, fmt.Errorf("transcript を開けません: %w", err)
	}
	defer func() { _ = f.Close() }()

	index := map[string]int{}
	var out []EditedFile
	scanner := jsonlscan.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry jsonlscan.JSONLLine
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		for _, block := range entry.Message.Content {
			if block.Type != "tool_use" || !editTools[block.Name] {
				continue
			}
			var in editInput
			if err := json.Unmarshal(block.Input, &in); err != nil {
				continue
			}
			if !strings.HasSuffix(in.FilePath, ".md") {
				continue
			}
			i, ok := index[in.FilePath]
			if !ok {
				i = len(out)
				index[in.FilePath] = i
				out = append(out, EditedFile{Path: in.FilePath})
			}
			out[i].add(block.Name, in)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("transcript の読み取りに失敗しました: %w", err)
	}
	return out, nil
}

// FocusLines は書いた断片が現在の本文に現れる行番号を返す｡Whole なら nil を返す｡
//
// 断片は今の本文から探し直す｡後の編集で消えた断片は数えず、行がずれても追える｡
// 同じ断片が複数箇所にあればすべてを数える (どれを書いたかは transcript から決まらない)｡
func FocusLines(src string, e EditedFile) map[int]bool {
	if e.Whole {
		return nil
	}
	lines := map[int]bool{}
	for _, frag := range e.Fragments {
		if frag == "" {
			continue
		}
		// 末尾の改行は次の行を書いたことにはならない｡
		span := strings.Count(strings.TrimSuffix(frag, "\n"), "\n")
		for off := 0; ; {
			i := strings.Index(src[off:], frag)
			if i < 0 {
				break
			}
			start := strings.Count(src[:off+i], "\n") + 1
			for n := start; n <= start+span; n++ {
				lines[n] = true
			}
			off += i + len(frag)
		}
	}
	return lines
}

// hookOutput は Stop hook が解釈する JSON｡
type hookOutput struct {
	Decision string `json:"decision,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func runGate(args []string) error {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	transcript := fs.String("transcript", "", "セッション transcript (JSONL) の絶対パス")
	configPath := fs.String("config", defaultConfigPath(), "除外設定 JSON のパス")
	rulesPath := fs.String("rules", defaultRulesPath(), "追加ルール JSON のパス")
	maxFiles := fs.Int("max-files", 20, "点検する .md の上限")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *transcript == "" {
		return fmt.Errorf("--transcript は必須です")
	}

	rs, err := rules.LoadRules(*rulesPath)
	if err != nil {
		return err
	}
	cfg, err := LoadGateConfig(*configPath)
	if err != nil {
		return err
	}

	edited, err := EditedMarkdown(*transcript)
	if err != nil {
		return err
	}

	byPath := map[string]EditedFile{}
	paths := make([]string, 0, len(edited))
	for _, e := range edited {
		byPath[e.Path] = e
		paths = append(paths, e.Path)
	}
	selected, skipped := selectTargets(paths, cfg, *maxFiles)
	if len(selected) == 0 {
		return emit(hookOutput{})
	}
	targets := make([]EditedFile, 0, len(selected))
	for _, p := range selected {
		targets = append(targets, byPath[p])
	}

	findings, err := scanFiles(targets, rs)
	if err != nil {
		return err
	}
	if countSeverity(findings, prose.SeverityError) == 0 {
		return emit(hookOutput{})
	}

	return emit(hookOutput{
		Decision: "block",
		Reason:   buildReason(findings, skipped),
	})
}

// selectTargets は除外と上限を当てる｡上限を超えた分は報告に残す｡
// 黙って切り落とすと「全部見た」と読めてしまうため｡
func selectTargets(edited []string, cfg *GateConfig, maxFiles int) (targets, skipped []string) {
	for _, path := range edited {
		if cfg.IsExcluded(path) {
			continue
		}
		if st, err := os.Stat(path); err != nil || st.IsDir() {
			continue
		}
		targets = append(targets, path)
	}
	if maxFiles > 0 && len(targets) > maxFiles {
		skipped = targets[maxFiles:]
		targets = targets[:maxFiles]
	}
	return targets, skipped
}

func buildReason(findings []prose.Finding, skipped []string) string {
	var b strings.Builder
	b.WriteString("このセッションで書いた markdown に、機械点検で拾える不備が残っている｡\n")
	b.WriteString("該当箇所を直してから完了する｡\n")
	b.WriteString("\n直し方の原則: 検出語だけを別の語へ置き換えない｡" +
		"同じ問題が形を変えて残るため、検出語を含む文をまるごと書き直す｡\n")
	b.WriteString(formatFindings(findings))

	if len(skipped) > 0 {
		sorted := append([]string(nil), skipped...)
		sort.Strings(sorted)
		fmt.Fprintf(&b, "\n未点検 (上限超過) %d 件: %s\n", len(sorted), strings.Join(sorted, ", "))
	}
	b.WriteString("\n語彙・表記の点検はここでは行っていない｡" +
		"外部へ出す文書なら proofread skill を続けて呼ぶ｡\n")
	b.WriteString("\nこの指摘が面に合わない場合 (議事録・対話ログなど) は、" +
		"該当ファイルの先頭に無効化コメントを置くか ~/.claude/writing-gate.json の exclude_extra に足す｡\n")
	fmt.Fprintf(&b, "無効化コメントの書式: <!-- writing-gate-disable %s -->"+
		" (ルール ID を省くと全ルールが無効になる)\n", strings.Join(errorRuleIDs(findings), " "))
	return b.String()
}

// errorRuleIDs は block の原因になったルール ID を重複なく出現順で返す｡
func errorRuleIDs(findings []prose.Finding) []string {
	var ids []string
	for _, f := range findings {
		if f.Severity == prose.SeverityError && !slices.Contains(ids, f.RuleID) {
			ids = append(ids, f.RuleID)
		}
	}
	return ids
}

func emit(out hookOutput) error {
	buf, err := json.Marshal(out)
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	_, err = os.Stdout.Write(buf)
	return err
}
