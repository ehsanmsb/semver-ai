package application

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/analyzer"
	"github.com/ehsanmsb/semver-ai/internal/config"
	gitrepo "github.com/ehsanmsb/semver-ai/internal/git"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type provider struct {
	class string
	fail  bool
	seen  string
}

func (p *provider) Classify(_ context.Context, r ai.Request) (ai.Result, error) {
	p.seen = r.Diff + "\n" + r.Metadata
	if p.fail {
		return ai.Result{}, fmt.Errorf("AI unavailable")
	}
	var f []ai.Finding
	for _, file := range r.Files {
		f = append(f, ai.Finding{Classification: p.class, Explanation: "Change " + file, Evidence: file, Files: []string{file}, Confidence: .9, Compatibility: "API changed"})
	}
	return ai.Result{Changes: f}, nil
}

type remote struct {
	scm.Store
	head string
	tags []scm.Tag
}

func (r remote) Project(context.Context) (scm.Project, error) {
	return scm.Project{ID: 42, Path: "g/p"}, nil
}
func (r remote) Branch(context.Context, string) (string, error) { return r.head, nil }
func (r remote) Tags(context.Context) ([]scm.Tag, error)        { return r.tags, nil }
func TestPreparation(t *testing.T) {
	d := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = d
		if out, e := c.CombinedOutput(); e != nil {
			t.Fatal(e, string(out))
		}
	}
	git("init", "-b", "main")
	git("config", "user.name", "test")
	git("config", "user.email", "test@example.com")
	write := func(name, text string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(d, name), []byte(text), 0600); e != nil {
			t.Fatal(e)
		}
		git("add", ".")
		git("commit", "-m", text)
	}
	write("a.go", "first MR")
	write("b.go", "second MR")
	cfg := config.Defaults()
	cfg.AI.Model = "test"
	repo := gitrepo.Repository{Dir: d}
	head, e := repo.Head(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	p := &provider{class: "FEATURE"}
	svc := Service{cfg, repo, analyzer.Analyzer{Provider: p, Config: cfg.Analysis}, remote{head: head}}
	report, c, e := svc.Prepare(context.Background(), "main")
	if e != nil || c == nil || c.Next != "0.1.0" || report.Status != "RELEASE_AVAILABLE" {
		t.Fatal(report, c, e)
	}
	if !strings.Contains(p.seen, "first MR") || !strings.Contains(p.seen, "second MR") {
		t.Fatal("incomplete change set")
	}
	p.class = "DOCS"
	report, c, e = svc.Prepare(context.Background(), "main")
	if e != nil || c != nil || report.Status != "NO_RELEASE" {
		t.Fatal(report, c, e)
	}
	p.fail = true
	_, c, e = svc.Prepare(context.Background(), "main")
	if e == nil || c != nil {
		t.Fatal("AI failure did not block preparation")
	}
	if _, _, e = svc.Prepare(context.Background(), "feature"); e == nil {
		t.Fatal("wrong branch accepted")
	}
}
