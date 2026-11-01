package version

import (
	"github.com/ehsanmsb/semver-ai/internal/config"
	"testing"
)

func TestCalculations(t *testing.T) {
	c := config.Defaults().Release
	for _, tc := range []struct {
		classes       []string
		current, want string
		bump          Bump
	}{{[]string{"FIX"}, "1.4.2", "1.4.3", Patch}, {[]string{"FEATURE", "FIX"}, "1.4.2", "1.5.0", Minor}, {[]string{"BREAKING", "FEATURE"}, "1.4.2", "2.0.0", Major}, {[]string{"DOCS", "REFACTOR", "TEST", "CHORE"}, "1.4.2", "", None}, {[]string{"BREAKING"}, "0.4.2", "0.5.0", Major}, {[]string{"FIX"}, "", "0.1.0", Patch}, {[]string{"DOCS"}, "", "", None}} {
		b, e := Aggregate(tc.classes, c)
		if e != nil || b != tc.bump {
			t.Fatalf("aggregate %v %v", b, e)
		}
		got, e := Next(tc.current, b, c)
		if e != nil || got != tc.want {
			t.Errorf("next=%q err=%v want=%q", got, e, tc.want)
		}
	}
	c.Pre1.Breaking = "major"
	got, e := Next("0.9.1", Major, c)
	if got != "1.0.0" || e != nil {
		t.Fatal(got, e)
	}
	if _, e = Aggregate([]string{"UNKNOWN"}, c); e == nil {
		t.Fatal("unknown accepted")
	}
	for _, v := range []string{"v1.2.3", "1.2", "01.2.3", "1.0.0-beta", "1.0.0+build"} {
		if _, e = Next(v, Patch, c); e == nil {
			t.Fatal("invalid version accepted", v)
		}
	}
	if Latest([]string{"v1.9.0", "v1.10.0", "v2.0.0-rc.1", "other", "v01.9.0"}, "v") != "v1.10.0" {
		t.Fatal("wrong latest")
	}
}
