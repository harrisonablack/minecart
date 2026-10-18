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
	// Version is the runtime version, e.g. "21.0.2".
	Version string
	// Vendor is the distribution vendor, e.g. "Temurin", "Corretto", "Zulu",
	// or "Oracle". It is empty when the vendor cannot be determined.
	Vendor string
}

// Detect returns the version and vendor of the java binary on PATH.
//
// Java writes its version information to stderr, so the combined output of
// `java -version` is captured and parsed.
func Detect(ctx context.Context) (Info, error) {
	out, err := exec.CommandContext(ctx, "java", "-version").CombinedOutput()
	if err != nil {
		return Info{}, fmt.Errorf("run java -version: %w", err)
	}
	return parse(string(out))
}

// versionRe matches the version string in `java -version` output, e.g.
// `openjdk version "21.0.2"` or `java version "21.0.2"`.
var versionRe = regexp.MustCompile(`version "([^"]+)"`)

// vendorRe matches the distribution vendor in the runtime line, e.g.
// `OpenJDK Runtime Environment Temurin-21.0.2+13`.
var vendorRe = regexp.MustCompile(`Runtime Environment ([A-Za-z]+)`)

// parse extracts the version and vendor from java -version output.
func parse(output string) (Info, error) {
	m := versionRe.FindStringSubmatch(output)
	if m == nil {
		return Info{}, fmt.Errorf("parse java version from output: %q", strings.TrimSpace(output))
	}
	info := Info{Version: m[1]}

	if v := vendorRe.FindStringSubmatch(output); v != nil {
		info.Vendor = v[1]
	} else if strings.Contains(output, "Java(TM) SE Runtime Environment") {
		info.Vendor = "Oracle"
	}
	return info, nil
}
