package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Registry root directory (or user@host:/path)")
	hostname := fs.String("hostname", "", "Hostname the registry is served from (e.g. registry.example.com)")
	basePath := fs.String("base-path", "", "URL prefix the registry is served under (e.g. /tf-providers)")
	sshKey := fs.String("ssh-key", "", "SSH private key for remote registry")
	sshPort := fs.Int("ssh-port", 22, "SSH port for remote registry")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	opts := sshOpts{key: *sshKey, port: *sshPort}
	return withRemote(*registryDir, opts, func(dir string) error {
		return initRegistry(dir, *hostname, *basePath)
	})
}

func initRegistry(registryDir, hostname, basePath string) error {
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return err
	}

	// Load existing config so we don't overwrite providers, then update settings.
	cfg, err := loadConfig(registryDir)
	if err != nil {
		return err
	}
	cfg.Hostname = hostname
	cfg.BasePath = basePath

	// Generate a GPG key pair if one doesn't exist yet.
	if cfg.GPGKeyID == "" {
		logInfo("Generating GPG key pair…")
		keyID, pubKey, privKey, err := generateGPGKey()
		if err != nil {
			return fmt.Errorf("generate GPG key: %w", err)
		}
		cfg.GPGKeyID = keyID
		cfg.GPGPublicKey = pubKey
		cfg.GPGPrivateKey = privKey
		logOK(fmt.Sprintf("GPG key generated:      %s", keyID))
	} else {
		logInfo(fmt.Sprintf("GPG key (existing):     %s", cfg.GPGKeyID))
	}

	if err := saveConfig(registryDir, cfg); err != nil {
		return err
	}

	wellKnown := filepath.Join(registryDir, ".well-known")
	if err := os.MkdirAll(wellKnown, 0755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(wellKnown, "terraform.json"), map[string]string{
		"providers.v1": providersV1Path(basePath),
	}); err != nil {
		return err
	}

	absDir, _ := filepath.Abs(registryDir)
	providersPath := providersV1Path(basePath)
	nginxConf := fmt.Sprintf(`server {
    listen 80;
    server_name registry.example.com;

    root %s;

    location ~^%s {
        add_header Content-Type application/json;
        try_files $uri $uri/index.json =404;
        default_type application/json;
    }

    location = /.well-known/terraform.json {
        add_header Content-Type application/json;
        try_files $uri =404;
    }

    location ~\.(zip|sig)$ {
        add_header Content-Type application/octet-stream;
    }
}
`, absDir, providersPath)
	if err := os.WriteFile(filepath.Join(registryDir, "nginx.conf.example"), []byte(nginxConf), 0644); err != nil {
		return err
	}

	absWellKnown, _ := filepath.Abs(wellKnown)
	logOK(fmt.Sprintf("Registry initialized at: %s", absDir))
	if hostname != "" {
		logInfo(fmt.Sprintf("Hostname:               %s", hostname))
	}
	logInfo(fmt.Sprintf("providers.v1 path:      %s", providersPath))
	logInfo(fmt.Sprintf("Well-known discovery:   %s/terraform.json", absWellKnown))
	logInfo(fmt.Sprintf("Nginx sample config:    %s/nginx.conf.example", absDir))
	return nil
}
