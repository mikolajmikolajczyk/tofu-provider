package main

import (
	"path/filepath"
	"strings"
)

type Platform struct {
	OS   string
	Arch string
}

func detectPlatform(filename string) Platform {
	f := strings.ToLower(filepath.Base(filename))

	os := "linux"
	switch {
	case strings.Contains(f, "darwin") || strings.Contains(f, "macos") || strings.Contains(f, "osx"):
		os = "darwin"
	case strings.Contains(f, "windows") || strings.Contains(f, "win"):
		os = "windows"
	case strings.Contains(f, "freebsd"):
		os = "freebsd"
	}

	arch := "amd64"
	switch {
	case strings.Contains(f, "arm64") || strings.Contains(f, "aarch64"):
		arch = "arm64"
	case strings.Contains(f, "arm"):
		arch = "arm"
	case strings.Contains(f, "386"):
		arch = "386"
	}

	return Platform{OS: os, Arch: arch}
}
