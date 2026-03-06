package render

import (
	"strings"
	"testing"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

func TestFilesForModuleRedisUsesProtectedModeAndPersistence(t *testing.T) {
	cfg := testConfig()

	files, err := FilesForModule(cfg, "redis")
	if err != nil {
		t.Fatalf("FilesForModule returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 redis config file, got %d", len(files))
	}

	content := files[0].Content
	for _, want := range []string{
		"protected-mode yes",
		"bind 127.0.0.1 10.77.0.13",
		"appendonly yes",
		"appendfsync everysec",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected redis config to contain %q", want)
		}
	}
}

func TestFilesForModuleRspamdBindsControllerLocallyAndProxyPublicly(t *testing.T) {
	cfg := testConfig()

	files, err := FilesForModule(cfg, "rspamd")
	if err != nil {
		t.Fatalf("FilesForModule returned error: %v", err)
	}

	joined := joinContents(files)
	for _, want := range []string{
		`bind_socket = "10.77.0.12:11332";`,
		`self_scan = yes;`,
		`bind_socket = "127.0.0.1:11334";`,
		`secure_ip = "127.0.0.1";`,
		`servers = "10.77.0.13:6379";`,
		`enabled = false;`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected rspamd config to contain %q", want)
		}
	}
}

func TestFilesForModulePostfixUsesRspamdMilterAndSafeDefaults(t *testing.T) {
	cfg := testConfig()

	files, err := FilesForModule(cfg, "postfix")
	if err != nil {
		t.Fatalf("FilesForModule returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 postfix config file, got %d", len(files))
	}

	content := files[0].Content
	for _, want := range []string{
		"myhostname = mx1.example.com",
		"mydomain = example.com",
		"disable_vrfy_command = yes",
		"smtp_tls_security_level = may",
		"smtpd_milters = inet:10.77.0.12:11332",
		"non_smtpd_milters = inet:10.77.0.12:11332",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected postfix config to contain %q", want)
		}
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Host: config.Host{
			Hostname: "mx1.example.com",
		},
		Network: config.Network{
			Domain: "example.com",
		},
		Modules: map[string]config.ModuleConfig{
			"postfix": {Enabled: true, IP4: "10.77.0.10"},
			"dovecot": {Enabled: true, IP4: "10.77.0.11"},
			"rspamd":  {Enabled: true, IP4: "10.77.0.12"},
			"redis":   {Enabled: true, IP4: "10.77.0.13"},
		},
	}
}

func joinContents(files []File) string {
	parts := make([]string, 0, len(files))
	for _, file := range files {
		parts = append(parts, file.Content)
	}
	return strings.Join(parts, "\n")
}
