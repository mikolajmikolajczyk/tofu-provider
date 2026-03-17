# tofu-provider

A CLI tool for managing a static Terraform / OpenTofu provider registry — no server-side logic required. It generates the JSON files that the [Terraform registry protocol](https://developer.hashicorp.com/terraform/internals/provider-registry-protocol) expects, packages your provider binary into a zip, and can push the result to a remote host over SSH.

## Installation

Download the binary for your platform from the [releases page](../../releases), or build from source:

```sh
go install tofu-provider@latest
```

## Concepts

The registry is a directory tree of static JSON files and zip archives. Once generated, you serve it with any static file server (nginx, Caddy, S3, etc.). Terraform and OpenTofu discover providers through the well-known endpoint `/.well-known/terraform.json` and then follow links to version and download JSON files.

```
registry/
├── .well-known/
│   └── terraform.json
└── v1/
    └── providers/
        └── <namespace>/
            └── <name>/
                ├── versions/
                │   └── index.json
                └── <version>/
                    ├── terraform-provider-<name>_<version>_SHA256SUMS
                    ├── terraform-provider-<name>_<version>_SHA256SUMS.sig
                    └── download/
                        └── <os>/
                            └── <arch>/
                                ├── index.json
                                └── terraform-provider-<name>_<version>_<os>_<arch>.zip
```

## Usage

### Initialize a registry

```sh
tofu-provider init
tofu-provider init --registry-dir /var/www/registry

# Registry served under a subpath with absolute download URLs
tofu-provider init --hostname registry.example.com --base-path /tf-providers

# Directly on a remote host
tofu-provider init --registry-dir deploy@registry.example.com:/var/www/registry --ssh-key ~/.ssh/id_ed25519
```

Available options for `init`:

| Flag | Default | Description |
|---|---|---|
| `--registry-dir` | `./registry` | Local path or `user@host:/path` |
| `--hostname` | — | Hostname the registry is served from (e.g. `registry.example.com`) |
| `--base-path` | — | URL prefix the registry is served under (e.g. `/tf-providers`) |
| `--ssh-key` | — | SSH private key for remote registry |
| `--ssh-port` | `22` | SSH port for remote registry |

`--hostname` and `--base-path` are saved in `.registry.json` and automatically applied by subsequent `add` commands — you don't need to repeat them.

When `--hostname` is set, `download_url`, `shasums_url`, and `shasums_signature_url` in each download `index.json` are generated as absolute HTTPS URLs, which OpenTofu requires. Without it they fall back to relative paths.

`init` also generates an RSA-4096 GPG key pair and stores it in `.registry.json`. The private key is used by `add` to sign the `SHA256SUMS` file — satisfying the registry protocol's requirement for a detached GPG signature. If you skip `init`, the key is generated automatically on the first `add`.

### Add a provider binary

Platform (OS and architecture) is auto-detected from the filename. The binary is wrapped in a zip if it isn't one already.

```sh
# Single platform
tofu-provider add myprovider 1.2.3 ./terraform-provider-myprovider_linux_amd64

# Multiple platforms at once — OS/arch detected per file, namespace applies to all
tofu-provider add myprovider 1.2.3 \
  ./terraform-provider-myprovider_linux_amd64 \
  ./terraform-provider-myprovider_linux_arm64 \
  ./terraform-provider-myprovider_darwin_arm64 \
  ./terraform-provider-myprovider_windows_amd64.exe \
  --namespace mycompany

# Override namespace or platform explicitly
tofu-provider add myprovider 1.2.3 ./provider.zip \
  --namespace mycompany \
  --os darwin --arch arm64
```

Available options for `add`:

| Flag | Default | Description |
|---|---|---|
| `--namespace` | `local` | Provider namespace |
| `--os` | auto-detected | Target OS (single file only) |
| `--arch` | auto-detected | Target architecture (single file only) |
| `--registry-dir` | `./registry` | Local path or `user@host:/path` |
| `--protocols` | `6.0,5.1` | Supported Terraform protocol versions |
| `--ssh-key` | — | SSH private key for remote registry |
| `--ssh-port` | `22` | SSH port for remote registry |

### List providers

```sh
tofu-provider list
tofu-provider list --registry-dir deploy@registry.example.com:/var/www/registry --ssh-key ~/.ssh/id_ed25519
```

### Remove a provider version

```sh
tofu-provider remove myprovider 1.2.3
tofu-provider remove myprovider 1.2.3 --namespace mycompany
```

### Test locally

```sh
tofu-provider serve
tofu-provider serve --port 9090
```

The server resolves JSON endpoints with or without a trailing slash, mimicking nginx `try_files` behaviour.

### Deploy to a remote host

Pushes the local registry directory to a remote server using `rsync` (falls back to `scp`):

```sh
tofu-provider deploy deploy@registry.example.com:/var/www/registry
tofu-provider deploy deploy@registry.example.com:/var/www/registry --ssh-key ~/.ssh/deploy_key --ssh-port 2222
```

All commands that take `--registry-dir` also accept a remote path (`user@host:/path`) directly, performing a pull → modify → push cycle automatically. `deploy` is for an explicit one-way push when you already have the registry locally.

## Remote registry (CI usage)

All commands work against a remote registry transparently — no local copy needed:

```sh
# In CI: add a freshly built binary directly to the remote registry
tofu-provider add myprovider 1.2.3 ./provider_linux_amd64 \
  --registry-dir deploy@registry.example.com:/var/www/registry \
  --ssh-key ~/.ssh/deploy_key
```

The tool pulls the registry to a temp directory, applies the change, then pushes back in a single rsync pass.

## Serving with nginx

After running `init`, a sample nginx config is written to `nginx.conf.example` in your registry directory. The minimal setup:

```nginx
server {
    listen 443 ssl;
    server_name registry.example.com;
    root /var/www/registry;

    location ~^/v1/providers/ {
        add_header Content-Type application/json;
        try_files $uri $uri/index.json =404;
    }

    location = /.well-known/terraform.json {
        add_header Content-Type application/json;
    }

    location ~\.(zip|sig)$ {
        add_header Content-Type application/octet-stream;
    }
}
```

## Using the registry in Terraform / OpenTofu

```hcl
terraform {
  required_providers {
    myprovider = {
      source  = "registry.example.com/mycompany/myprovider"
      version = "1.2.3"
    }
  }
}
```

## License

MIT — see [LICENSE](LICENSE).
