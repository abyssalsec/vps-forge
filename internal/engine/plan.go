package engine

import (
	"sort"

	"github.com/abyssalsec/vps-forge/internal/config"
	"github.com/abyssalsec/vps-forge/internal/module"
	"github.com/abyssalsec/vps-forge/internal/platform"
)

type Plan struct {
	Profile          string
	RequestedModules []string
	Changes          []module.Change
}

func requestedModules(
	cfg config.ResolvedConfig,
) []string {

	modules := []string{"platform"}

	if len(cfg.Users) > 0 {
		modules = append(
			modules,
			"users",
		)
	}

	if cfg.Security.SSH.Enabled {
		modules = append(
			modules,
			"ssh",
		)
	}

	if cfg.Security.Firewall.Enabled {
		modules = append(
			modules,
			"firewall",
		)
	}

	if cfg.Security.Fail2Ban {
		modules = append(
			modules,
			"fail2ban",
		)
	}

	if cfg.Security.UnattendedUpgrades {
		modules = append(
			modules,
			"updates",
		)
	}

	if cfg.Web.Nginx {
		modules = append(
			modules,
			"nginx",
		)
	}

	if cfg.Runtime.PHP.Enabled {
		modules = append(
			modules,
			"php",
		)
	}

	if cfg.Database.Engine != "" {
		modules = append(
			modules,
			"database",
		)
	}

	if cfg.Docker.Enabled {
		modules = append(
			modules,
			"docker",
		)
	}

	if cfg.Web.TLS.Enabled {
		modules = append(
			modules,
			"tls",
		)
	}

	if cfg.Backup.Enabled {
		modules = append(
			modules,
			"backup",
		)
	}

	sort.Strings(modules)

	return modules
}

func BuildPlan(
	cfg config.ResolvedConfig,
	facts platform.Facts,
) Plan {

	plan := Plan{
		Profile: cfg.Profile,

		RequestedModules: requestedModules(
			cfg,
		),
	}

	if cfg.Server.Hostname != "" &&
		cfg.Server.Hostname != facts.Hostname {

		plan.Changes = append(
			plan.Changes,
			module.Change{
				ID:      "server.hostname",
				Module:  "platform",
				Action:  "update",
				Risk:    module.RiskLow,
				Summary: "Update system hostname",
				Current: facts.Hostname,
				Desired: cfg.Server.Hostname,
			},
		)
	}

	if cfg.Server.Timezone != "" &&
		cfg.Server.Timezone != facts.Timezone {

		plan.Changes = append(
			plan.Changes,
			module.Change{
				ID:      "server.timezone",
				Module:  "platform",
				Action:  "update",
				Risk:    module.RiskLow,
				Summary: "Update system timezone",
				Current: facts.Timezone,
				Desired: cfg.Server.Timezone,
			},
		)
	}

	return plan
}
