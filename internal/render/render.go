package render

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/vnedyalk0v/mailjail/internal/config"
)

type File struct {
	Path    string
	Mode    fs.FileMode
	Content string
}

func FilesForModule(cfg *config.Config, module string) ([]File, error) {
	switch module {
	case "redis":
		return redisFiles(cfg), nil
	case "rspamd":
		return rspamdFiles(cfg)
	case "postfix":
		return postfixFiles(cfg)
	default:
		return nil, fmt.Errorf("no renderer defined for module %q", module)
	}
}

func redisFiles(cfg *config.Config) []File {
	redisIP := cfg.Modules["redis"].IP4

	content := fmt.Sprintf(`# Managed by MailJail.
bind 127.0.0.1 %s
protected-mode yes
port 6379
tcp-backlog 128
timeout 0
tcp-keepalive 300
daemonize no
supervised no
loglevel notice
databases 16
save 900 1
save 300 10
save 60 10000
appendonly yes
appendfsync everysec
dir /var/db/redis/
`, redisIP)

	return []File{
		{
			Path:    "/usr/local/etc/redis.conf",
			Mode:    0o640,
			Content: content,
		},
	}
}

func rspamdFiles(cfg *config.Config) ([]File, error) {
	rspamdIP := cfg.Modules["rspamd"].IP4
	redisIP := cfg.Modules["redis"].IP4
	if rspamdIP == "" {
		return nil, fmt.Errorf("rspamd module IP is required for rendering")
	}
	if redisIP == "" {
		return nil, fmt.Errorf("redis module IP is required for rspamd rendering")
	}

	return []File{
		{
			Path: "/usr/local/etc/rspamd/local.d/worker-proxy.inc",
			Mode: 0o640,
			Content: fmt.Sprintf(`# Managed by MailJail.
bind_socket = "%s:11332";
milter = yes;
timeout = 120s;
upstream "local" {
  default = yes;
  self_scan = yes;
}
`, rspamdIP),
		},
		{
			Path: "/usr/local/etc/rspamd/local.d/worker-controller.inc",
			Mode: 0o640,
			Content: `# Managed by MailJail.
bind_socket = "127.0.0.1:11334";
secure_ip = "127.0.0.1";
`,
		},
		{
			Path: "/usr/local/etc/rspamd/local.d/worker-normal.inc",
			Mode: 0o640,
			Content: `# Managed by MailJail.
enabled = false;
`,
		},
		{
			Path: "/usr/local/etc/rspamd/local.d/redis.conf",
			Mode: 0o640,
			Content: fmt.Sprintf(`# Managed by MailJail.
servers = "%s:6379";
expand_keys = true;
timeout = 1s;
`, redisIP),
		},
	}, nil
}

func postfixFiles(cfg *config.Config) ([]File, error) {
	rspamdIP := cfg.Modules["rspamd"].IP4
	if rspamdIP == "" {
		return nil, fmt.Errorf("rspamd module IP is required for postfix rendering")
	}

	domain := strings.TrimSpace(cfg.Network.Domain)
	hostname := strings.TrimSpace(cfg.Host.Hostname)
	if domain == "" || hostname == "" {
		return nil, fmt.Errorf("host.hostname and network.domain are required for postfix rendering")
	}

	content := fmt.Sprintf(`# Managed by MailJail.
compatibility_level = 3.6
myhostname = %s
mydomain = %s
myorigin = $mydomain
inet_interfaces = all
inet_protocols = ipv4
mynetworks_style = host
smtpd_banner = $myhostname ESMTP
biff = no
append_dot_mydomain = no
readme_directory = no
disable_vrfy_command = yes
smtputf8_enable = yes
smtp_tls_security_level = may
smtp_dns_support_level = enabled
smtpd_relay_restrictions = permit_mynetworks, reject_unauth_destination
smtpd_recipient_restrictions = reject_unauth_destination
milter_default_action = accept
milter_protocol = 6
smtpd_milters = inet:%s:11332
non_smtpd_milters = inet:%s:11332
`, hostname, domain, rspamdIP, rspamdIP)

	return []File{
		{
			Path:    "/usr/local/etc/postfix/main.cf",
			Mode:    0o640,
			Content: content,
		},
	}, nil
}
