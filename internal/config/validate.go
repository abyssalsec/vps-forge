package config

import (
	"fmt"
	"regexp"
	"strings"
)

var usernamePattern = regexp.MustCompile(
	`^[a-z_][a-z0-9_-]{0,31}$`,
)

func Validate(cfg Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf(
			"unsupported configuration version: %d",
			cfg.Version,
		)
	}

	switch cfg.Profile {
	case "minimal", "web", "php", "docker":
	default:
		return fmt.Errorf(
			"unsupported profile %q",
			cfg.Profile,
		)
	}

	seenUsers := make(map[string]struct{})

	for _, user := range cfg.Users {
		if !usernamePattern.MatchString(user.Name) {
			return fmt.Errorf(
				"invalid Linux username %q",
				user.Name,
			)
		}

		if _, exists := seenUsers[user.Name]; exists {
			return fmt.Errorf(
				"duplicate user %q",
				user.Name,
			)
		}

		seenUsers[user.Name] = struct{}{}

		for _, key := range user.SSHAuthorizedKeys {
			if strings.TrimSpace(key) == "" {
				return fmt.Errorf(
					"user %q contains an empty SSH key",
					user.Name,
				)
			}
		}
	}

	resolved := Resolve(cfg)

	if resolved.Security.Firewall.Provider != "ufw" {
		return fmt.Errorf(
			"unsupported firewall provider %q: MVP supports ufw only",
			resolved.Security.Firewall.Provider,
		)
	}

	if resolved.Web.TLS.Enabled {
		if !resolved.Web.Nginx {
			return fmt.Errorf(
				"TLS requires nginx to be enabled",
			)
		}

		if strings.TrimSpace(resolved.Web.Domain) == "" {
			return fmt.Errorf(
				"TLS requires web.domain",
			)
		}

		if !strings.Contains(
			resolved.Web.TLS.Email,
			"@",
		) {
			return fmt.Errorf(
				"TLS requires a valid contact email",
			)
		}
	}

	if resolved.Runtime.PHP.Enabled &&
		!resolved.Web.Nginx {

		return fmt.Errorf(
			"PHP profile requires nginx",
		)
	}

	switch resolved.Database.Engine {
	case "", "mariadb", "postgresql":
	default:
		return fmt.Errorf(
			"unsupported database engine %q",
			resolved.Database.Engine,
		)
	}

	if resolved.Backup.Enabled {
		if len(resolved.Backup.Paths) == 0 {
			return fmt.Errorf(
				"backup.enabled requires at least one backup path",
			)
		}

		if resolved.Backup.RetentionDays < 1 {
			return fmt.Errorf(
				"backup retention_days must be greater than zero",
			)
		}

		switch resolved.Backup.Schedule {
		case "daily", "weekly":
		default:
			return fmt.Errorf(
				"unsupported backup schedule %q",
				resolved.Backup.Schedule,
			)
		}
	}

	return nil
}
