package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
)

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Registry root directory (or user@host:/path)")
	sshKey := fs.String("ssh-key", "", "SSH private key for remote registry")
	sshPort := fs.Int("ssh-port", 22, "SSH port for remote registry")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	opts := sshOpts{key: *sshKey, port: *sshPort}
	return withRemote(*registryDir, opts, func(dir string) error {
		return listRegistry(dir)
	})
}

func listRegistry(registryDir string) error {
	cfg, err := loadConfig(registryDir)
	if err != nil {
		return err
	}

	if len(cfg.Providers) == 0 {
		logInfo("No providers registered yet.")
		return nil
	}

	fmt.Print("\nProviders in registry:\n\n")

	keys := make([]string, 0, len(cfg.Providers))
	for k := range cfg.Providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		p := cfg.Providers[k]
		fmt.Printf("  \033[36m%s/%s\033[0m\n", p.Namespace, p.Name)

		vers := make([]string, 0, len(p.Versions))
		for v := range p.Versions {
			vers = append(vers, v)
		}
		sort.Slice(vers, func(i, j int) bool { return vers[i] > vers[j] })

		for _, v := range vers {
			fmt.Printf("    v%s  →  %s\n", v, strings.Join(p.Versions[v].Platforms, ", "))
		}
	}
	fmt.Print("\n")
	return nil
}
