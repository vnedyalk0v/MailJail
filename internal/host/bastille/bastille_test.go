package bastille

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncFileWritesInsideJailRootAndIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	client := Client{Prefix: tempDir}

	jailRoot := filepath.Join(tempDir, "jails", "mx1-redis", "root")
	if err := os.MkdirAll(jailRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	changed, err := client.SyncFile("mx1-redis", "/usr/local/etc/redis.conf", []byte("protected-mode yes\n"), 0o640)
	if err != nil {
		t.Fatalf("SyncFile returned error: %v", err)
	}
	if !changed {
		t.Fatalf("expected first SyncFile call to report changed=true")
	}

	path := filepath.Join(jailRoot, "usr/local/etc/redis.conf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if got, want := string(data), "protected-mode yes\n"; got != want {
		t.Fatalf("expected file content %q, got %q", want, got)
	}

	changed, err = client.SyncFile("mx1-redis", "/usr/local/etc/redis.conf", []byte("protected-mode yes\n"), 0o640)
	if err != nil {
		t.Fatalf("second SyncFile returned error: %v", err)
	}
	if changed {
		t.Fatalf("expected identical SyncFile call to report changed=false")
	}
}

func TestSyncFileRejectsRelativePaths(t *testing.T) {
	client := Client{Prefix: t.TempDir()}

	if _, err := client.SyncFile("mx1-redis", "usr/local/etc/redis.conf", []byte("x"), 0o640); err == nil {
		t.Fatalf("expected SyncFile to reject relative paths")
	}
}
