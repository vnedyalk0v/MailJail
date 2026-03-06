package bastille

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/host/command"
)

const defaultPrefix = "/usr/local/bastille"

type Client struct {
	Runner command.Runner
	Prefix string
}

func New(runner command.Runner) Client {
	return Client{
		Runner: runner,
		Prefix: defaultPrefix,
	}
}

func (c Client) EnsureRelease(ctx context.Context, release string) error {
	if release == "" {
		return errors.New("release is required")
	}

	if _, err := os.Stat(filepath.Join(c.Prefix, "releases", release)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat bastille release: %w", err)
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"bootstrap", release},
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("bootstrap Bastille release %s: %w", release, err)
	}
	return nil
}

func (c Client) EnsureJail(ctx context.Context, name, release, ipCIDR, iface string) error {
	if name == "" || release == "" || ipCIDR == "" {
		return errors.New("name, release, and ipCIDR are required")
	}

	if _, err := os.Stat(filepath.Join(c.Prefix, "jails", name)); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat bastille jail: %w", err)
	}

	args := []string{"create", name, release, ipCIDR}
	if iface != "" {
		args = append(args, iface)
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    args,
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("create Bastille jail %s: %w", name, err)
	}

	return nil
}

func (c Client) InstallPackages(ctx context.Context, jail string, packages []string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if len(packages) == 0 {
		return errors.New("at least one package is required")
	}

	args := []string{"pkg", jail, "install", "-y"}
	args = append(args, packages...)

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    args,
		Timeout: 20 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("install packages in jail %s: %w", jail, err)
	}
	return nil
}

func (c Client) SetRC(ctx context.Context, jail, setting string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if setting == "" {
		return errors.New("setting is required")
	}

	_, err := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"sysrc", jail, setting},
		Timeout: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("set sysrc in jail %s: %w", jail, err)
	}
	return nil
}

func (c Client) EnsureServiceStarted(ctx context.Context, jail, service string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if service == "" {
		return errors.New("service is required")
	}

	_, statusErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"cmd", jail, "service", service, "onestatus"},
		Timeout: 30 * time.Second,
	})
	if statusErr == nil {
		return nil
	}

	_, startErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"service", jail, service, "start"},
		Timeout: 2 * time.Minute,
	})
	if startErr != nil {
		return fmt.Errorf("start service %s in jail %s: %w", service, jail, startErr)
	}

	return nil
}

func (c Client) SyncFile(jail, path string, content []byte, mode fs.FileMode) (bool, error) {
	if jail == "" {
		return false, errors.New("jail is required")
	}
	if path == "" {
		return false, errors.New("path is required")
	}
	if !filepath.IsAbs(path) {
		return false, fmt.Errorf("path %q must be absolute", path)
	}

	fullPath, err := c.jailPath(jail, path)
	if err != nil {
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return false, fmt.Errorf("create parent directory for %s: %w", fullPath, err)
	}

	existing, err := os.ReadFile(fullPath)
	if err == nil {
		info, statErr := os.Stat(fullPath)
		if statErr == nil && bytes.Equal(existing, content) && info.Mode().Perm() == mode.Perm() {
			return false, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("read existing file %s: %w", fullPath, err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(fullPath), ".mailjail-*")
	if err != nil {
		return false, fmt.Errorf("create temp file for %s: %w", fullPath, err)
	}

	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return false, fmt.Errorf("write temp file for %s: %w", fullPath, err)
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return false, fmt.Errorf("chmod temp file for %s: %w", fullPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return false, fmt.Errorf("close temp file for %s: %w", fullPath, err)
	}

	if err := os.Rename(tempPath, fullPath); err != nil {
		return false, fmt.Errorf("replace %s: %w", fullPath, err)
	}

	return true, nil
}

func (c Client) ReloadOrRestartServiceIfRunning(ctx context.Context, jail, service string) error {
	if jail == "" {
		return errors.New("jail is required")
	}
	if service == "" {
		return errors.New("service is required")
	}

	_, statusErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"cmd", jail, "service", service, "onestatus"},
		Timeout: 30 * time.Second,
	})
	if statusErr != nil {
		return nil
	}

	_, reloadErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"service", jail, service, "reload"},
		Timeout: 2 * time.Minute,
	})
	if reloadErr == nil {
		return nil
	}

	_, restartErr := c.Runner.Run(ctx, command.Spec{
		Path:    "bastille",
		Args:    []string{"service", jail, service, "restart"},
		Timeout: 2 * time.Minute,
	})
	if restartErr != nil {
		return fmt.Errorf("reload or restart service %s in jail %s: %w", service, jail, restartErr)
	}

	return nil
}

func (c Client) jailPath(jail, path string) (string, error) {
	cleanRoot := filepath.Clean(filepath.Join(c.Prefix, "jails", jail, "root"))
	relativePath := strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	fullPath := filepath.Join(cleanRoot, relativePath)

	rel, err := filepath.Rel(cleanRoot, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve path %s for jail %s: %w", path, jail, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes jail root", path)
	}

	return fullPath, nil
}
