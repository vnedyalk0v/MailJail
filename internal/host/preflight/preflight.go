package preflight

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/host/command"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Result struct {
	Name    string
	Status  Status
	Message string
}

func Run(ctx context.Context, runner command.Runner, cfg *config.Config) []Result {
	_ = ctx
	_ = runner
	_ = cfg

	results := []Result{
		checkOS(),
		checkTool("bastille", runtime.GOOS == "freebsd"),
		checkTool("zfs", runtime.GOOS == "freebsd"),
		checkTool("pfctl", runtime.GOOS == "freebsd"),
	}

	return results
}

func HasBlockingFailures(results []Result) bool {
	for _, result := range results {
		if result.Status == StatusFail {
			return true
		}
	}
	return false
}

func checkOS() Result {
	if runtime.GOOS == "freebsd" {
		return Result{
			Name:    "os",
			Status:  StatusPass,
			Message: "running on FreeBSD",
		}
	}

	return Result{
		Name:    "os",
		Status:  StatusWarn,
		Message: "running on a non-FreeBSD host; host provisioning commands are intended for FreeBSD targets",
	}
}

func checkTool(name string, require bool) Result {
	if _, err := exec.LookPath(name); err == nil {
		return Result{
			Name:    name,
			Status:  StatusPass,
			Message: "tool is available",
		}
	}

	status := StatusWarn
	message := "tool is not available in PATH"
	if require {
		status = StatusFail
		message = "tool is required on a FreeBSD target host"
	}

	return Result{
		Name:    name,
		Status:  status,
		Message: message,
	}
}
