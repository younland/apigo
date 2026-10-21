package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// maxUploadBytes caps the file a form field uploads, so a huge one cannot stall
// a frame while it is copied.
const maxUploadBytes = 50 << 20

// staticPath is where a collection keeps an uploaded file, resolved from the
// path its interface file records.
func (s *store) staticPath(collectionID, rel string) string {
	return filepath.Join(s.collectionDir(collectionID), filepath.FromSlash(rel))
}

// importStatic copies a file into a collection's static directory, under the
// day it came and its SHA-256, and returns where it keeps it relative to the
// collection: static/<day>/<hash>. A file already kept there is left as it is,
// so uploading one twice costs nothing.
func (s *store) importStatic(collectionID, src string) (string, error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q 是一个目录，不是文件", filepath.Base(src))
	}
	if info.Size() > maxUploadBytes {
		return "", fmt.Errorf("文件超过 %d MB", maxUploadBytes>>20)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	rel := path.Join("static", time.Now().Format("2006-01-02"), hex.EncodeToString(sum[:]))
	dst := s.staticPath(collectionID, rel)
	if kept, err := os.Stat(dst); err == nil && kept.Size() == info.Size() {
		return rel, nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	tmpDir := filepath.Join(s.collectionDir(collectionID), tmpDirName)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	tmp := filepath.Join(tmpDir, hex.EncodeToString(sum[:])+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", err
	}
	return rel, nil
}

// removeStatic deletes a file a collection kept for an upload, once the field
// that held it is gone or holds another: only paths inside the collection's
// static directory are touched.
func (s *store) removeStatic(collectionID, rel string) error {
	if !strings.HasPrefix(path.Clean(strings.ReplaceAll(rel, "\\", "/")), "static/") {
		return nil // not one of ours: nothing to delete
	}
	err := os.Remove(s.staticPath(collectionID, rel))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
