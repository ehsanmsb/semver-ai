package analyzer

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"strings"
	"testing"
)

type fake func(ai.Request) (ai.Result, error)

func (f fake) Classify(_ context.Context, r ai.Request) (ai.Result, error) { return f(r) }
func TestAnalysis(t *testing.T) {
	in := ai.Request{Diff: "diff --git a/a.go b/a.go\n+fixed\n+password=secret", Files: []string{"a.go"}}
	f := ai.Finding{Classification: "FIX", Explanation: "repair", Evidence: "+fixed", Files: in.Files, Confidence: .9}
	for _, mode := range []string{"valid", "low", "unknown", "evidence", "files", "unavailable", "uncertain", "empty"} {
		t.Run(mode, func(t *testing.T) {
			a := Analyzer{Config: config.Defaults().Analysis, Provider: fake(func(r ai.Request) (ai.Result, error) {
				if strings.Contains(r.Diff, "password=secret") {
					t.Fatal("secret not redacted")
				}
				x := f
				switch mode {
				case "low":
					x.Confidence = .5
				case "unknown":
					x.Classification = "UNKNOWN"
				case "evidence":
					x.Evidence = "invented"
				case "files":
					x.Files = []string{"other"}
				case "unavailable":
					return ai.Result{}, fmt.Errorf("down")
				case "uncertain":
					x.Uncertainty = "missing context"
				case "empty":
					return ai.Result{}, nil
				}
				return ai.Result{Changes: []ai.Finding{x}}, nil
			})}
			_, e := a.Analyze(context.Background(), in)
			if (e != nil) != (mode != "valid") {
				t.Fatal(e)
			}
		})
	}
}
func TestRedaction(t *testing.T) {
	s := Redact("token: value\n-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\nsk-longsecret123456\nexact-value", []string{"exact-value"})
	for _, secret := range []string{"value", "abc", "longsecret", "exact"} {
		if strings.Contains(s, secret) {
			t.Fatalf("leak: %s", s)
		}
	}
}
func TestChunkingPreservesBreaking(t *testing.T) {
	c := config.Defaults().Analysis
	c.Limits.ContextTokens = 6000
	calls := 0
	a := Analyzer{Config: c, Provider: fake(func(r ai.Request) (ai.Result, error) {
		calls++
		class := "FIX"
		file := "a"
		if strings.Contains(r.Diff, "removed") {
			class = "BREAKING"
			file = "b"
		}
		return ai.Result{Changes: []ai.Finding{{Classification: class, Explanation: "change", Evidence: file, Files: []string{file}, Confidence: .9, Compatibility: "removed API"}}}, nil
	})}
	in := ai.Request{Diff: "diff --git a/a b/a\n" + strings.Repeat("a", 1100) + "\ndiff --git a/b b/b\nremoved " + strings.Repeat("b", 1100), Files: []string{"a", "b"}}
	r, e := a.Analyze(context.Background(), in)
	if e != nil || calls != 2 || r.Changes[1].Classification != "BREAKING" {
		t.Fatal(r, e, calls)
	}
}

func TestCredentialFileBlocksUpload(t *testing.T) {
	a := Analyzer{Config: config.Defaults().Analysis, Provider: fake(func(ai.Request) (ai.Result, error) { t.Fatal("credential file uploaded"); return ai.Result{}, nil })}
	_, e := a.Analyze(context.Background(), ai.Request{Files: []string{".env.production"}, Diff: "secret=custom-value"})
	if e == nil {
		t.Fatal("credential file accepted")
	}
}
