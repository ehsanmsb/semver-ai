package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"regexp"
	"strings"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?im)(["']?(?:password|passwd|secret|token|api[_-]?key|access[_-]?key|authorization)["']?\s*[:=]\s*)[^\r\n]+`),
	regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{12,}|glpat-[A-Za-z0-9_-]+|gh[pousr]_[A-Za-z0-9_]+|AKIA[A-Z0-9]{16})\b`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/-]+=*`),
	regexp.MustCompile(`https?://[^\s/@]+:[^\s/@]+@[^\s]+`),
}

func Redact(s string, secrets []string) string {
	for _, v := range secrets {
		if len(v) > 0 {
			s = strings.ReplaceAll(s, v, "[REDACTED]")
		}
	}
	for _, p := range patterns {
		s = p.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

type Analyzer struct {
	Provider ai.Provider
	Config   config.Analysis
	Secrets  []string
}

func (a Analyzer) Analyze(ctx context.Context, in ai.Request) (ai.Result, error) {
	var all ai.Result
	if len(in.Diff) > a.Config.Limits.DiffBytes {
		return all, fmt.Errorf("diff exceeds max_diff_bytes; nothing was silently truncated")
	}
	if len(in.Files) == 0 {
		return all, nil
	}
	in.Diff = Redact(in.Diff, a.Secrets)
	in.Metadata = Redact(in.Metadata, a.Secrets)
	in.Context = Redact(in.Context, a.Secrets)
	budget := a.Config.Limits.ContextTokens - 4096
	meta, _ := json.Marshal(ai.Request{Files: in.Files, Metadata: in.Metadata, Context: in.Context})
	budget -= len(meta)
	if budget < 256 {
		return all, fmt.Errorf("metadata/context exceeds analysis budget; increase max_context_tokens")
	}
	// Chunk only at file boundaries; oversized individual files fail closed.
	parts := strings.Split(in.Diff, "\ndiff --git ")
	var chunks []string
	current := ""
	for i, part := range parts {
		if i > 0 {
			part = "diff --git " + part
		}
		encoded, _ := json.Marshal(part)
		if len(encoded) > budget {
			return all, fmt.Errorf("single file diff exceeds chunk budget; increase max_context_tokens")
		}
		enc, _ := json.Marshal(current + "\n" + part)
		if len(enc) > budget {
			chunks = append(chunks, current)
			current = part
		} else {
			current += "\n" + part
		}
	}
	if strings.TrimSpace(current) != "" {
		chunks = append(chunks, current)
	}
	if len(chunks) == 0 {
		return all, fmt.Errorf("changed files without analyzable diff")
	}
	allowed := map[string]bool{}
	covered := map[string]bool{}
	for _, f := range in.Files {
		allowed[f] = true
	}
	for _, chunk := range chunks {
		req := in
		req.Diff = chunk
		result, e := a.Provider.Classify(ctx, req)
		if e != nil {
			return all, e
		}
		if e = ai.Validate(result, a.Config.Confidence); e != nil {
			return all, e
		}
		source := chunk + "\n" + in.Metadata + "\n" + in.Context
		for _, f := range result.Changes {
			if !strings.Contains(source, f.Evidence) {
				return all, fmt.Errorf("AI evidence is not present in supplied context")
			}
			for _, file := range f.Files {
				if !allowed[file] {
					return all, fmt.Errorf("AI referenced an unchanged file")
				}
				covered[file] = true
			}
			f.Explanation = Redact(f.Explanation, a.Secrets)
			f.Evidence = Redact(f.Evidence, a.Secrets)
			f.Compatibility = Redact(f.Compatibility, a.Secrets)
			all.Changes = append(all.Changes, f)
		}
	}
	for _, file := range in.Files {
		if !covered[file] {
			return all, fmt.Errorf("AI omitted a changed file")
		}
	}
	return all, nil
}
