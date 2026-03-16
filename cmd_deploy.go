package main

import (
	"flag"
	"fmt"
	"os"
)

func cmdDeploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	registryDir := fs.String("registry-dir", "./registry", "Local registry root directory")
	sshKey := fs.String("ssh-key", "", "Path to SSH private key")
	sshPort := fs.Int("ssh-port", 22, "SSH port")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) < 1 {
		return fmt.Errorf("usage: tofu-provider deploy <user@host:/remote/path> [options]")
	}
	remote := positional[0]

	if _, err := os.Stat(*registryDir); os.IsNotExist(err) {
		return fmt.Errorf("registry directory not found: %s (run init first)", *registryDir)
	}

	opts := sshOpts{key: *sshKey, port: *sshPort}
	if err := rsyncPush(*registryDir, remote, opts); err != nil {
		return err
	}
	logOK(fmt.Sprintf("Registry deployed to %s", remote))
	return nil
}
