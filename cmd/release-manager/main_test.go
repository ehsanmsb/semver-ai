package main

import (
	"bytes"
	"testing"
)

func TestCLI(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want int
	}{{nil, 2}, {[]string{"missing"}, 2}, {[]string{"version", "--build", "--json"}, 0}, {[]string{"validate-config", "--config", "../../examples/.release-manager.yml"}, 0}, {[]string{"predict", "--config", "missing"}, 1}, {[]string{"prepare-release", "--help"}, 0}} {
		var out, err bytes.Buffer
		if got := run(tc.args, &out, &err); got != tc.want {
			t.Errorf("%v: got %d want %d: %s", tc.args, got, tc.want, err.String())
		}
	}
}
