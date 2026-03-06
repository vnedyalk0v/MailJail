package apply

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/config"
	"github.com/vnedyalk0v/mailjail/internal/host/bastille"
	"github.com/vnedyalk0v/mailjail/internal/host/command"
	"github.com/vnedyalk0v/mailjail/internal/host/pf"
	"github.com/vnedyalk0v/mailjail/internal/host/zfs"
	"github.com/vnedyalk0v/mailjail/internal/plan"
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
