package application

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/analyzer"
	"github.com/ehsanmsb/semver-ai/internal/config"
	gitrepo "github.com/ehsanmsb/semver-ai/internal/git"
	"github.com/ehsanmsb/semver-ai/internal/release"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	"github.com/ehsanmsb/semver-ai/internal/version"
	"time"
)

type Service struct {
	Config   config.Config
	Repo     gitrepo.Repository
	Analyzer analyzer.Analyzer
	Store    scm.Store
}
type Report struct {
	Status   string       `json:"status"`
	Current  string       `json:"current_version"`
	Next     string       `json:"next_version"`
	Bump     version.Bump `json:"bump"`
	Target   string       `json:"target_commit"`
	Analysis ai.Result    `json:"analysis"`
	Notes    string       `json:"release_notes"`
}

func (s Service) Analyze(ctx context.Context, snap gitrepo.Snapshot, metadata string) (Report, error) {
	r := Report{Status: "NO_RELEASE", Bump: version.None, Target: snap.Target, Analysis: ai.Result{Changes: []ai.Finding{}}}
	if snap.BaseTag != "" {
		var ok bool
		r.Current, ok = version.FromTag(snap.BaseTag, s.Config.Release.TagPrefix)
		if !ok {
			return r, fmt.Errorf("invalid base tag")
		}
	}
	if len(snap.Files) == 0 {
		return r, nil
	}
	if s.Config.Analysis.Context.Commits {
		metadata += "\n" + snap.Commits
	}
	contextText := ""
	var e error
	if s.Config.Analysis.Context.Files {
		contextText, e = s.Repo.Context(ctx, snap)
		if e != nil {
			return r, e
		}
	}
	r.Analysis, e = s.Analyzer.Analyze(ctx, ai.Request{Diff: snap.Diff, Files: snap.Files, Metadata: metadata, Context: contextText})
	if e != nil {
		return r, e
	}
	classes := []string{}
	for _, f := range r.Analysis.Changes {
		classes = append(classes, f.Classification)
	}
	r.Bump, e = version.Aggregate(classes, s.Config.Release)
	if e != nil {
		return r, e
	}
	r.Next, e = version.Next(r.Current, r.Bump, s.Config.Release)
	if e != nil {
		return r, e
	}
	r.Notes = release.Notes(r.Analysis.Changes, s.Config.ReleaseNotes.Sections)
	if r.Next != "" {
		r.Status = "RELEASE_AVAILABLE"
	}
	return r, nil
}
func (s Service) Predict(ctx context.Context) (Report, error) {
	snap, e := s.Repo.Snapshot(ctx, s.Config.Release.TagPrefix)
	if e != nil {
		return Report{}, e
	}
	return s.Analyze(ctx, snap, "")
}
func (s Service) Prepare(ctx context.Context, branch string) (Report, *release.Candidate, error) {
	if branch != s.Config.Release.Branch {
		return Report{}, nil, fmt.Errorf("prepare-release requires configured release branch")
	}
	snap, e := s.Repo.Snapshot(ctx, s.Config.Release.TagPrefix)
	if e != nil {
		return Report{}, nil, e
	}
	project, e := s.Store.Project(ctx)
	if e != nil {
		return Report{}, nil, e
	}
	head, e := s.Store.Branch(ctx, branch)
	if e != nil {
		return Report{}, nil, e
	}
	if head != snap.Target {
		return Report{}, nil, fmt.Errorf("local target differs from remote release branch")
	}
	tags, e := s.Store.Tags(ctx)
	if e != nil {
		return Report{}, nil, e
	}
	names := []string{}
	for _, t := range tags {
		names = append(names, t.Name)
		if t.Name == snap.BaseTag && t.Commit != snap.BaseCommit {
			return Report{}, nil, fmt.Errorf("local base tag differs from GitLab")
		}
	}
	if version.Latest(names, s.Config.Release.TagPrefix) != snap.BaseTag {
		return Report{}, nil, fmt.Errorf("local release tags are out of date; fetch tags")
	}
	r, e := s.Analyze(ctx, snap, "")
	if e != nil || r.Next == "" {
		return r, nil, e
	}
	c := &release.Candidate{Schema: "1", Project: project.Path, ProjectID: project.ID, APIURL: s.Config.GitLab.APIURL, Branch: branch, BaseTag: snap.BaseTag, BaseCommit: snap.BaseCommit, Target: snap.Target, Range: snap.Range, Next: r.Next, Bump: r.Bump, Status: "PENDING_APPROVAL", CreatedAt: time.Now().UTC(), ConfigHash: release.Hash(s.Config), Notes: r.Notes, Analysis: release.Analysis{Provider: s.Config.AI.Provider, Model: s.Config.AI.Model, Confidence: 1, Changes: r.Analysis.Changes}}
	for _, f := range r.Analysis.Changes {
		if f.Confidence < c.Analysis.Confidence {
			c.Analysis.Confidence = f.Confidence
		}
	}
	c.Seal()
	if e = c.Validate(s.Config, time.Now()); e != nil {
		return r, nil, e
	}
	return r, c, nil
}
