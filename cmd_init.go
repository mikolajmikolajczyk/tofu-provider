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
	sshKey := fs.String("ssh-key", "", "SSH private key for remote registry")
	sshPort := fs.Int("ssh-port", 22, "SSH port for remote registry")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	opts := sshOpts{key: *sshKey, port: *sshPort}
	return withRemote(*registryDir, opts, func(dir string) error {
		return initRegistry(dir)
	})
}

func initRegistry(registryDir string) error {
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(registryDir, registryConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := saveConfig(registryDir, &RegistryConfig{
			Providers: make(map[string]*ProviderEntry),
		}); err != nil {
			return err
		}
	}

	wellKnown := filepath.Join(registryDir, ".well-known")
	if err := os.MkdirAll(wellKnown, 0755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(wellKnown, "terraform.json"), map[string]string{
		"providers.v1": "/v1/providers/",
	}); err != nil {
		return err
	}

	absDir, _ := filepath.Abs(registryDir)
	nginxConf := fmt.Sprintf(`server {
    listen 80;
    server_name registry.example.com;

    root %s;

    location ~^/v1/providers/ {
        add_header Content-Type application/json;
        try_files $uri $uri/index.json =404;
        default_type application/json;
    }

    location = /.well-known/terraform.json {
        add_header Content-Type application/json;
        try_files $uri =404;
    }

    location ~\.zip$ {
        add_header Content-Type application/octet-stream;
    }
}
`, absDir)
	if err := os.WriteFile(filepath.Join(registryDir, "nginx.conf.example"), []byte(nginxConf), 0644); err != nil {
		return err
	}

	absWellKnown, _ := filepath.Abs(wellKnown)
	logOK(fmt.Sprintf("Registry initialized at: %s", absDir))
	logInfo(fmt.Sprintf("Well-known discovery: %s/terraform.json", absWellKnown))
	logInfo(fmt.Sprintf("Nginx sample config:   %s/nginx.conf.example", absDir))
	return nil
}
