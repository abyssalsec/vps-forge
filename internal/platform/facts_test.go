package platform

import "testing"

func TestParseOSRelease(t *testing.T) {
	data := `NAME="Ubuntu"
ID=ubuntu
VERSION_ID="24.04"
VERSION_CODENAME=noble
PRETTY_NAME="Ubuntu 24.04 LTS"
`

	values := parseOSRelease(data)

	if values["ID"] != "ubuntu" {
		t.Fatalf(
			"expected ubuntu, got %q",
			values["ID"],
		)
	}

	if values["VERSION_ID"] != "24.04" {
		t.Fatalf(
			"expected 24.04, got %q",
			values["VERSION_ID"],
		)
	}

	if values["PRETTY_NAME"] !=
		"Ubuntu 24.04 LTS" {

		t.Fatalf(
			"unexpected pretty name %q",
			values["PRETTY_NAME"],
		)
	}
}

func TestSupportedUbuntu(t *testing.T) {
	tests := []struct {
		version   string
		supported bool
	}{
		{"22.04", true},
		{"24.04", true},
		{"26.04", true},
		{"20.04", false},
		{"28.04", false},
	}

	for _, test := range tests {
		facts := Facts{
			ID:        "ubuntu",
			VersionID: test.version,
		}

		if facts.Supported() != test.supported {
			t.Fatalf(
				"version %s support mismatch",
				test.version,
			)
		}
	}
}
