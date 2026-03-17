package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const registryConfigFile = ".registry.json"

// ── stored config (.registry.json) ──────────────────────────────────────────

type RegistryConfig struct {
	Hostname  string                    `json:"hostname,omitempty"`
	BasePath  string                    `json:"base_path,omitempty"`
	Providers map[string]*ProviderEntry `json:"providers"`
}

// providersV1Path returns the providers.v1 URL path for the well-known file,
// e.g. "" → "/v1/providers/", "/tf-providers" → "/tf-providers/v1/providers/".
func providersV1Path(basePath string) string {
	bp := strings.TrimRight(basePath, "/")
	if bp != "" && !strings.HasPrefix(bp, "/") {
		bp = "/" + bp
	}
	return bp + "/v1/providers/"
}

// downloadFileURL returns an absolute URL for a file inside the download directory.
// Returns an empty string when hostname is not set (caller falls back to bare filename).
// e.g. hostname="registry.example.com", basePath="/tf-providers",
//
//	namespace="myco", name="myprovider", version="1.0.0",
//	platformKey="linux_amd64", filename="terraform-provider-myprovider_1.0.0_linux_amd64.zip"
//
// → "https://registry.example.com/tf-providers/v1/providers/myco/myprovider/1.0.0/download/linux_amd64/terraform-provider-myprovider_1.0.0_linux_amd64.zip"
func downloadFileURL(hostname, basePath, namespace, name, version, platformKey, filename string) string {
	if hostname == "" {
		return ""
	}
	base := strings.TrimRight(providersV1Path(basePath), "/")
	return fmt.Sprintf("https://%s%s/%s/%s/%s/download/%s/%s",
		hostname, base, namespace, name, version, platformKey, filename)
}

type ProviderEntry struct {
	Namespace string                    `json:"namespace"`
	Name      string                    `json:"name"`
	Versions  map[string]*VersionEntry  `json:"versions"`
}

type VersionEntry struct {
	Platforms []string `json:"platforms"`
}

// ── registry API document types ──────────────────────────────────────────────

type VersionsDoc struct {
	Versions []VersionInfo `json:"versions"`
}

type VersionInfo struct {
	Version   string         `json:"version"`
	Protocols []string       `json:"protocols"`
	Platforms []PlatformInfo `json:"platforms"`
}

type PlatformInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type DownloadDoc struct {
	Protocols   []string    `json:"protocols"`
	OS          string      `json:"os"`
	Arch        string      `json:"arch"`
	Filename    string      `json:"filename"`
	DownloadURL string      `json:"download_url"`
	Shasum      string      `json:"shasum"`
	ShasumURL   string      `json:"shasum_url"`
	SigningKeys  SigningKeys `json:"signing_keys"`
}

type SigningKeys struct {
	GPGPublicKeys []GPGKey `json:"gpg_public_keys"`
}

type GPGKey struct {
	KeyID      string `json:"key_id"`
	ASCIIArmor string `json:"ascii_armor"`
}

// ── helpers ──────────────────────────────────────────────────────────────────

func loadConfig(registryDir string) (*RegistryConfig, error) {
	path := filepath.Join(registryDir, registryConfigFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &RegistryConfig{Providers: make(map[string]*ProviderEntry)}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg RegistryConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]*ProviderEntry)
	}
	return &cfg, nil
}

func saveConfig(registryDir string, cfg *RegistryConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(registryDir, registryConfigFile), data, 0644)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
