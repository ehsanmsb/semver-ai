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

func TestCompletionBudget(t *testing.T) {
	base := "version: 1\nai:\n  model: test\n"
	for _, tc := range []struct {
		suffix string
		want   int
		bad    bool
	}{
		{"", 4096, false}, {"  max_completion_tokens: 16384\n", 16384, false},
		{"  max_completion_tokens: 0\n", 0, true}, {"  max_completion_tokens: -1\n", 0, true},
		{"  max_completion_tokens: 131073\n", 0, true},
	} {
		c, e := Decode(strings.NewReader(base + tc.suffix))
		if (e != nil) != tc.bad {
			t.Fatalf("%s: %v", tc.suffix, e)
		}
		if e == nil && c.AI.MaxCompletionTokens != tc.want {
			t.Fatal(c.AI.MaxCompletionTokens)
		}
	}
}
