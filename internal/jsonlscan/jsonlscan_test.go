package jsonlscan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWalkJSONLFiles(t *testing.T) {
	t.Run("存在しないディレクトリはエラーなしで終了", func(t *testing.T) {
		err := WalkJSONLFiles("/nonexistent/dir", WalkOptions{Days: 30}, func(path string) error {
			t.Error("コールバックが呼ばれてはいけない")
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("JSONL ファイルのみコールバックが呼ばれる", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "test.jsonl"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmp, "test.txt"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		var found []string
		err := WalkJSONLFiles(tmp, WalkOptions{Days: 30}, func(path string) error {
			found = append(found, filepath.Base(path))
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(found) != 1 || found[0] != "test.jsonl" {
			t.Errorf("got %v, want [test.jsonl]", found)
		}
	})

	t.Run("古いファイルはスキップされる", func(t *testing.T) {
		tmp := t.TempDir()
		f := filepath.Join(tmp, "old.jsonl")
		if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		oldTime := time.Now().Add(-60 * 24 * time.Hour)
		if err := os.Chtimes(f, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}

		var count int
		err := WalkJSONLFiles(tmp, WalkOptions{Days: 30}, func(path string) error {
			count++
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("古いファイルがコールバックに渡された: count=%d", count)
		}
	})
}

func TestNewScanner(t *testing.T) {
	t.Run("大きなバッファで行を読み取れる", func(t *testing.T) {
		data := strings.Repeat("a", 5*1024*1024) + "\n"
		r := strings.NewReader(data)
		s := NewScanner(r)
		if !s.Scan() {
			t.Fatal("スキャンに失敗")
		}
		if len(s.Text()) != 5*1024*1024 {
			t.Errorf("got len=%d, want %d", len(s.Text()), 5*1024*1024)
		}
	})
}

func TestCountUniqueFiles(t *testing.T) {
	type item struct{ path string }
	items := []item{
		{path: "/a/b.jsonl"},
		{path: "/a/b.jsonl"},
		{path: "/c/d.jsonl"},
	}
	got := CountUniqueFiles(items, func(i item) string { return i.path })
	if got != 2 {
		t.Errorf("got %d, want 2", got)
	}
}
