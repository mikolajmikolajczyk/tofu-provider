package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

type Platform struct {
	OS   string
	Arch string
}

// separators splits a file name into the tokens a Go release artifact is
// normally built from: terraform-provider-foo_1.2.3_linux_amd64.zip.
var separators = regexp.MustCompile(`[^a-z0-9]+`)

var knownOS = map[string]string{
	"darwin":  "darwin",
	"macos":   "darwin",
	"osx":     "darwin",
	"windows": "windows",
	"win":     "windows",
	"freebsd": "freebsd",
	"linux":   "linux",
}

var knownArch = map[string]string{
	"amd64":   "amd64",
	"arm64":   "arm64",
	"aarch64": "arm64",
	"arm":     "arm",
	"386":     "386",
}

// detectPlatform guesses the target platform from a file name.
//
// Whole tokens first, substrings only as a fallback. A substring match over the
// whole name is wrong whenever the name carries anything else numeric: a build
// uploaded as terraform-provider-okd_linux_amd64_34947983860 was registered as
// linux/386, because the CI run id contains "386" and that matched before the
// amd64 default. The binary was x86-64, so the published provider could not be
// downloaded at all.
//
// Tokens are read back to front, because a Go release artifact ends in
// <os>_<arch> and anything before that is a name or a version.
//
// The fallback keeps names with no separators working, e.g. providerlinuxarm64.
func detectPlatform(filename string) Platform {
	f := strings.ToLower(filepath.Base(filename))

	// Back to front: Go release artifacts end in <os>_<arch>, and anything
	// earlier is a name or a version. Reading forwards, a version like
	// foo_1.386.0_linux_amd64 would settle on 386 before reaching amd64.
	tokens := separators.Split(f, -1)
	var os, arch string
	for i := len(tokens) - 1; i >= 0; i-- {
		tok := tokens[i]
		if os == "" {
			if v, ok := knownOS[tok]; ok {
				os = v
			}
		}
		if arch == "" {
			if v, ok := knownArch[tok]; ok {
				arch = v
			}
		}
	}

	if os == "" {
		os = detectOSSubstring(f)
	}
	if arch == "" {
		arch = detectArchSubstring(f)
	}

	return Platform{OS: os, Arch: arch}
}

func detectOSSubstring(f string) string {
	switch {
	case strings.Contains(f, "darwin") || strings.Contains(f, "macos") || strings.Contains(f, "osx"):
		return "darwin"
	case strings.Contains(f, "windows") || strings.Contains(f, "win"):
		return "windows"
	case strings.Contains(f, "freebsd"):
		return "freebsd"
	}
	return "linux"
}

func detectArchSubstring(f string) string {
	switch {
	case strings.Contains(f, "arm64") || strings.Contains(f, "aarch64"):
		return "arm64"
	case strings.Contains(f, "arm"):
		return "arm"
	case strings.Contains(f, "386"):
		return "386"
	}
	return "amd64"
}
