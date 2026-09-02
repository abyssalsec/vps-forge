package platform

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

type Facts struct {
	ID              string
	VersionID       string
	VersionCodename string
	PrettyName      string
	Hostname        string
	Kernel          string
	Architecture    string
	Timezone        string
	Systemd         bool
	EffectiveUID    int
}

func parseOSRelease(data string) map[string]string {
	values := make(map[string]string)

	scanner := bufio.NewScanner(
		strings.NewReader(data),
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		value = strings.TrimSpace(value)

		if len(value) >= 2 {
			if (value[0] == '"' &&
				value[len(value)-1] == '"') ||
				(value[0] == '\'' &&
					value[len(value)-1] == '\'') {

				value = value[1 : len(value)-1]
			}
		}

		values[key] = value
	}

	return values
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func detectTimezone() string {
	if timezone := readTrimmed("/etc/timezone"); timezone != "" {
		return timezone
	}

	target, err := os.Readlink("/etc/localtime")
	if err != nil {
		return ""
	}

	const prefix = "/usr/share/zoneinfo/"

	if strings.HasPrefix(target, prefix) {
		return strings.TrimPrefix(
			target,
			prefix,
		)
	}

	return ""
}

func Detect() (Facts, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return Facts{}, fmt.Errorf(
			"read /etc/os-release: %w",
			err,
		)
	}

	osRelease := parseOSRelease(
		string(data),
	)

	hostname, err := os.Hostname()
	if err != nil {
		return Facts{}, fmt.Errorf(
			"detect hostname: %w",
			err,
		)
	}

	_, systemdErr := os.Stat(
		"/run/systemd/system",
	)

	return Facts{
		ID: osRelease["ID"],

		VersionID: osRelease["VERSION_ID"],

		VersionCodename: osRelease["VERSION_CODENAME"],

		PrettyName: osRelease["PRETTY_NAME"],

		Hostname: hostname,

		Kernel: readTrimmed(
			"/proc/sys/kernel/osrelease",
		),

		Architecture: runtime.GOARCH,

		Timezone: detectTimezone(),

		Systemd: systemdErr == nil,

		EffectiveUID: os.Geteuid(),
	}, nil
}

func (f Facts) IsRoot() bool {
	return f.EffectiveUID == 0
}

func (f Facts) Supported() bool {
	if f.ID != "ubuntu" {
		return false
	}

	switch f.VersionID {
	case "22.04", "24.04", "26.04":
		return true
	default:
		return false
	}
}
