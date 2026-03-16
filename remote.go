package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type sshOpts struct {
	key  string
	port int
}

func (o sshOpts) sshE() string {
	s := fmt.Sprintf("ssh -p %d -o StrictHostKeyChecking=accept-new", o.port)
	if o.key != "" {
		s += " -i " + o.key
	}
	return s
}

// isRemote returns true when path looks like user@host:/path or host:/path.
func isRemote(path string) bool {
	return strings.Contains(path, ":") && !strings.HasPrefix(path, "/")
}

// withRemote transparently handles remote registry dirs.
// If registryDir is a remote SSH path (user@host:/path), it pulls to a temp
// dir, calls fn, then pushes changes back. Local paths call fn directly.
func withRemote(registryDir string, opts sshOpts, fn func(localDir string) error) error {
	if !isRemote(registryDir) {
		return fn(registryDir)
	}

	tmp, err := os.MkdirTemp("", "tofu-registry-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	logInfo(fmt.Sprintf("Pulling registry from %s …", registryDir))
	if err := syncFiles(registryDir+"/", tmp+"/", opts); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	if err := fn(tmp); err != nil {
		return err
	}

	logInfo(fmt.Sprintf("Pushing changes back to %s …", registryDir))
	if err := syncFiles(tmp+"/", registryDir+"/", opts); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	return nil
}

func rsyncPush(localDir, remote string, opts sshOpts) error {
	return syncFiles(strings.TrimRight(localDir, "/")+"/", remote, opts)
}

// syncFiles copies src to dst using rsync if available, falling back to scp.
// For push (local→remote, with --delete) vs pull (remote→local, no --delete)
// the direction is inferred from whether src or dst is a remote path.
func syncFiles(src, dst string, opts sshOpts) error {
	if rsync, err := exec.LookPath("rsync"); err == nil {
		args := []string{"-az", "-e", opts.sshE()}
		// Only delete when pushing (dst is remote) to avoid wiping local temp on pull
		if isRemote(dst) {
			args = append(args, "--delete")
		}
		args = append(args, src, dst)
		logInfo("rsync " + strings.Join(args, " "))
		return runCmd(rsync, args)
	}

	if scp, err := exec.LookPath("scp"); err == nil {
		args := []string{"-r", "-P", fmt.Sprintf("%d", opts.port), "-o", "StrictHostKeyChecking=accept-new"}
		if opts.key != "" {
			args = append(args, "-i", opts.key)
		}
		args = append(args, src, dst)
		logInfo("scp " + strings.Join(args, " "))
		return runCmd(scp, args)
	}

	return fmt.Errorf("neither rsync nor scp found in PATH")
}

func runCmd(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
