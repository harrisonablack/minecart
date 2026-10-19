// Package java detects the installed Java runtime version and vendor.
package java

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Info describes the detected Java runtime.
type Info struct {
	// Version is the runtime version, e.g. "21.0.11".
	Version string
	// Vendor is the VM vendor, e.g. "Eclipse Adoptium", "Amazon.com Inc.",
	// "Azul Systems, Inc.", or "Oracle Corporation".
	Vendor string
}

// Detect returns the version and vendor of the java binary on PATH.
//
// Java writes its settings to stderr, so the combined output of
// `java -XshowSettings:properties -version` is captured and parsed. If java
// is not installed or the command exits non-zero, Detect returns an error.
func Detect(ctx context.Context) (Info, error) {
	out, err := exec.CommandContext(ctx, "java", "-XshowSettings:properties", "-version").CombinedOutput()
	if err != nil {
		return Info{}, fmt.Errorf("run java -XshowSettings:properties -version: %w", err)
	}
	return parse(string(out))
}

// versionRe matches the java.version property in
// `java -XshowSettings:properties` output, e.g. `java.version = 21.0.11`.
// It is anchored to a whole property line so echoed options such as
// `-Dapp.java.version=...` are not mistaken for the property.
var versionRe = regexp.MustCompile(`(?m)^\s*java\.version\s*=\s*(.+)$`)

// vendorRe matches the java.vm.vendor property, e.g.
// `java.vm.vendor = Eclipse Adoptium`. It is anchored to a whole property
// line for the same reason as versionRe.
var vendorRe = regexp.MustCompile(`(?m)^\s*java\.vm\.vendor\s*=\s*(.+)$`)

// parse extracts the version and vendor from java -XshowSettings:properties
// output.
func parse(output string) (Info, error) {
	m := versionRe.FindStringSubmatch(output)
	if m == nil {
		return Info{}, fmt.Errorf("parse java version from output: %q", strings.TrimSpace(output))
	}
	info := Info{Version: strings.TrimSpace(m[1])}

	if v := vendorRe.FindStringSubmatch(output); v != nil {
		info.Vendor = strings.TrimSpace(v[1])
	}
	return info, nil
}
