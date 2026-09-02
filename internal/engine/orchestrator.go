package engine

import (
	"context"
	"fmt"

	"github.com/abyssalsec/vps-forge/internal/journal"
	"github.com/abyssalsec/vps-forge/internal/module"
)

type ExecutionPlan struct {
	Modules []string
	Changes []module.Change
}

type Verification struct {
	Checks []module.Check
}

func PlanModules(
	ctx context.Context,
	registry *Registry,
	environment module.Environment,
	requested []string,
) (ExecutionPlan, error) {

	ordered, err := registry.Resolve(
		requested,
	)

	if err != nil {
		return ExecutionPlan{}, err
	}

	result := ExecutionPlan{}

	for _, current := range ordered {
		if err := current.Validate(
			ctx,
			environment,
		); err != nil {

			return ExecutionPlan{}, fmt.Errorf(
				"validate module %s: %w",
				current.Name(),
				err,
			)
		}

		changes, err := current.Plan(
			ctx,
			environment,
		)

		if err != nil {
			return ExecutionPlan{}, fmt.Errorf(
				"plan module %s: %w",
				current.Name(),
				err,
			)
		}

		result.Modules = append(
			result.Modules,
			current.Name(),
		)

		for _, change := range changes {
			change.Module = current.Name()

			result.Changes = append(
				result.Changes,
				change,
			)
		}
	}

	return result, nil
}

func recordJournal(
	environment module.Environment,
	entry journal.Entry,
) error {

	if environment.Journal == nil {
		return nil
	}

	return environment.Journal.Record(
		entry,
	)
}

func ApplyModules(
	ctx context.Context,
	registry *Registry,
	environment module.Environment,
	plan ExecutionPlan,
) error {

	for _, name := range plan.Modules {
		current, exists := registry.Get(name)

		if !exists {
			return fmt.Errorf(
				"planned module %q is not registered",
				name,
			)
		}

		var changes []module.Change

		for _, change := range plan.Changes {
			if change.Module == name {
				changes = append(
					changes,
					change,
				)
			}
		}

		if len(changes) == 0 {
			continue
		}

		if err := recordJournal(
			environment,
			journal.Entry{
				Event:  "module.apply",
				Module: name,
				Status: "started",
				Message: fmt.Sprintf(
					"%d change(s)",
					len(changes),
				),
			},
		); err != nil {
			return fmt.Errorf(
				"record module apply start: %w",
				err,
			)
		}

		if err := current.Apply(
			ctx,
			environment,
			changes,
		); err != nil {

			recordErr := recordJournal(
				environment,
				journal.Entry{
					Event:   "module.apply",
					Module:  name,
					Status:  "failed",
					Message: err.Error(),
				},
			)

			if recordErr != nil {
				return fmt.Errorf(
					"apply module %s: %w; journal error: %v",
					name,
					err,
					recordErr,
				)
			}

			return fmt.Errorf(
				"apply module %s: %w",
				name,
				err,
			)
		}

		if err := recordJournal(
			environment,
			journal.Entry{
				Event:  "module.apply",
				Module: name,
				Status: "ok",
			},
		); err != nil {
			return fmt.Errorf(
				"record module apply completion: %w",
				err,
			)
		}
	}

	return nil
}

func VerifyModules(
	ctx context.Context,
	registry *Registry,
	environment module.Environment,
	requested []string,
) (Verification, error) {

	ordered, err := registry.Resolve(
		requested,
	)

	if err != nil {
		return Verification{}, err
	}

	result := Verification{}

	for _, current := range ordered {
		checks, err := current.Verify(
			ctx,
			environment,
		)

		if err != nil {
			return Verification{}, fmt.Errorf(
				"verify module %s: %w",
				current.Name(),
				err,
			)
		}

		for _, check := range checks {
			check.Module = current.Name()

			result.Checks = append(
				result.Checks,
				check,
			)
		}
	}

	return result, nil
}
