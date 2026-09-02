package config

type Config struct {
	Version  int            `yaml:"version"`
	Profile  string         `yaml:"profile"`
	Server   ServerConfig   `yaml:"server"`
	Users    []UserConfig   `yaml:"users"`
	Security SecurityConfig `yaml:"security"`
	Web      WebConfig      `yaml:"web"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
	Database DatabaseConfig `yaml:"database"`
	Docker   DockerConfig   `yaml:"docker"`
	Backup   BackupConfig   `yaml:"backup"`
}

type ServerConfig struct {
	Hostname string `yaml:"hostname"`
	Timezone string `yaml:"timezone"`
}

type UserConfig struct {
	Name              string   `yaml:"name"`
	Sudo              bool     `yaml:"sudo"`
	SSHAuthorizedKeys []string `yaml:"ssh_authorized_keys"`
}

type SecurityConfig struct {
	SSH                SSHConfig      `yaml:"ssh"`
	Firewall           FirewallConfig `yaml:"firewall"`
	Fail2Ban           ToggleConfig   `yaml:"fail2ban"`
	UnattendedUpgrades ToggleConfig   `yaml:"unattended_upgrades"`
}

type SSHConfig struct {
	Enabled                *bool `yaml:"enabled"`
	PasswordAuthentication *bool `yaml:"password_authentication"`
	RootLogin              *bool `yaml:"root_login"`
}

type FirewallConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Provider string `yaml:"provider"`
}

type ToggleConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type WebConfig struct {
	Nginx  ToggleConfig `yaml:"nginx"`
	Domain string       `yaml:"domain"`
	TLS    TLSConfig    `yaml:"tls"`
}

type TLSConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Email   string `yaml:"email"`
}

type RuntimeConfig struct {
	PHP PHPConfig `yaml:"php"`
}

type PHPConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Version string `yaml:"version"`
}

type DatabaseConfig struct {
	Engine string `yaml:"engine"`
}

type DockerConfig struct {
	Enabled *bool `yaml:"enabled"`
}

type BackupConfig struct {
	Enabled       *bool    `yaml:"enabled"`
	Schedule      string   `yaml:"schedule"`
	RetentionDays int      `yaml:"retention_days"`
	Paths         []string `yaml:"paths"`
}

type ResolvedConfig struct {
	Version  int
	Profile  string
	Server   ServerConfig
	Users    []UserConfig
	Security ResolvedSecurityConfig
	Web      ResolvedWebConfig
	Runtime  ResolvedRuntimeConfig
	Database DatabaseConfig
	Docker   ResolvedDockerConfig
	Backup   ResolvedBackupConfig
}

type ResolvedSecurityConfig struct {
	SSH                ResolvedSSHConfig
	Firewall           ResolvedFirewallConfig
	Fail2Ban           bool
	UnattendedUpgrades bool
}

type ResolvedSSHConfig struct {
	Enabled                bool
	PasswordAuthentication bool
	RootLogin              bool
}

type ResolvedFirewallConfig struct {
	Enabled  bool
	Provider string
}

type ResolvedWebConfig struct {
	Nginx  bool
	Domain string
	TLS    ResolvedTLSConfig
}

type ResolvedTLSConfig struct {
	Enabled bool
	Email   string
}

type ResolvedRuntimeConfig struct {
	PHP ResolvedPHPConfig
}

type ResolvedPHPConfig struct {
	Enabled bool
	Version string
}

type ResolvedDockerConfig struct {
	Enabled bool
}

type ResolvedBackupConfig struct {
	Enabled       bool
	Schedule      string
	RetentionDays int
	Paths         []string
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}

func Resolve(cfg Config) ResolvedConfig {
	minimal := false
	web := false
	php := false
	docker := false

	switch cfg.Profile {
	case "minimal":
		minimal = true

	case "web":
		minimal = true
		web = true

	case "php":
		minimal = true
		web = true
		php = true

	case "docker":
		minimal = true
		docker = true
	}

	firewallProvider := cfg.Security.Firewall.Provider
	if firewallProvider == "" {
		firewallProvider = "ufw"
	}

	phpVersion := cfg.Runtime.PHP.Version
	if phpVersion == "" {
		phpVersion = "system"
	}

	databaseEngine := cfg.Database.Engine
	if php && databaseEngine == "" {
		databaseEngine = "mariadb"
	}

	backupSchedule := cfg.Backup.Schedule
	if backupSchedule == "" {
		backupSchedule = "daily"
	}

	retentionDays := cfg.Backup.RetentionDays
	if retentionDays == 0 {
		retentionDays = 7
	}

	return ResolvedConfig{
		Version: cfg.Version,
		Profile: cfg.Profile,
		Server:  cfg.Server,
		Users:   cfg.Users,

		Security: ResolvedSecurityConfig{
			SSH: ResolvedSSHConfig{
				Enabled: boolValue(
					cfg.Security.SSH.Enabled,
					minimal,
				),
				PasswordAuthentication: boolValue(
					cfg.Security.SSH.PasswordAuthentication,
					false,
				),
				RootLogin: boolValue(
					cfg.Security.SSH.RootLogin,
					false,
				),
			},

			Firewall: ResolvedFirewallConfig{
				Enabled: boolValue(
					cfg.Security.Firewall.Enabled,
					minimal,
				),
				Provider: firewallProvider,
			},

			Fail2Ban: boolValue(
				cfg.Security.Fail2Ban.Enabled,
				minimal,
			),

			UnattendedUpgrades: boolValue(
				cfg.Security.UnattendedUpgrades.Enabled,
				minimal,
			),
		},

		Web: ResolvedWebConfig{
			Nginx: boolValue(
				cfg.Web.Nginx.Enabled,
				web,
			),

			Domain: cfg.Web.Domain,

			TLS: ResolvedTLSConfig{
				Enabled: boolValue(
					cfg.Web.TLS.Enabled,
					web,
				),
				Email: cfg.Web.TLS.Email,
			},
		},

		Runtime: ResolvedRuntimeConfig{
			PHP: ResolvedPHPConfig{
				Enabled: boolValue(
					cfg.Runtime.PHP.Enabled,
					php,
				),
				Version: phpVersion,
			},
		},

		Database: DatabaseConfig{
			Engine: databaseEngine,
		},

		Docker: ResolvedDockerConfig{
			Enabled: boolValue(
				cfg.Docker.Enabled,
				docker,
			),
		},

		Backup: ResolvedBackupConfig{
			Enabled: boolValue(
				cfg.Backup.Enabled,
				false,
			),
			Schedule:      backupSchedule,
			RetentionDays: retentionDays,
			Paths:         cfg.Backup.Paths,
		},
	}
}
