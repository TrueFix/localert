package logfile

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLogNeverExceedsTwoFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "localert.log")
	w, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	line := bytes.Repeat([]byte("x"), 1000)
	for i := 0; i < 5000; i++ { // ~5 MB of writes
		if _, err := w.Write(append(line, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	w.Close()
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*"))
	if len(matches) != 2 {
		t.Fatalf("expected current + one backup, got %v", matches)
	}
	for _, file := range matches {
		info, _ := os.Stat(file)
		if info.Size() > MaxSize {
			t.Fatalf("%s is %d bytes", file, info.Size())
		}
	}
}
