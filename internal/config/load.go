package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf(
			"open configuration: %w",
			err,
		)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var cfg Config

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf(
			"parse configuration: %w",
			err,
		)
	}

	var extra any

	err = decoder.Decode(&extra)

	if err != io.EOF {
		if err == nil {
			return Config{}, fmt.Errorf(
				"configuration must contain exactly one YAML document",
			)
		}

		return Config{}, fmt.Errorf(
			"parse trailing configuration data: %w",
			err,
		)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func LoadResolved(path string) (ResolvedConfig, error) {
	cfg, err := Load(path)
	if err != nil {
		return ResolvedConfig{}, err
	}

	return Resolve(cfg), nil
}
