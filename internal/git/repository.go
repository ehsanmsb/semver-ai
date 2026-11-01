package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/ehsanmsb/semver-ai/internal/version"
)

type Repository struct{ Dir string }
type Snapshot struct {
	BaseTag    string   `json:"base_tag"`
	BaseCommit string   `json:"base_commit"`
	Target     string   `json:"target_commit"`
	Range      string   `json:"commit_range"`
	Diff       string   `json:"diff"`
	Commits    string   `json:"commits"`
	Files      []string `json:"files"`
	Context    string   `json:"repository_context"`
}
type limitedBuffer struct {
	bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, fmt.Errorf("git output exceeds safe limit")
	}
	return b.Buffer.Write(p)
}
func (r Repository) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.quotePath=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir = r.Dir
	var out limitedBuffer
	out.max = 20 * 1024 * 1024
	cmd.Stdout = &out
	cmd.Stderr = &limitedBuffer{max: 4096}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed (check repository/history): %w", args[0], err)
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}
func (r Repository) Head(ctx context.Context) (string, error) {
	return r.run(ctx, "rev-parse", "--verify", "HEAD^{commit}")
}
func (r Repository) Branch(ctx context.Context) (string, error) {
	return r.run(ctx, "symbolic-ref", "--short", "HEAD")
}
func (r Repository) Resolve(ctx context.Context, ref string) (string, error) {
	return r.run(ctx, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
}
func (r Repository) Complete(ctx context.Context) error {
	s, e := r.run(ctx, "rev-parse", "--is-shallow-repository")
	if e != nil {
		return e
	}
	if s != "false" {
		return fmt.Errorf("shallow repository: fetch full history and tags (GIT_DEPTH: 0)")
	}
	return nil
}
func (r Repository) Latest(ctx context.Context, prefix, target string) (string, error) {
	if e := r.Complete(ctx); e != nil {
		return "", e
	}
	s, e := r.run(ctx, "tag", "--merged", target)
	if e != nil {
		return "", e
	}
	return version.Latest(strings.Split(s, "\n"), prefix), nil
}
func (r Repository) Snapshot(ctx context.Context, prefix string) (Snapshot, error) {
	var s Snapshot
	var e error
	if s.Target, e = r.Head(ctx); e != nil {
		return s, e
	}
	if s.BaseTag, e = r.Latest(ctx, prefix, s.Target); e != nil {
		return s, e
	}
	if s.BaseTag != "" {
		if s.BaseCommit, e = r.Resolve(ctx, s.BaseTag); e != nil {
			return s, e
		}
	}
	return r.Between(ctx, s.BaseTag, s.BaseCommit, s.Target)
}
func (r Repository) Between(ctx context.Context, tag, base, target string) (Snapshot, error) {
	s := Snapshot{BaseTag: tag, BaseCommit: base, Target: target}
	if e := r.Complete(ctx); e != nil {
		return s, e
	}
	var e error
	if s.Target, e = r.Resolve(ctx, target); e != nil {
		return s, e
	}
	diffBase := base
	if base == "" {
		diffBase, e = r.run(ctx, "hash-object", "-t", "tree", "/dev/null")
		s.Range = s.Target
	} else {
		if s.BaseCommit, e = r.Resolve(ctx, base); e != nil {
			return s, e
		}
		diffBase = s.BaseCommit
		if _, e = r.run(ctx, "merge-base", "--is-ancestor", s.BaseCommit, s.Target); e != nil {
			return s, fmt.Errorf("base is not an ancestor of target")
		}
		s.Range = s.BaseCommit + ".." + s.Target
	}
	if e != nil {
		return s, e
	}
	if s.Diff, e = r.run(ctx, "diff", "--no-ext-diff", "--no-textconv", "--no-renames", "--unified=3", diffBase, s.Target, "--"); e != nil {
		return s, e
	}
	files, e := r.run(ctx, "diff", "--name-only", "-z", "--no-renames", diffBase, s.Target, "--")
	if e != nil {
		return s, e
	}
	for _, f := range strings.Split(files, "\x00") {
		if f != "" {
			s.Files = append(s.Files, f)
		}
	}
	if s.Commits, e = r.run(ctx, "log", "--format=%H%n%B", s.Range, "--"); e != nil {
		return s, e
	}
	return s, nil
}

// Context includes complete changed API/schema files, never arbitrary working-tree files.
func (r Repository) Context(ctx context.Context, s Snapshot) (string, error) {
	var b strings.Builder
	for _, f := range s.Files {
		lower := strings.ToLower(f)
		if !(strings.HasSuffix(lower, ".proto") || strings.HasSuffix(lower, ".sql") || strings.Contains(path.Base(lower), "openapi") || strings.Contains(path.Base(lower), "swagger")) {
			continue
		}
		exists, e := r.run(ctx, "ls-tree", s.Target, "--", f)
		if e != nil {
			return "", e
		}
		if exists == "" {
			continue
		}
		content, e := r.run(ctx, "show", s.Target+":"+f)
		if e != nil {
			return "", e
		}
		if b.Len()+len(content) > 1000000 {
			return "", fmt.Errorf("repository API context exceeds 1 MB; narrow the release range")
		}
		fmt.Fprintf(&b, "\nFILE %s\n%s\n", f, content)
	}
	return b.String(), nil
}
