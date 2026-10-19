// Package logfile is a tiny size-capped log writer for installs that run for years.
// It keeps the current file plus one ".1" backup, so disk use never exceeds 2×MaxSize.
package logfile

import (
	"os"
	"path/filepath"
	"sync"
)

const MaxSize = 1 << 20 // 1 MiB per file

type Writer struct {
	mu   sync.Mutex
	path string
	file *os.File
	size int64
}

func Open(path string) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	w := &Writer{path: path}
	return w, w.open()
}

func (w *Writer) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	w.file, w.size = file, info.Size()
	return nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size+int64(len(p)) > MaxSize && w.size > 0 {
		w.file.Close()
		w.file = nil
		_ = os.Rename(w.path, w.path+".1") // replaces the previous backup
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
