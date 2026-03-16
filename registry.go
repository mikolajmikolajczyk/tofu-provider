package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const registryConfigFile = ".registry.json"

// ── stored config (.registry.json) ──────────────────────────────────────────

type RegistryConfig struct {
	Providers map[string]*ProviderEntry `json:"providers"`
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
