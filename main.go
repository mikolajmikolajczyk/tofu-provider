package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		os.Exit(0)
	}

	command := args[0]
	rest := args[1:]

	var err error
	switch command {
	case "init":
		err = cmdInit(rest)
	case "add":
		err = cmdAdd(rest)
	case "list":
		err = cmdList(rest)
	case "remove":
		err = cmdRemove(rest)
	case "serve":
		err = cmdServe(rest)
	case "deploy":
		err = cmdDeploy(rest)
	default:
		logError(fmt.Sprintf("unknown command: %s. Run with --help for usage.", command))
		os.Exit(1)
	}

	if err != nil {
		logError(err.Error())
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`tofu-provider v%s — static Terraform/OpenTofu provider registry

USAGE
  tofu-provider init [--registry-dir <dir>] [--base-path <path>] [--ssh-key <path>] [--ssh-port <port>]
      Create the registry directory structure and initial config.

      Options:
        --base-path <path>    URL prefix the registry is served under (default: "")
                              e.g. --base-path /tf-providers generates {"providers.v1": "/tf-providers/v1/providers/"}
                              Saved in .registry.json and reused by subsequent commands.

  tofu-provider add <name> <version> <path_to_file> [<path_to_file> ...] [options]
      Add one or more provider binaries to the registry.
      Platform is auto-detected per file from its name; --os/--arch apply only for a single file.

      Options:
        --namespace <ns>      Provider namespace  (default: "local")
        --os <os>             Target OS           (default: autodetect from filename or "linux")
        --arch <arch>         Target arch         (default: autodetect from filename or "amd64")
        --registry-dir <dir>  Registry root dir   (default: "./registry", or user@host:/path)
        --protocols <p,q>     Supported protocols (default: "6.0,5.1")
        --gpg-key-id <id>     GPG key ID label    (optional, decorative only)
        --ssh-key <path>      SSH private key for remote registry
        --ssh-port <port>     SSH port            (default: 22)

  tofu-provider list [--registry-dir <dir>] [--ssh-key <path>] [--ssh-port <port>]
      List all providers in the registry.

  tofu-provider remove <name> <version> [--namespace <ns>] [--registry-dir <dir>] [--ssh-key <path>] [--ssh-port <port>]
      Remove a provider version from the registry.

  tofu-provider serve [--registry-dir <dir>] [--port <port>]
      Start a local HTTP server for testing.

  tofu-provider deploy <user@host:/remote/path> [options]
      Sync the registry to a remote server over SSH using rsync (or scp as fallback).

      Options:
        --registry-dir <dir>  Local registry root dir  (default: "./registry")
        --ssh-key <path>      SSH private key
        --ssh-port <port>     SSH port                 (default: 22)

EXAMPLES
  # Minimal — add provider binary for linux/amd64
  tofu-provider add myprovider 1.2.3 ./terraform-provider-myprovider_v1.2.3

  # Explicit namespace + OS/arch
  tofu-provider add myprovider 1.2.3 ./provider.zip --namespace mycompany --os darwin --arch arm64

  # Then serve to test with tofu
  tofu-provider serve

NGINX SNIPPET
  After generating, point nginx at the registry dir:

    server {
      listen 443 ssl;
      server_name registry.example.com;
      root /var/www/registry;
      location / {
        add_header Content-Type application/json;
        try_files $uri $uri/ =404;
      }
      location ~\.zip$ {
        add_header Content-Type application/zip;
      }
    }

  In terraform / tofu:

    terraform {
      required_providers {
        myprovider = {
          source  = "registry.example.com/mycompany/myprovider"
          version = "1.2.3"
        }
      }
    }
`, version)
}
