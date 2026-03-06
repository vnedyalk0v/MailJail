package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/apply"
	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/health"
	"github.com/vnedyalk0v/mailjail/internal/host/command"
	"github.com/vnedyalk0v/mailjail/internal/host/preflight"
	"github.com/vnedyalk0v/mailjail/internal/plan"
	"github.com/vnedyalk0v/mailjail/internal/schema"
	"github.com/vnedyalk0v/mailjail/internal/state"
)

const defaultConfigPath = "mailjail.yml"

var version = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "validate":
		return runValidate(ctx, logger, args[1:], stdout, stderr)
	case "plan":
		return runPlan(ctx, logger, args[1:], stdout, stderr)
	case "apply":
		return runApply(ctx, logger, args[1:], stdout, stderr)
	case "status":
		return runStatus(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var outputPath string
	var force bool

	fs.StringVar(&outputPath, "f", defaultConfigPath, "path to write the starter config")
	fs.BoolVar(&force, "force", false, "overwrite the target file if it already exists")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if !force {
		if _, err := os.Stat(outputPath); err == nil {
			fmt.Fprintf(stderr, "%s already exists; use --force to overwrite\n", outputPath)
			return 1
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "failed to check %s: %v\n", outputPath, err)
			return 1
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil && filepath.Dir(outputPath) != "." {
		fmt.Fprintf(stderr, "failed to create parent directory for %s: %v\n", outputPath, err)
		return 1
	}

	if err := os.WriteFile(outputPath, []byte(config.Template()), 0o640); err != nil {
		fmt.Fprintf(stderr, "failed to write %s: %v\n", outputPath, err)
		return 1
	}

	fmt.Fprintf(stdout, "wrote starter config to %s\n", outputPath)
	return 0
}

func runValidate(ctx context.Context, logger *slog.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string

	fs.StringVar(&configPath, "c", defaultConfigPath, "path to the MailJail config file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, validationFailed := loadAndValidateConfig(logger, configPath, stderr)
	if validationFailed {
		return 1
	}

	results := preflight.Run(ctx, command.OSRunner{}, cfg)
	printPreflight(stdout, results)

	if preflight.HasBlockingFailures(results) {
		return 1
	}

	fmt.Fprintf(stdout, "configuration for stack %q is valid\n", cfg.Metadata.Name)
	return 0
}

func runPlan(ctx context.Context, logger *slog.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string

	fs.StringVar(&configPath, "c", defaultConfigPath, "path to the MailJail config file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, validationFailed := loadAndValidateConfig(logger, configPath, stderr)
	if validationFailed {
		return 1
	}

	results := preflight.Run(ctx, command.OSRunner{}, cfg)
	printPreflight(stdout, results)

	pl, err := plan.Build(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "failed to build plan: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "plan generated at %s\n", pl.GeneratedAt.Format(time.RFC3339))
	for idx, action := range pl.Actions {
		fmt.Fprintf(stdout, "%d. [%s] %s\n", idx+1, action.Type, action.Summary)
		if len(action.CommandPreview) > 0 {
			fmt.Fprintf(stdout, "   command: %s\n", action.CommandString())
		}
	}

	return 0
}

func runApply(ctx context.Context, logger *slog.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	var stateDir string

	fs.StringVar(&configPath, "c", defaultConfigPath, "path to the MailJail config file")
	fs.StringVar(&stateDir, "state-dir", state.DefaultDir(), "directory used to store MailJail local state")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, validationFailed := loadAndValidateConfig(logger, configPath, stderr)
	if validationFailed {
		return 1
	}

	results := preflight.Run(ctx, command.OSRunner{}, cfg)
	printPreflight(stdout, results)
	if preflight.HasBlockingFailures(results) {
		return 1
	}

	pl, err := plan.Build(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "failed to build plan: %v\n", err)
		return 1
	}

	applier := apply.New(command.OSRunner{}, logger)
	execResults, err := applier.Execute(ctx, pl)
	for _, result := range execResults {
		status := "ok"
		if result.Skipped {
			status = "skipped"
		}
		if result.Err != "" {
			status = "error"
		}
		fmt.Fprintf(stdout, "[%s] %s\n", status, result.Action.Summary)
		if result.Err != "" {
			fmt.Fprintf(stdout, "  %s\n", result.Err)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "apply failed: %v\n", err)
	}

	record := state.ApplyRecord{
		ConfigPath: configPath,
		Plan:       state.PlanSnapshotFromPlan(pl),
		Results:    state.ResultsSnapshotFromApply(execResults),
		Health:     health.DefaultSnapshot(cfg),
	}

	if saveErr := state.SaveApply(stateDir, record); saveErr != nil {
		fmt.Fprintf(stderr, "apply completed but failed to write state: %v\n", saveErr)
		if err == nil {
			return 1
		}
	}

	if err != nil {
		return 1
	}

	fmt.Fprintf(stdout, "apply completed successfully\n")
	return 0
}

func runStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var stateDir string

	fs.StringVar(&stateDir, "state-dir", state.DefaultDir(), "directory used to store MailJail local state")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	record, err := state.LoadLatest(stateDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "no apply state found in %s\n", stateDir)
			return 0
		}
		fmt.Fprintf(stderr, "failed to load state from %s: %v\n", stateDir, err)
		return 1
	}

	fmt.Fprintf(stdout, "last apply: %s\n", record.AppliedAt.Format(time.RFC3339))
	fmt.Fprintf(stdout, "config: %s\n", record.ConfigPath)
	fmt.Fprintf(stdout, "planned actions: %d\n", len(record.Plan.Actions))
	for _, result := range record.Results {
		fmt.Fprintf(stdout, "- [%s] %s\n", result.Status, result.Summary)
	}
	for _, module := range record.Health.Modules {
		fmt.Fprintf(stdout, "- module=%s enabled=%t status=%s\n", module.Name, module.Enabled, module.Status)
	}

	return 0
}

func loadAndValidateConfig(logger *slog.Logger, configPath string, stderr io.Writer) (*config.Config, bool) {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "failed to load %s: %v\n", configPath, err)
		return nil, true
	}

	issues := schema.Validate(cfg)
	if len(issues) > 0 {
		logger.Warn("configuration validation failed", "issues", len(issues))
		for _, issue := range issues {
			fmt.Fprintf(stderr, "%s: %s\n", issue.Path, issue.Message)
		}
		return nil, true
	}

	return cfg, false
}

func printPreflight(stdout io.Writer, results []preflight.Result) {
	for _, result := range results {
		fmt.Fprintf(stdout, "preflight [%s] %s: %s\n", result.Status, result.Name, result.Message)
	}
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, `mailjail manages a Bastille-backed FreeBSD mail stack.

Usage:
  mailjail <command> [options]

Commands:
  version   Print the CLI version
  init      Write a starter config file
  validate  Validate configuration and host prerequisites
  plan      Print the current execution plan
  apply     Execute the first provisioning slice
  status    Show the latest recorded apply status`)
}
