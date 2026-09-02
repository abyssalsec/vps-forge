package engine

import (
	"testing"

	"github.com/abyssalsec/vps-forge/internal/config"
	"github.com/abyssalsec/vps-forge/internal/platform"
)

func TestBuildPlanDetectsServerChanges(t *testing.T) {
	cfg := config.ResolvedConfig{
		Profile: "minimal",

		Server: config.ServerConfig{
			Hostname: "app01",
			Timezone: "UTC",
		},
	}

	facts := platform.Facts{
		Hostname: "old-host",
		Timezone: "Europe/Warsaw",
	}

	plan := BuildPlan(
		cfg,
		facts,
	)

	if len(plan.Changes) != 2 {
		t.Fatalf(
			"expected 2 changes, got %d",
			len(plan.Changes),
		)
	}

	if plan.Changes[0].ID !=
		"server.hostname" {

		t.Fatalf(
			"unexpected first change %q",
			plan.Changes[0].ID,
		)
	}
}

func TestBuildPlanIsIdempotentWhenStateMatches(
	t *testing.T,
) {
	cfg := config.ResolvedConfig{
		Profile: "minimal",

		Server: config.ServerConfig{
			Hostname: "app01",
			Timezone: "UTC",
		},
	}

	facts := platform.Facts{
		Hostname: "app01",
		Timezone: "UTC",
	}

	plan := BuildPlan(
		cfg,
		facts,
	)

	if len(plan.Changes) != 0 {
		t.Fatalf(
			"expected zero changes, got %d",
			len(plan.Changes),
		)
	}
}
