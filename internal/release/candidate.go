package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"github.com/ehsanmsb/semver-ai/internal/safejson"
	"github.com/ehsanmsb/semver-ai/internal/version"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Analysis struct {
	Provider   string       `json:"provider"`
	Model      string       `json:"model"`
	Confidence float64      `json:"confidence"`
	Changes    []ai.Finding `json:"changes"`
}
type Candidate struct {
	Schema     string       `json:"schema_version"`
	Project    string       `json:"project"`
	ProjectID  int64        `json:"project_id"`
	APIURL     string       `json:"api_url"`
	Branch     string       `json:"branch"`
	BaseTag    string       `json:"base_tag"`
	BaseCommit string       `json:"base_commit"`
	Target     string       `json:"target_commit"`
	Range      string       `json:"commit_range"`
	Next       string       `json:"next_version"`
	Bump       version.Bump `json:"bump"`
	Status     string       `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	ConfigHash string       `json:"config_sha256"`
	Analysis   Analysis     `json:"analysis"`
	Notes      string       `json:"release_notes"`
	Digest     string       `json:"sha256"`
}

func Hash(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func (c *Candidate) Seal() { c.Digest = ""; c.Digest = Hash(c) }

var commitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

func (c Candidate) Validate(cfg config.Config, now time.Time) error {
	digest := c.Digest
	c.Digest = ""
	if len(digest) != 64 || digest != Hash(c) {
		return fmt.Errorf("candidate integrity mismatch")
	}
	if c.Schema != "1" || c.Status != "PENDING_APPROVAL" {
		return fmt.Errorf("unsupported candidate schema/status")
	}
	if c.Project == "" || c.ProjectID <= 0 || c.APIURL != cfg.GitLab.APIURL || c.Branch != cfg.Release.Branch || c.ConfigHash != Hash(cfg) {
		return fmt.Errorf("candidate project/configuration mismatch")
	}
	if !commitID.MatchString(c.Target) || (c.BaseTag != "" && !commitID.MatchString(c.BaseCommit)) || (c.BaseTag == "" && c.BaseCommit != "") {
		return fmt.Errorf("invalid candidate commit identity")
	}
	expectedRange := c.Target
	if c.BaseCommit != "" {
		expectedRange = c.BaseCommit + ".." + c.Target
	}
	if c.Range != expectedRange {
		return fmt.Errorf("candidate range mismatch")
	}
	if c.CreatedAt.IsZero() || c.CreatedAt.After(now.Add(time.Minute)) || now.Sub(c.CreatedAt) > 7*24*time.Hour {
		return fmt.Errorf("candidate expired or timestamp invalid; prepare again")
	}
	if c.Analysis.Provider != cfg.AI.Provider || c.Analysis.Model != cfg.AI.Model {
		return fmt.Errorf("candidate AI identity mismatch")
	}
	if e := ai.Validate(ai.Result{Changes: c.Analysis.Changes}, cfg.Analysis.Confidence); e != nil {
		return e
	}
	confidence := 1.0
	classes := []string{}
	for _, f := range c.Analysis.Changes {
		classes = append(classes, f.Classification)
		if f.Confidence < confidence {
			confidence = f.Confidence
		}
	}
	if confidence != c.Analysis.Confidence {
		return fmt.Errorf("candidate confidence mismatch")
	}
	b, e := version.Aggregate(classes, cfg.Release)
	if e != nil || b == version.None || b != c.Bump {
		return fmt.Errorf("candidate bump mismatch")
	}
	current := ""
	if c.BaseTag != "" {
		var ok bool
		current, ok = version.FromTag(c.BaseTag, cfg.Release.TagPrefix)
		if !ok {
			return fmt.Errorf("invalid candidate base tag")
		}
	}
	next, e := version.Next(current, b, cfg.Release)
	if e != nil || next != c.Next {
		return fmt.Errorf("candidate version mismatch")
	}
	if c.Notes != Notes(c.Analysis.Changes, cfg.ReleaseNotes.Sections) {
		return fmt.Errorf("candidate release notes mismatch")
	}
	return nil
}
func Notes(changes []ai.Finding, sections []string) string {
	names := map[string]string{"breaking_changes": "Breaking Changes", "features": "Features", "bug_fixes": "Bug Fixes", "performance": "Performance Improvements", "other": "Other Changes"}
	groups := map[string][]string{}
	for _, f := range changes {
		s := "other"
		switch f.Classification {
		case "BREAKING":
			s = "breaking_changes"
		case "FEATURE":
			s = "features"
		case "FIX":
			s = "bug_fixes"
		case "PERFORMANCE":
			s = "performance"
		}
		line := html.EscapeString(strings.Join(strings.Fields(f.Explanation), " "))
		for _, c := range []string{"\\", "[", "]", "*", "_", "`"} {
			line = strings.ReplaceAll(line, c, "\\"+c)
		}
		groups[s] = append(groups[s], "- "+line)
	}
	var b strings.Builder
	for _, s := range sections {
		if len(groups[s]) > 0 {
			fmt.Fprintf(&b, "## %s\n\n%s\n\n", names[s], strings.Join(groups[s], "\n"))
		}
	}
	return b.String()
}
func Load(path string) (Candidate, error) {
	var c Candidate
	f, e := os.Open(path)
	if e != nil {
		return c, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 4*1024*1024+1))
	if e != nil || len(b) > 4*1024*1024 {
		return c, fmt.Errorf("candidate unreadable or too large")
	}
	if e = safejson.Decode(b, &c); e != nil {
		return c, fmt.Errorf("invalid candidate JSON")
	}
	return c, nil
}
func Write(path string, data []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".semver-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.Write(data); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func (c Candidate) Save(path string) error {
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	if len(b) > 4*1024*1024 {
		return fmt.Errorf("candidate too large")
	}
	return Write(path, append(b, '\n'))
}
