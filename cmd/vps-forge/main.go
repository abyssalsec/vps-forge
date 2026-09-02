package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/abyssalsec/vps-forge/internal/config"
	"github.com/abyssalsec/vps-forge/internal/engine"
	"github.com/abyssalsec/vps-forge/internal/platform"
)

var version = "dev"

func usage() {
	fmt.Print(`VPS Forge

Production-oriented Linux VPS provisioning.

Usage:
  vps-forge audit
  vps-forge plan -c forge.yaml
  vps-forge version
  vps-forge help

Commands:
  audit      Inspect the current VPS platform
  plan       Validate configuration and calculate safe planned changes
  version    Show VPS Forge version
  help       Show this help

Stage 1 is read-only. No system changes are performed.
`)
}

func runAudit(args []string) int {
	flags := flag.NewFlagSet(
		"audit",
		flag.ContinueOnError,
	)

	if err := flags.Parse(args); err != nil {
		return 1
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(
			os.Stderr,
			"ERROR: audit does not accept positional arguments",
		)
		return 1
	}

	facts, err := platform.Detect()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"ERROR: %v\n",
			err,
		)
		return 1
	}

	fmt.Println("VPS Forge Audit")
	fmt.Println()

	fmt.Printf(
		"OS:           %s\n",
		facts.PrettyName,
	)

	fmt.Printf(
		"Distribution: %s\n",
		facts.ID,
	)

	fmt.Printf(
		"Version:      %s\n",
		facts.VersionID,
	)

	fmt.Printf(
		"Codename:     %s\n",
		facts.VersionCodename,
	)

	fmt.Printf(
		"Kernel:       %s\n",
		facts.Kernel,
	)

	fmt.Printf(
		"Architecture: %s\n",
		facts.Architecture,
	)

	fmt.Printf(
		"Hostname:     %s\n",
		facts.Hostname,
	)

	fmt.Printf(
		"Timezone:     %s\n",
		facts.Timezone,
	)

	fmt.Printf(
		"systemd:      %t\n",
		facts.Systemd,
	)

	fmt.Printf(
		"Running root: %t\n",
		facts.IsRoot(),
	)

	fmt.Printf(
		"Supported:    %t\n",
		facts.Supported(),
	)

	return 0
}

func runPlan(args []string) int {
	flags := flag.NewFlagSet(
		"plan",
		flag.ContinueOnError,
	)

	configPath := flags.String(
		"c",
		"",
		"configuration file",
	)

	if err := flags.Parse(args); err != nil {
		return 1
	}

	if flags.NArg() != 0 {
		fmt.Fprintln(
			os.Stderr,
			"ERROR: plan does not accept positional arguments",
		)
		return 1
	}

	if *configPath == "" {
		fmt.Fprintln(
			os.Stderr,
			"ERROR: plan requires -c PATH",
		)
		return 1
	}

	cfg, err := config.LoadResolved(
		*configPath,
	)

	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"ERROR: %v\n",
			err,
		)
		return 1
	}

	facts, err := platform.Detect()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"ERROR: %v\n",
			err,
		)
		return 1
	}

	if !facts.Supported() {
		fmt.Fprintf(
			os.Stderr,
			"ERROR: unsupported platform: %s %s\n",
			facts.ID,
			facts.VersionID,
		)
		return 1
	}

	plan := engine.BuildPlan(
		cfg,
		facts,
	)

	fmt.Println("VPS Forge Plan")
	fmt.Println()

	fmt.Printf(
		"Platform: %s\n",
		facts.PrettyName,
	)

	fmt.Printf(
		"Profile:  %s\n",
		plan.Profile,
	)

	fmt.Printf(
		"Modules:  %s\n",
		strings.Join(
			plan.RequestedModules,
			", ",
		),
	)

	fmt.Println()
	fmt.Println(
		"Planning scope: server identity",
	)
	fmt.Println()

	if len(plan.Changes) == 0 {
		fmt.Println(
			"No server identity changes required.",
		)

		fmt.Println()
		fmt.Println("Plan: 0 changes")

		return 0
	}

	fmt.Printf(
		"%-10s %-8s %-8s %-24s %s\n",
		"MODULE",
		"ACTION",
		"RISK",
		"ID",
		"CHANGE",
	)

	for _, change := range plan.Changes {
		fmt.Printf(
			"%-10s %-8s %-8s %-24s %s\n",
			change.Module,
			strings.ToUpper(change.Action),
			strings.ToUpper(string(change.Risk)),
			change.ID,
			change.Summary,
		)

		fmt.Printf(
			"  current: %q\n",
			change.Current,
		)

		fmt.Printf(
			"  desired: %q\n",
			change.Desired,
		)
	}

	fmt.Println()

	fmt.Printf(
		"Plan: %d change(s)\n",
		len(plan.Changes),
	)

	fmt.Println(
		"No changes were applied.",
	)

	return 0
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var rc int

	switch os.Args[1] {
	case "audit":
		rc = runAudit(os.Args[2:])

	case "plan":
		rc = runPlan(os.Args[2:])

	case "version", "--version":
		fmt.Println(version)
		rc = 0

	case "help", "--help", "-h":
		usage()
		rc = 0

	default:
		fmt.Fprintf(
			os.Stderr,
			"ERROR: unknown command %q\n\n",
			os.Args[1],
		)

		usage()
		rc = 1
	}

	os.Exit(rc)
}
