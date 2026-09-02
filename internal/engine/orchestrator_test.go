package engine

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/abyssalsec/vps-forge/internal/module"
)

type testModule struct {
	name         string
	dependencies []string
	events       *[]string
	change       bool
}

func (m *testModule) Name() string {
	return m.name
}

func (m *testModule) Dependencies() []string {
	return append(
		[]string(nil),
		m.dependencies...,
	)
}

func (m *testModule) Validate(
	_ context.Context,
	_ module.Environment,
) error {

	*m.events = append(
		*m.events,
		"validate:"+m.name,
	)

	return nil
}

func (m *testModule) Plan(
	_ context.Context,
	_ module.Environment,
) ([]module.Change, error) {

	*m.events = append(
		*m.events,
		"plan:"+m.name,
	)

	if !m.change {
		return nil, nil
	}

	return []module.Change{
		{
			ID:      m.name + ".change",
			Action:  "update",
			Risk:    module.RiskLow,
			Summary: "test change",
		},
	}, nil
}

func (m *testModule) Apply(
	_ context.Context,
	_ module.Environment,
	changes []module.Change,
) error {

	if len(changes) == 0 {
		return fmt.Errorf(
			"apply called without changes",
		)
	}

	*m.events = append(
		*m.events,
		"apply:"+m.name,
	)

	return nil
}

func (m *testModule) Verify(
	_ context.Context,
	_ module.Environment,
) ([]module.Check, error) {

	*m.events = append(
		*m.events,
		"verify:"+m.name,
	)

	return []module.Check{
		{
			ID:      m.name + ".ok",
			Status:  module.CheckOK,
			Message: "test module verified",
		},
	}, nil
}

func TestRegistryResolvesDependencies(t *testing.T) {
	events := []string{}

	registry, err := NewRegistry(
		&testModule{
			name:   "base",
			events: &events,
		},
		&testModule{
			name:         "web",
			dependencies: []string{"base"},
			events:       &events,
		},
		&testModule{
			name:         "tls",
			dependencies: []string{"web"},
			events:       &events,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	ordered, err := registry.Resolve(
		[]string{"tls"},
	)

	if err != nil {
		t.Fatal(err)
	}

	var names []string

	for _, current := range ordered {
		names = append(
			names,
			current.Name(),
		)
	}

	expected := []string{
		"base",
		"web",
		"tls",
	}

	if !reflect.DeepEqual(
		names,
		expected,
	) {
		t.Fatalf(
			"expected %v, got %v",
			expected,
			names,
		)
	}
}

func TestRegistryRejectsDependencyCycle(
	t *testing.T,
) {
	events := []string{}

	registry, err := NewRegistry(
		&testModule{
			name:         "a",
			dependencies: []string{"b"},
			events:       &events,
		},
		&testModule{
			name:         "b",
			dependencies: []string{"a"},
			events:       &events,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := registry.Resolve(
		[]string{"a"},
	); err == nil {

		t.Fatal(
			"expected dependency cycle error",
		)
	}
}

func TestPlanApplyVerifyOrder(t *testing.T) {
	events := []string{}

	registry, err := NewRegistry(
		&testModule{
			name:   "base",
			events: &events,
			change: true,
		},
		&testModule{
			name:         "web",
			dependencies: []string{"base"},
			events:       &events,
			change:       true,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	environment := module.Environment{}

	plan, err := PlanModules(
		ctx,
		registry,
		environment,
		[]string{"web"},
	)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(
		plan.Modules,
		[]string{"base", "web"},
	) {
		t.Fatalf(
			"unexpected module order: %v",
			plan.Modules,
		)
	}

	if len(plan.Changes) != 2 {
		t.Fatalf(
			"expected 2 changes, got %d",
			len(plan.Changes),
		)
	}

	if err := ApplyModules(
		ctx,
		registry,
		environment,
		plan,
	); err != nil {
		t.Fatal(err)
	}

	verification, err := VerifyModules(
		ctx,
		registry,
		environment,
		[]string{"web"},
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(verification.Checks) != 2 {
		t.Fatalf(
			"expected 2 checks, got %d",
			len(verification.Checks),
		)
	}

	expectedEvents := []string{
		"validate:base",
		"plan:base",
		"validate:web",
		"plan:web",
		"apply:base",
		"apply:web",
		"verify:base",
		"verify:web",
	}

	if !reflect.DeepEqual(
		events,
		expectedEvents,
	) {
		t.Fatalf(
			"expected events %v, got %v",
			expectedEvents,
			events,
		)
	}
}
