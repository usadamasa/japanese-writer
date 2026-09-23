package jsonlscan

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ContentBlock は message.content[] の要素を表す｡
type ContentBlock struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// JSONLLine はセッション JSONL ファイルの1行を表す｡
type JSONLLine struct {
	Type    string `json:"type"`
	Message struct {
		Content []ContentBlock `json:"content"`
	} `json:"message"`
}

// WalkOptions は WalkJSONLFiles の走査オプション｡
type WalkOptions struct {
	Days int
}

// WalkJSONLFiles は指定ディレクトリの JSONL ファイルを走査し、
// 各ファイルに対してコールバック関数を呼び出す｡
// ディレクトリが存在しない場合はエラーなしで終了する｡
func WalkJSONLFiles(dir string, opts WalkOptions, fn func(path string) error) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	cutoff := time.Now().Add(-time.Duration(opts.Days) * 24 * time.Hour)

	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}

		return fn(path)
	})
}

// NewScanner は JSONL ファイル読み込み用の bufio.Scanner を作成する｡
// バッファサイズは最大10MBに設定される｡
func NewScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return s
}

// CountUniqueFiles はスキャン結果のユニークなファイル数をカウントする｡
func CountUniqueFiles[T any](results []T, getPath func(T) string) int {
	seen := make(map[string]bool)
	for _, r := range results {
		seen[getPath(r)] = true
	}
	return len(seen)
}
