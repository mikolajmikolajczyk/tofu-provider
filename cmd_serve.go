package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Registry root directory")
	port := fs.Int("port", 8080, "Port to listen on")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	absDir, err := filepath.Abs(*registryDir)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		fsPath := filepath.Join(absDir, filepath.FromSlash(urlPath))

		// Reject path traversal attempts
		if !strings.HasPrefix(fsPath, absDir+string(filepath.Separator)) && fsPath != absDir {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		resolved, ct := resolveFile(fsPath)
		if resolved == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
				"errors": []map[string]string{
					{"status": "404", "title": "Not Found", "detail": urlPath},
				},
			})
			fmt.Printf("404  %s %s\n", r.Method, r.URL)
			return
		}

		data, err := os.ReadFile(resolved)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", ct)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(data) //nolint:errcheck

		rel, _ := filepath.Rel(absDir, resolved)
		fmt.Printf("200  %s %s  →  /%s\n", r.Method, r.URL, filepath.ToSlash(rel))
	})

	addr := fmt.Sprintf(":%d", *port)
	logOK(fmt.Sprintf("Registry server running at http://localhost%s", addr))
	logInfo(fmt.Sprintf("Test with: curl http://localhost%s/v1/providers/local/myprovider/versions", addr))
	fmt.Print("\nPress Ctrl+C to stop.\n\n")

	return http.ListenAndServe(addr, mux)
}

func resolveFile(fsPath string) (resolved, contentType string) {
	contentTypeFor := func(p string) string {
		switch {
		case strings.HasSuffix(p, ".json"):
			return "application/json"
		case strings.HasSuffix(p, ".zip"):
			return "application/zip"
		case strings.HasSuffix(p, ".sha256"):
			return "text/plain"
		default:
			return "application/octet-stream"
		}
	}

	if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
		return fsPath, contentTypeFor(fsPath)
	}
	if _, err := os.Stat(fsPath + "/index.json"); err == nil {
		return fsPath + "/index.json", "application/json"
	}
	if _, err := os.Stat(fsPath + "index.json"); err == nil {
		return fsPath + "index.json", "application/json"
	}
	return "", ""
}
