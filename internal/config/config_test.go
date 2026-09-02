package config

import (
	"os"
	"path/filepath"
	"testing"
)

func boolPointer(value bool) *bool {
	return &value
}

func TestResolveMinimalProfile(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profile: "minimal",
	}

	resolved := Resolve(cfg)

	if !resolved.Security.SSH.Enabled {
		t.Fatal(
			"minimal profile must enable SSH management",
		)
	}

	if !resolved.Security.Firewall.Enabled {
		t.Fatal(
			"minimal profile must enable firewall",
		)
	}

	if resolved.Security.Firewall.Provider != "ufw" {
		t.Fatalf(
			"expected ufw, got %q",
			resolved.Security.Firewall.Provider,
		)
	}

	if !resolved.Security.Fail2Ban {
		t.Fatal(
			"minimal profile must enable Fail2Ban",
		)
	}

	if !resolved.Security.UnattendedUpgrades {
		t.Fatal(
			"minimal profile must enable unattended upgrades",
		)
	}
}

func TestExplicitBooleanOverridesProfile(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profile: "minimal",

		Security: SecurityConfig{
			Firewall: FirewallConfig{
				Enabled: boolPointer(false),
			},
		},
	}

	resolved := Resolve(cfg)

	if resolved.Security.Firewall.Enabled {
		t.Fatal(
			"explicit false must override profile default",
		)
	}
}

func TestUnknownYAMLFieldRejected(t *testing.T) {
	temp := t.TempDir()

	path := filepath.Join(
		temp,
		"forge.yaml",
	)

	data := []byte(`version: 1
profile: minimal
unknown_option: true
`)

	if err := os.WriteFile(
		path,
		data,
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal(
			"expected unknown YAML field to fail",
		)
	}
}

func TestTLSRequiresDomain(t *testing.T) {
	enabled := true

	cfg := Config{
		Version: 1,
		Profile: "minimal",

		Web: WebConfig{
			Nginx: ToggleConfig{
				Enabled: &enabled,
			},

			TLS: TLSConfig{
				Enabled: &enabled,
				Email:   "admin@example.com",
			},
		},
	}

	if err := Validate(cfg); err == nil {
		t.Fatal(
			"expected TLS without domain to fail",
		)
	}
}
