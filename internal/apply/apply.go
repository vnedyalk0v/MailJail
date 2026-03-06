package apply

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/host/bastille"
	"github.com/vnedyalk0v/mailjail/internal/host/command"
	"github.com/vnedyalk0v/mailjail/internal/host/pf"
	"github.com/vnedyalk0v/mailjail/internal/host/zfs"
	"github.com/vnedyalk0v/mailjail/internal/plan"
	"github.com/vnedyalk0v/mailjail/internal/render"
)

type Result struct {
	Action    plan.Action
	Status    string
	StartedAt time.Time
	EndedAt   time.Time
	Skipped   bool
	Err       string
}

type Applier struct {
	bastille bastille.Client
	cfg      *config.Config
	pf       pf.Client
	zfs      zfs.Client
	logger   *slog.Logger
}

func New(runner command.Runner, logger *slog.Logger, cfg *config.Config) *Applier {
	return &Applier{
		bastille: bastille.New(runner),
		cfg:      cfg,
		pf:       pf.New(runner),
		zfs:      zfs.New(runner),
		logger:   logger,
	}
}

func (a *Applier) Execute(ctx context.Context, pl *plan.Plan) ([]Result, error) {
	results := make([]Result, 0, len(pl.Actions))

	for _, action := range pl.Actions {
		startedAt := time.Now().UTC()
		a.logger.Info("executing action", "type", action.Type, "target", action.Target)

		result := Result{
			Action:    action,
			Status:    "ok",
			StartedAt: startedAt,
		}

		err := a.executeAction(ctx, action)
		result.EndedAt = time.Now().UTC()

		if err != nil {
			result.Status = "error"
			result.Err = err.Error()
			results = append(results, result)
			return results, fmt.Errorf("execute %s for %s: %w", action.Type, action.Target, err)
		}

		results = append(results, result)
	}

	return results, nil
}

func (a *Applier) executeAction(ctx context.Context, action plan.Action) error {
	switch action.Type {
	case plan.ActionEnsureDataset:
		return a.zfs.EnsureDataset(ctx, action.Target)
	case plan.ActionEnsureBastilleSetup:
		return a.bastille.EnsureRelease(ctx, action.Target)
	case plan.ActionEnsurePFAnchor:
		return a.pf.EnsureAnchor(ctx, a.cfg)
	case plan.ActionEnsureBaseJail:
		return a.ensureBaseJail(ctx, action)
	case plan.ActionInstallPackages:
		return a.ensurePackages(ctx, action)
	case plan.ActionEnableService:
		return a.enableService(ctx, action)
	case plan.ActionRenderModuleConfig:
		return a.renderModuleConfig(ctx, action)
	case plan.ActionStartService:
		return a.startService(ctx, action)
	default:
		return fmt.Errorf("unsupported action type %s", action.Type)
	}
}

func (a *Applier) ensureBaseJail(ctx context.Context, action plan.Action) error {
	if len(action.CommandPreview) < 5 {
		return fmt.Errorf("invalid base jail action preview")
	}

	ipCIDR := action.CommandPreview[4]
	if _, err := netip.ParsePrefix(ipCIDR); err != nil {
		return fmt.Errorf("invalid base jail CIDR %q: %w", ipCIDR, err)
	}

	iface := ""
	if len(action.CommandPreview) > 5 {
		iface = action.CommandPreview[5]
	}

	return a.bastille.EnsureJail(ctx, action.Target, action.CommandPreview[3], ipCIDR, iface)
}

func (a *Applier) ensurePackages(ctx context.Context, action plan.Action) error {
	jail := action.Target
	if metadataJail := action.Metadata["jail"]; metadataJail != "" {
		jail = metadataJail
	}
	if jail == "" {
		return fmt.Errorf("package action requires jail metadata")
	}
	if len(action.Items) == 0 {
		return fmt.Errorf("package action requires at least one package")
	}

	return a.bastille.InstallPackages(ctx, jail, action.Items)
}

func (a *Applier) enableService(ctx context.Context, action plan.Action) error {
	jail := action.Metadata["jail"]
	rcvar := action.Metadata["rcvar"]
	if jail == "" || rcvar == "" {
		return fmt.Errorf("enable service action requires jail and rcvar metadata")
	}

	return a.bastille.SetRC(ctx, jail, rcvar)
}

func (a *Applier) renderModuleConfig(ctx context.Context, action plan.Action) error {
	module := action.Metadata["module"]
	jail := action.Metadata["jail"]
	service := action.Metadata["service"]
	if module == "" || jail == "" || service == "" {
		return fmt.Errorf("render config action requires module, jail, and service metadata")
	}

	files, err := render.FilesForModule(a.cfg, module)
	if err != nil {
		return err
	}

	changedAny := false
	for _, file := range files {
		changed, syncErr := a.bastille.SyncFile(jail, file.Path, []byte(file.Content), file.Mode)
		if syncErr != nil {
			return fmt.Errorf("sync %s config file %s: %w", module, file.Path, syncErr)
		}
		if changed {
			changedAny = true
			a.logger.Info("updated rendered config", "module", module, "jail", jail, "path", file.Path, "mode", strconv.FormatUint(uint64(file.Mode.Perm()), 8))
		}
	}

	if !changedAny {
		a.logger.Info("rendered config already current", "module", module, "jail", jail)
		return nil
	}

	return a.bastille.ReloadOrRestartServiceIfRunning(ctx, jail, service)
}

func (a *Applier) startService(ctx context.Context, action plan.Action) error {
	jail := action.Metadata["jail"]
	service := action.Metadata["service"]
	if jail == "" || service == "" {
		return fmt.Errorf("start service action requires jail and service metadata")
	}

	return a.bastille.EnsureServiceStarted(ctx, jail, service)
}
