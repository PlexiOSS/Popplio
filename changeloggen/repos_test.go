package changeloggen

import "testing"

func TestValidPackageName(t *testing.T) {
	for name, want := range map[string]bool{
		"sdk-js":    true,
		"api-types": true,
		"sdk-rust":  true,
		"a":         true,
		"":          false,
		"-sdk":      false,
		"SDK":       false,
		"sdk_js":    false,
		"../etc":    false,
		"sdk js":    false,
		"x123456789012345678901234567890123456789012345678901234567890123":  true,
		"x1234567890123456789012345678901234567890123456789012345678901234": false,
	} {
		if got := ValidPackageName(name); got != want {
			t.Errorf("ValidPackageName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestPackageLabel(t *testing.T) {
	for name, want := range map[string]string{
		"sdk-js":     "JavaScript SDK",
		"sdk-python": "Python SDK",
		"sdk-go":     "Go SDK",
		"sdk-rust":   "Rust SDK",
		"api-types":  "API Types",
		"sdk-kotlin": "Kotlin SDK",
		"widgets":    "widgets",
	} {
		if got := PackageLabel(name); got != want {
			t.Errorf("PackageLabel(%q) = %q, want %q", name, got, want)
		}
	}
}
