package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ensureZip copies srcPath into destDir/zipName as a zip archive.
// If srcPath is already a .zip it is copied as-is.
func ensureZip(srcPath, destDir, zipName string) (string, error) {
	destZip := filepath.Join(destDir, zipName)

	if strings.HasSuffix(strings.ToLower(srcPath), ".zip") {
		return destZip, copyFile(srcPath, destZip)
	}

	zf, err := os.Create(destZip)
	if err != nil {
		return "", err
	}
	defer zf.Close()

	w := zip.NewWriter(zf)
	defer w.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return "", err
	}

	fh, err := zip.FileInfoHeader(info)
	if err != nil {
		return "", err
	}
	fh.Name = filepath.Base(srcPath)
	fh.Method = zip.Deflate

	entry, err := w.CreateHeader(fh)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(entry, src); err != nil {
		return "", err
	}

	return destZip, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
