package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vnedyalk0v/mailjail/internal/apply"
	"github.com/vnedyalk0v/mailjail/internal/health"
	"github.com/vnedyalk0v/mailjail/internal/plan"
)

type PlanSnapshot struct {
	GeneratedAt time.Time     `json:"generatedAt"`
	ConfigName  string        `json:"configName"`
	Actions     []plan.Action `json:"actions"`
}

type ResultSnapshot struct {
	Type      plan.ActionType `json:"type"`
	Target    string          `json:"target"`
	Summary   string          `json:"summary"`
	Status    string          `json:"status"`
	StartedAt time.Time       `json:"startedAt"`
	EndedAt   time.Time       `json:"endedAt"`
	Error     string          `json:"error,omitempty"`
}

type ApplyRecord struct {
	AppliedAt  time.Time            `json:"appliedAt"`
	ConfigPath string               `json:"configPath"`
	Plan       PlanSnapshot         `json:"plan"`
	Results    []ResultSnapshot     `json:"results"`
	Health     health.StateSnapshot `json:"health"`
}

func DefaultDir() string {
	if runtime.GOOS == "freebsd" {
		return "/var/db/mailjail"
	}
	return ".mailjail/state"
}

func SaveApply(dir string, record ApplyRecord) error {
	record.AppliedAt = time.Now().UTC()

	if err := os.MkdirAll(filepath.Join(dir, "history"), 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal apply record: %w", err)
	}

	latestPath := filepath.Join(dir, "latest.json")
	if err := os.WriteFile(latestPath, data, 0o640); err != nil {
		return fmt.Errorf("write latest state: %w", err)
	}

	historyPath := filepath.Join(dir, "history", record.AppliedAt.Format("20060102T150405Z")+".json")
	if err := os.WriteFile(historyPath, data, 0o640); err != nil {
		return fmt.Errorf("write history state: %w", err)
	}

	return nil
}

func LoadLatest(dir string) (*ApplyRecord, error) {
	data, err := os.ReadFile(filepath.Join(dir, "latest.json"))
	if err != nil {
		return nil, err
	}

	var record ApplyRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &record, nil
}

func PlanSnapshotFromPlan(pl *plan.Plan) PlanSnapshot {
	return PlanSnapshot{
		GeneratedAt: pl.GeneratedAt,
		ConfigName:  pl.ConfigName,
		Actions:     pl.Actions,
	}
}

func ResultsSnapshotFromApply(results []apply.Result) []ResultSnapshot {
	snapshots := make([]ResultSnapshot, 0, len(results))
	for _, result := range results {
		snapshots = append(snapshots, ResultSnapshot{
			Type:      result.Action.Type,
			Target:    result.Action.Target,
			Summary:   result.Action.Summary,
			Status:    result.Status,
			StartedAt: result.StartedAt,
			EndedAt:   result.EndedAt,
			Error:     result.Err,
		})
	}
	return snapshots
}
