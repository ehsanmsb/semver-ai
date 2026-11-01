package release

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	"github.com/ehsanmsb/semver-ai/internal/version"
	"strings"
	"sync"
	"testing"
	"time"
)

var target = strings.Repeat("a", 40)
var base = strings.Repeat("b", 40)

type store struct {
	mu                      sync.Mutex
	project                 scm.Project
	head                    string
	tags                    map[string]scm.Tag
	releases                map[string]scm.Release
	failRelease, tagTimeout bool
	writes                  int
}

func (s *store) Project(context.Context) (scm.Project, error)   { return s.project, nil }
func (s *store) Branch(context.Context, string) (string, error) { return s.head, nil }
func (s *store) Tags(context.Context) ([]scm.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var tags []scm.Tag
	for _, v := range s.tags {
		tags = append(tags, v)
	}
	return tags, nil
}
func (s *store) Tag(_ context.Context, n string) (scm.Tag, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.tags[n]
	return v, ok, nil
}
func (s *store) CreateTag(_ context.Context, v scm.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tags[v.Name]; ok {
		return fmt.Errorf("conflict")
	}
	s.tags[v.Name] = v
	s.writes++
	if s.tagTimeout {
		return fmt.Errorf("timeout after write")
	}
	return nil
}
func (s *store) Release(_ context.Context, n string) (scm.Release, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.releases[n]
	return v, ok, nil
}
func (s *store) CreateRelease(_ context.Context, v scm.Release) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failRelease {
		return fmt.Errorf("release unavailable")
	}
	if _, ok := s.releases[v.Tag]; ok {
		return fmt.Errorf("conflict")
	}
	s.releases[v.Tag] = v
	s.writes++
	return nil
}
func fixture() (config.Config, Candidate, *store) {
	cfg := config.Defaults()
	cfg.AI.Model = "test-model"
	findings := []ai.Finding{{Classification: "FIX", Explanation: "Repair bug", Evidence: "fixed", Files: []string{"a.go"}, Confidence: .95}}
	c := Candidate{Schema: "1", Project: "group/project", ProjectID: 42, APIURL: cfg.GitLab.APIURL, Branch: "main", BaseTag: "v1.0.0", BaseCommit: base, Target: target, Range: base + ".." + target, Next: "1.0.1", Bump: version.Patch, Status: "PENDING_APPROVAL", CreatedAt: time.Now().UTC(), ConfigHash: Hash(cfg), Analysis: Analysis{Provider: cfg.AI.Provider, Model: cfg.AI.Model, Confidence: .95, Changes: findings}, Notes: Notes(findings, cfg.ReleaseNotes.Sections)}
	c.Seal()
	s := &store{project: scm.Project{ID: 42, Path: "group/project"}, head: target, tags: map[string]scm.Tag{"v1.0.0": {Name: "v1.0.0", Commit: base}}, releases: map[string]scm.Release{}}
	return cfg, c, s
}
func TestRecoverAndRepeat(t *testing.T) {
	cfg, c, s := fixture()
	p := Publisher{s, cfg}
	ctx := context.Background()
	s.failRelease = true
	s.tagTimeout = true
	if e := p.Publish(ctx, c, true, false); e == nil {
		t.Fatal("expected release failure")
	}
	if len(s.tags) != 2 || len(s.releases) != 0 {
		t.Fatal("partial state")
	}
	s.failRelease = false
	if e := p.Publish(ctx, c, true, false); e != nil {
		t.Fatal(e)
	}
	if e := p.Publish(ctx, c, true, false); e != nil {
		t.Fatal(e)
	}
	if s.writes != 2 {
		t.Fatal("duplicate publication", s.writes)
	}
}
func TestRejectUnsafePublication(t *testing.T) {
	for _, mode := range []string{"unauthorized", "branch moved", "new tag", "base moved", "wrong project", "conflicting tag", "tampered", "expired", "wrong version", "notes", "config"} {
		t.Run(mode, func(t *testing.T) {
			cfg, c, s := fixture()
			auth := true
			switch mode {
			case "unauthorized":
				auth = false
			case "branch moved":
				s.head = base
			case "new tag":
				s.tags["v1.0.2"] = scm.Tag{Name: "v1.0.2", Commit: target}
			case "base moved":
				s.tags[c.BaseTag] = scm.Tag{Name: c.BaseTag, Commit: target}
			case "wrong project":
				s.project.ID = 99
			case "conflicting tag":
				s.tags["v1.0.1"] = scm.Tag{Name: "v1.0.1", Commit: base}
			case "tampered":
				c.Next = "9.0.0"
			case "expired":
				c.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
				c.Seal()
			case "wrong version":
				c.Next = "9.0.0"
				c.Seal()
			case "notes":
				c.Notes = "invented"
				c.Seal()
			case "config":
				cfg.Release.TagPrefix = "release-"
			}
			if e := (Publisher{s, cfg}).Publish(context.Background(), c, auth, false); e == nil {
				t.Fatal("unsafe publish accepted")
			}
			if s.writes != 0 {
				t.Fatal("unexpected mutation")
			}
		})
	}
}
func TestDryRunAndConcurrentRetry(t *testing.T) {
	cfg, c, s := fixture()
	p := Publisher{s, cfg}
	if e := p.Publish(context.Background(), c, true, true); e != nil || s.writes != 0 {
		t.Fatal(e, s.writes)
	}
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := p.Publish(context.Background(), c, true, false); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if s.writes != 2 {
		t.Fatal(s.writes)
	}
}
func TestCandidateRoundTrip(t *testing.T) {
	cfg, c, _ := fixture()
	path := t.TempDir() + "/candidate.json"
	if e := c.Save(path); e != nil {
		t.Fatal(e)
	}
	loaded, e := Load(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = loaded.Validate(cfg, time.Now()); e != nil {
		t.Fatal(e)
	}
}
