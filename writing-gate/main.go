// writing-gate は markdown の文章品質を機械点検する CLI｡
//
// textlint が拾えない 2 種類だけを見る｡
//
//  1. 文書全体の集計 (語尾の連続・文体の混在・比喩の反復)
//  2. 置換先を持たない漏出フレーズ (制作過程の痕跡・却下案の痕跡)
//
// 語彙レベルの表記ゆれは prh (textlint) の担当で、ここでは扱わない｡
// Stop hook から起動するため npx を経由せず、単体で完結させる｡
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/usadamasa/japanese-writer/writing-gate/internal/prose"
	"github.com/usadamasa/japanese-writer/writing-gate/internal/rules"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "scan":
		err = runScan(os.Args[2:])
	case "gate":
		err = runGate(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		err = fmt.Errorf("不明なサブコマンド: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "writing-gate: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  writing-gate scan [--format json|text] [--rules PATH] FILE...
      指定した .md を点検し、検出結果を出力する｡

  writing-gate gate --transcript PATH [--config PATH] [--rules PATH]
      transcript から今セッションで編集された .md を拾って点検し、
      severity=error が残っていれば Stop hook 用の block JSON を stdout へ出す｡
`)
}

// defaultRulesPath は追加ルールの既定の置き場｡無くてもよい｡
func defaultRulesPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "writing-gate-rules.json")
}

// defaultConfigPath は除外設定の既定の置き場｡無くてもよい｡
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "writing-gate.json")
}

func runScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	format := fs.String("format", "text", "出力形式 (json | text)")
	rulesPath := fs.String("rules", defaultRulesPath(), "追加ルール JSON のパス")
	if err := fs.Parse(args); err != nil {
		return err
	}
	targets := fs.Args()
	if len(targets) == 0 {
		return fmt.Errorf("点検対象を 1 つ以上指定してください")
	}

	rs, err := rules.LoadRules(*rulesPath)
	if err != nil {
		return err
	}

	findings, err := scanFiles(targets, rs)
	if err != nil {
		return err
	}

	switch *format {
	case "json":
		out, err := json.MarshalIndent(map[string]any{
			"findings": findings,
			"errors":   countSeverity(findings, prose.SeverityError),
			"warnings": countSeverity(findings, prose.SeverityWarn),
		}, "", "  ")
		if err != nil {
			return err
		}
		out = append(out, '\n')
		if _, err := os.Stdout.Write(out); err != nil {
			return err
		}
	case "text":
		if len(findings) == 0 {
			fmt.Println("writing-gate: 指摘なし")
			return nil
		}
		fmt.Print(formatFindings(findings))
	default:
		return fmt.Errorf("不明な --format: %s", *format)
	}
	return nil
}

// scanFiles は対象ファイルを順に点検する｡読めないファイルはエラーにする｡
// 「点検できなかった」を「指摘なし」と取り違えると、ゲートが素通りするため｡
func scanFiles(targets []string, rs *rules.RuleSet) ([]prose.Finding, error) {
	var findings []prose.Finding
	for _, path := range targets {
		src, err := os.ReadFile(path) // #nosec G304 -- CLI ツール: パスは利用者指定
		if err != nil {
			return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", path, err)
		}
		doc := prose.ParseDocument(path, string(src))
		findings = append(findings, prose.Check(doc, rs)...)
	}
	return findings, nil
}

func countSeverity(findings []prose.Finding, severity string) int {
	n := 0
	for _, f := range findings {
		if f.Severity == severity {
			n++
		}
	}
	return n
}

// formatFindings は人間と agent の両方が読める整形をする｡
func formatFindings(findings []prose.Finding) string {
	var b strings.Builder
	var lastFile string
	for _, f := range findings {
		if f.File != lastFile {
			fmt.Fprintf(&b, "\n%s\n", f.File)
			lastFile = f.File
		}
		fmt.Fprintf(&b, "  %s:%d [%s] %s\n", f.Severity, f.Line, f.RuleID, f.Message)
		if f.Excerpt != "" {
			fmt.Fprintf(&b, "      該当: %s\n", truncate(f.Excerpt, 60))
		}
		if f.Guidance != "" {
			fmt.Fprintf(&b, "      直し方: %s\n", f.Guidance)
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
