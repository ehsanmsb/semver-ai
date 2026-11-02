package config

import (
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	valid := "version: 1\nai:\n  model: test-model\n"
	c, e := Decode(strings.NewReader(valid))
	if e != nil || c.Release.InitialVersion != "0.1.0" || !c.Release.Approval.Required {
		t.Fatalf("defaults: %+v %v", c, e)
	}
	for _, s := range []string{"", valid + "unknown: true\n", valid + "release: null\n", valid + "version: 1\n", valid + "---\nversion: 1\n", valid + "release:\n  approval:\n    required: false\n", valid + "analysis:\n  confidence_threshold: .nan\n", valid + "gitlab:\n  api_url: http://gitlab.example.com/api/v4\n", "ai:\n  model: test\n  timeout: 0s\n", valid + "release:\n  initial_version: 1.2\n", valid + "release:\n  rules:\n    docs: banana\n"} {
		if _, e := Decode(strings.NewReader(s)); e == nil {
			t.Errorf("accepted invalid config %q", s)
		}
	}
}
