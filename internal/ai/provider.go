package ai

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"github.com/ehsanmsb/semver-ai/internal/version"
	"math"
	"strings"
)

type Finding struct {
	Classification string   `json:"classification"`
	Explanation    string   `json:"explanation"`
	Evidence       string   `json:"evidence"`
	Files          []string `json:"files"`
	Confidence     float64  `json:"confidence"`
	Compatibility  string   `json:"compatibility"`
	Uncertainty    string   `json:"uncertainty"`
}
type Result struct {
	Changes []Finding `json:"changes"`
}
type Request struct {
	Diff     string   `json:"diff"`
	Files    []string `json:"files"`
	Metadata string   `json:"metadata"`
	Context  string   `json:"context"`
}
type Provider interface {
	Classify(context.Context, Request) (Result, error)
}

func Validate(r Result, threshold float64) error {
	if len(r.Changes) == 0 {
		return fmt.Errorf("AI returned no findings for a non-empty change set")
	}
	for _, f := range r.Changes {
		if _, e := version.Impact(f.Classification, config.Defaults().Release); e != nil {
			return e
		}
		if strings.TrimSpace(f.Explanation) == "" || strings.TrimSpace(f.Evidence) == "" || len(f.Files) == 0 {
			return fmt.Errorf("AI finding lacks explanation, evidence or files")
		}
		if math.IsNaN(f.Confidence) || math.IsInf(f.Confidence, 0) || f.Confidence < threshold || f.Confidence > 1 {
			return fmt.Errorf("analysis confidence below threshold or invalid")
		}
		if strings.TrimSpace(f.Uncertainty) != "" {
			return fmt.Errorf("analysis contains unresolved uncertainty")
		}
		if f.Classification == "BREAKING" && strings.TrimSpace(f.Compatibility) == "" {
			return fmt.Errorf("breaking finding lacks compatibility detail")
		}
	}
	return nil
}
