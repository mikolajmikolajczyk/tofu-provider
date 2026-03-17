package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func cmdAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Registry root directory (or user@host:/path)")
	namespace := fs.String("namespace", "local", "Provider namespace")
	targetOS := fs.String("os", "", "Target OS")
	targetArch := fs.String("arch", "", "Target arch")
	protocols := fs.String("protocols", "6.0,5.1", "Supported protocols (comma-separated)")
	gpgKeyID := fs.String("gpg-key-id", "", "GPG key ID label (decorative only)")
	sshKey := fs.String("ssh-key", "", "SSH private key for remote registry")
	sshPort := fs.Int("ssh-port", 22, "SSH port for remote registry")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) < 3 {
		return fmt.Errorf("usage: tofu-provider add <name> <version> <path_to_file> [<path_to_file> ...]")
	}
	name, version, filePaths := positional[0], positional[1], positional[2:]

	// --os/--arch only apply when a single file is given
	if len(filePaths) > 1 && (*targetOS != "" || *targetArch != "") {
		return fmt.Errorf("--os/--arch cannot be used with multiple files; platform is auto-detected per file")
	}

	for _, fp := range filePaths {
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", fp)
		}
	}

	protocolList := splitTrimmed(*protocols, ",")
	opts := sshOpts{key: *sshKey, port: *sshPort}

	return withRemote(*registryDir, opts, func(dir string) error {
		for _, fp := range filePaths {
			detected := detectPlatform(filepath.Base(fp))
			goos := detected.OS
			goarch := detected.Arch
			if *targetOS != "" {
				goos = *targetOS
			}
			if *targetArch != "" {
				goarch = *targetArch
			}
			if err := addToRegistry(dir, name, version, fp, *namespace, goos, goarch, *gpgKeyID, protocolList); err != nil {
				return err
			}
		}
		return nil
	})
}

func addToRegistry(registryDir, name, version, filePath, namespace, targetOS, targetArch, gpgKeyID string, protocolList []string) error {
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return err
	}

	// Load config early to get base_path for the well-known file.
	cfg, err := loadConfig(registryDir)
	if err != nil {
		return err
	}

	// Ensure .well-known exists
	wellKnown := filepath.Join(registryDir, ".well-known")
	tfJSON := filepath.Join(wellKnown, "terraform.json")
	if _, err := os.Stat(tfJSON); os.IsNotExist(err) {
		if err := os.MkdirAll(wellKnown, 0755); err != nil {
			return err
		}
		if err := writeJSON(tfJSON, map[string]string{"providers.v1": providersV1Path(cfg.BasePath)}); err != nil {
			return err
		}
	}

	providerDir := filepath.Join(registryDir, "v1", "providers", namespace, name)
	platformKey := fmt.Sprintf("%s_%s", targetOS, targetArch)
	dlDir := filepath.Join(providerDir, version, "download", platformKey)
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return err
	}

	zipName := fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", name, version, targetOS, targetArch)
	logInfo(fmt.Sprintf("Packaging binary → %s …", zipName))

	zipPath, err := ensureZip(filePath, dlDir, zipName)
	if err != nil {
		return fmt.Errorf("failed to create zip: %w", err)
	}

	shasum, err := sha256File(zipPath)
	if err != nil {
		return err
	}
	logInfo(fmt.Sprintf("SHA256: %s", shasum))

	keyID := "unsigned"
	if gpgKeyID != "" {
		keyID = gpgKeyID
	}

	downloadDoc := DownloadDoc{
		Protocols:   protocolList,
		OS:          targetOS,
		Arch:        targetArch,
		Filename:    zipName,
		DownloadURL: zipName,
		Shasum:      shasum,
		ShasumURL:   zipName + ".sha256",
		SigningKeys:  SigningKeys{GPGPublicKeys: []GPGKey{{KeyID: keyID, ASCIIArmor: ""}}},
	}
	if err := os.WriteFile(zipPath+".sha256", []byte(fmt.Sprintf("%s  %s\n", shasum, zipName)), 0644); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dlDir, "index.json"), downloadDoc); err != nil {
		return err
	}

	// Update versions index
	versionsDir := filepath.Join(providerDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return err
	}
	versionsIndex := filepath.Join(versionsDir, "index.json")

	var versionsDoc VersionsDoc
	if data, err := os.ReadFile(versionsIndex); err == nil {
		json.Unmarshal(data, &versionsDoc) //nolint:errcheck
	}

	var vEntry *VersionInfo
	for i := range versionsDoc.Versions {
		if versionsDoc.Versions[i].Version == version {
			vEntry = &versionsDoc.Versions[i]
			break
		}
	}
	if vEntry == nil {
		versionsDoc.Versions = append(versionsDoc.Versions, VersionInfo{Version: version, Protocols: protocolList})
		vEntry = &versionsDoc.Versions[len(versionsDoc.Versions)-1]
	}

	hasPlatform := false
	for _, p := range vEntry.Platforms {
		if p.OS == targetOS && p.Arch == targetArch {
			hasPlatform = true
			break
		}
	}
	if !hasPlatform {
		vEntry.Platforms = append(vEntry.Platforms, PlatformInfo{OS: targetOS, Arch: targetArch})
	}
	vEntry.Protocols = protocolList

	sort.Slice(versionsDoc.Versions, func(i, j int) bool {
		return versionsDoc.Versions[i].Version > versionsDoc.Versions[j].Version
	})
	if err := writeJSON(versionsIndex, versionsDoc); err != nil {
		return err
	}

	// Update .registry.json (cfg already loaded at top of function)
	key := fmt.Sprintf("%s/%s", namespace, name)
	if cfg.Providers[key] == nil {
		cfg.Providers[key] = &ProviderEntry{
			Namespace: namespace,
			Name:      name,
			Versions:  make(map[string]*VersionEntry),
		}
	}
	if cfg.Providers[key].Versions[version] == nil {
		cfg.Providers[key].Versions[version] = &VersionEntry{}
	}
	platforms := cfg.Providers[key].Versions[version].Platforms
	hasPlatformInConfig := false
	for _, p := range platforms {
		if p == platformKey {
			hasPlatformInConfig = true
			break
		}
	}
	if !hasPlatformInConfig {
		cfg.Providers[key].Versions[version].Platforms = append(platforms, platformKey)
	}
	if err := saveConfig(registryDir, cfg); err != nil {
		return err
	}

	logOK(fmt.Sprintf("Provider added: %s/%s v%s (%s/%s)", namespace, name, version, targetOS, targetArch))
	logInfo(fmt.Sprintf("Versions index: %s", versionsIndex))
	logInfo(fmt.Sprintf("Download JSON:  %s", filepath.Join(dlDir, "index.json")))
	logInfo(fmt.Sprintf("Binary:         %s", zipPath))

	fmt.Printf("\n\033[33mTerraform / OpenTofu config:\033[0m\n\n"+
		"  terraform {\n"+
		"    required_providers {\n"+
		"      %s = {\n"+
		"        source  = \"<your-registry-host>/%s/%s\"\n"+
		"        version = \"%s\"\n"+
		"      }\n"+
		"    }\n"+
		"  }\n\n",
		name, namespace, name, version)

	return nil
}

func splitTrimmed(s, sep string) []string {
	parts := strings.Split(s, sep)
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
