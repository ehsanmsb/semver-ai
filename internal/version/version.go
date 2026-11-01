package version

import (
	"fmt"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/ehsanmsb/semver-ai/internal/config"
)

type Bump string

const (
	None  Bump = "NONE"
	Patch Bump = "PATCH"
	Minor Bump = "MINOR"
	Major Bump = "MAJOR"
)

func Rank(b Bump) int {
	switch b {
	case None:
		return 0
	case Patch:
		return 1
	case Minor:
		return 2
	case Major:
		return 3
	}
	return -1
}
func Impact(class string, c config.Release) (Bump, error) {
	rule := ""
	switch class {
	case "BREAKING":
		return Major, nil
	case "FEATURE":
		return Minor, nil
	case "FIX":
		return Patch, nil
	case "PERFORMANCE":
		rule = c.Rules.Performance
	case "DOCS":
		rule = c.Rules.Docs
	case "REFACTOR":
		rule = c.Rules.Refactor
	case "CHORE":
		rule = c.Rules.Chore
	case "TEST":
		rule = c.Rules.Tests
	default:
		return None, fmt.Errorf("unknown or uncertain classification: %s", class)
	}
	b := Bump(strings.ToUpper(rule))
	if Rank(b) < 0 {
		return None, fmt.Errorf("invalid bump policy")
	}
	return b, nil
}
func Aggregate(classes []string, c config.Release) (Bump, error) {
	b := None
	for _, s := range classes {
		x, e := Impact(s, c)
		if e != nil {
			return None, e
		}
		if Rank(x) > Rank(b) {
			b = x
		}
	}
	return b, nil
}
func Next(current string, b Bump, c config.Release) (string, error) {
	if Rank(b) < 0 {
		return "", fmt.Errorf("invalid bump")
	}
	if !config.StableVersion(c.InitialVersion) {
		return "", fmt.Errorf("invalid initial version")
	}
	if current != "" && !config.StableVersion(current) {
		return "", fmt.Errorf("current version must be stable SemVer")
	}
	if b == None {
		return "", nil
	}
	if current == "" {
		return c.InitialVersion, nil
	}
	v, _ := semver.StrictNewVersion(current)
	if b == Major && v.Major() == 0 {
		if c.Pre1.Breaking == "minor" {
			b = Minor
		} else if c.Pre1.Breaking != "major" {
			return "", fmt.Errorf("invalid pre-1.0 policy")
		}
	}
	var next semver.Version
	switch b {
	case Major:
		next = v.IncMajor()
	case Minor:
		next = v.IncMinor()
	case Patch:
		next = v.IncPatch()
	}
	if !next.GreaterThan(v) {
		return "", fmt.Errorf("version overflow")
	}
	return next.String(), nil
}
func FromTag(tag, prefix string) (string, bool) {
	if !strings.HasPrefix(tag, prefix) {
		return "", false
	}
	v := strings.TrimPrefix(tag, prefix)
	return v, config.StableVersion(v)
}
func Latest(tags []string, prefix string) string {
	latest := ""
	var best *semver.Version
	for _, tag := range tags {
		s, ok := FromTag(tag, prefix)
		if !ok {
			continue
		}
		v, _ := semver.StrictNewVersion(s)
		if best == nil || v.GreaterThan(best) {
			best = v
			latest = tag
		}
	}
	return latest
}
