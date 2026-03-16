package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func cmdRemove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Registry root directory (or user@host:/path)")
	namespace := fs.String("namespace", "local", "Provider namespace")
	sshKey := fs.String("ssh-key", "", "SSH private key for remote registry")
	sshPort := fs.Int("ssh-port", 22, "SSH port for remote registry")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) < 2 {
		return fmt.Errorf("usage: tofu-provider remove <name> <version>")
	}
	name, version := positional[0], positional[1]
	opts := sshOpts{key: *sshKey, port: *sshPort}

	return withRemote(*registryDir, opts, func(dir string) error {
		return removeFromRegistry(dir, *namespace, name, version)
	})
}

func removeFromRegistry(registryDir, namespace, name, version string) error {
	cfg, err := loadConfig(registryDir)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", namespace, name)
	entry, exists := cfg.Providers[key]
	if !exists {
		return fmt.Errorf("provider %s/%s not found in registry", namespace, name)
	}
	vEntry, exists := entry.Versions[version]
	if !exists {
		return fmt.Errorf("provider %s/%s v%s not found in registry", namespace, name, version)
	}

	for _, plat := range vEntry.Platforms {
		dlDir := filepath.Join(registryDir, "v1", "providers", namespace, name, version, "download", plat)
		os.RemoveAll(dlDir) //nolint:errcheck
	}

	delete(entry.Versions, version)
	if len(entry.Versions) == 0 {
		delete(cfg.Providers, key)
	}
	if err := saveConfig(registryDir, cfg); err != nil {
		return err
	}

	// Regenerate versions index
	versionsIndex := filepath.Join(registryDir, "v1", "providers", namespace, name, "versions", "index.json")
	if data, err := os.ReadFile(versionsIndex); err == nil {
		var doc VersionsDoc
		if err := json.Unmarshal(data, &doc); err == nil {
			filtered := doc.Versions[:0]
			for _, v := range doc.Versions {
				if v.Version != version {
					filtered = append(filtered, v)
				}
			}
			doc.Versions = filtered
			writeJSON(versionsIndex, doc) //nolint:errcheck
		}
	}

	logOK(fmt.Sprintf("Removed %s/%s v%s", namespace, name, version))
	return nil
}
