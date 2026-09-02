package engine

import (
	"fmt"
	"sort"

	"github.com/abyssalsec/vps-forge/internal/module"
)

type Registry struct {
	modules map[string]module.Module
}

func NewRegistry(
	modules ...module.Module,
) (*Registry, error) {

	registry := &Registry{
		modules: make(
			map[string]module.Module,
		),
	}

	for _, current := range modules {
		if current == nil {
			return nil, fmt.Errorf(
				"cannot register nil module",
			)
		}

		name := current.Name()

		if name == "" {
			return nil, fmt.Errorf(
				"module name cannot be empty",
			)
		}

		if _, exists := registry.modules[name]; exists {
			return nil, fmt.Errorf(
				"module %q is already registered",
				name,
			)
		}

		registry.modules[name] = current
	}

	return registry, nil
}

func (r *Registry) Get(
	name string,
) (module.Module, bool) {

	current, ok := r.modules[name]
	return current, ok
}

func (r *Registry) Names() []string {
	names := make(
		[]string,
		0,
		len(r.modules),
	)

	for name := range r.modules {
		names = append(
			names,
			name,
		)
	}

	sort.Strings(names)

	return names
}

func (r *Registry) Resolve(
	requested []string,
) ([]module.Module, error) {

	requestedCopy := append(
		[]string(nil),
		requested...,
	)

	sort.Strings(requestedCopy)

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)

	state := make(map[string]int)

	var ordered []module.Module

	var visit func(string, []string) error

	visit = func(
		name string,
		stack []string,
	) error {

		switch state[name] {
		case visited:
			return nil

		case visiting:
			cycle := append(
				append([]string(nil), stack...),
				name,
			)

			return fmt.Errorf(
				"module dependency cycle: %v",
				cycle,
			)
		}

		current, exists := r.modules[name]

		if !exists {
			return fmt.Errorf(
				"module %q is not registered",
				name,
			)
		}

		state[name] = visiting

		dependencies := append(
			[]string(nil),
			current.Dependencies()...,
		)

		sort.Strings(dependencies)

		for _, dependency := range dependencies {
			if _, exists := r.modules[dependency]; !exists {
				return fmt.Errorf(
					"module %q requires unregistered dependency %q",
					name,
					dependency,
				)
			}

			if err := visit(
				dependency,
				append(stack, name),
			); err != nil {
				return err
			}
		}

		state[name] = visited

		ordered = append(
			ordered,
			current,
		)

		return nil
	}

	for _, name := range requestedCopy {
		if err := visit(
			name,
			nil,
		); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}
