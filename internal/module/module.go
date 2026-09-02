package module

import (
	"context"

	"github.com/abyssalsec/vps-forge/internal/config"
	"github.com/abyssalsec/vps-forge/internal/executor"
	"github.com/abyssalsec/vps-forge/internal/journal"
	"github.com/abyssalsec/vps-forge/internal/platform"
	"github.com/abyssalsec/vps-forge/internal/system"
)

type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

type Change struct {
	ID      string
	Module  string
	Action  string
	Risk    Risk
	Summary string
	Current string
	Desired string
}

type CheckStatus string

const (
	CheckOK   CheckStatus = "ok"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
)

type Check struct {
	ID      string
	Module  string
	Status  CheckStatus
	Message string
}

type Environment struct {
	Config   config.ResolvedConfig
	Facts    platform.Facts
	Executor executor.Executor

	ApplyID string
	Journal journal.Recorder
	Files   system.FileManager
}

type Module interface {
	Name() string

	Dependencies() []string

	Validate(
		context.Context,
		Environment,
	) error

	Plan(
		context.Context,
		Environment,
	) ([]Change, error)

	Apply(
		context.Context,
		Environment,
		[]Change,
	) error

	Verify(
		context.Context,
		Environment,
	) ([]Check, error)
}
