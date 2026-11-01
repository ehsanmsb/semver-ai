package release

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	"github.com/ehsanmsb/semver-ai/internal/version"
	"time"
)

type Publisher struct {
	Store  scm.Store
	Config config.Config
}

func (p Publisher) Publish(ctx context.Context, c Candidate, authorized, dryRun bool) error {
	if !authorized {
		return fmt.Errorf("publication requires an authorized manual protected GitLab job")
	}
	if e := c.Validate(p.Config, time.Now()); e != nil {
		return e
	}
	project, e := p.Store.Project(ctx)
	if e != nil {
		return e
	}
	if project.ID != c.ProjectID || project.Path != c.Project {
		return fmt.Errorf("candidate belongs to another project")
	}
	head, e := p.Store.Branch(ctx, c.Branch)
	if e != nil {
		return e
	}
	if head != c.Target {
		return fmt.Errorf("candidate is stale: release branch moved; prepare again")
	}
	name := p.Config.Release.TagPrefix + c.Next
	tags, e := p.Store.Tags(ctx)
	if e != nil {
		return e
	}
	var prior []string
	for _, t := range tags {
		if t.Name == name {
			if t.Commit != c.Target {
				return fmt.Errorf("release tag points at another commit")
			}
			continue
		}
		prior = append(prior, t.Name)
	}
	if version.Latest(prior, p.Config.Release.TagPrefix) != c.BaseTag {
		return fmt.Errorf("candidate is stale: latest release tag changed")
	}
	if c.BaseTag != "" {
		base, exists, e := p.Store.Tag(ctx, c.BaseTag)
		if e != nil {
			return e
		}
		if !exists || base.Commit != c.BaseCommit {
			return fmt.Errorf("candidate base tag moved or disappeared")
		}
	}
	tag, exists, e := p.Store.Tag(ctx, name)
	if e != nil {
		return e
	}
	if exists && tag.Commit != c.Target {
		return fmt.Errorf("release tag conflicts with candidate")
	}
	want := scm.Release{Tag: name, Name: name, Description: c.Notes, Commit: c.Target}
	existing, released, e := p.Store.Release(ctx, name)
	if e != nil {
		return e
	}
	if released {
		if !exists || existing != want {
			return fmt.Errorf("existing release conflicts with candidate")
		}
		return nil
	}
	if dryRun {
		return nil
	}
	if !exists {
		if createErr := p.Store.CreateTag(ctx, scm.Tag{Name: name, Commit: c.Target}); createErr != nil {
			tag, ok, readErr := p.Store.Tag(ctx, name)
			if readErr != nil || !ok || tag.Commit != c.Target {
				return createErr
			}
		}
	}
	// A timed-out create may have succeeded; inspect rather than blindly retry it.
	if createErr := p.Store.CreateRelease(ctx, want); createErr != nil {
		v, ok, readErr := p.Store.Release(ctx, name)
		if readErr != nil || !ok || v != want {
			return createErr
		}
	}
	finalTag, ok, e := p.Store.Tag(ctx, name)
	if e != nil {
		return e
	}
	if !ok || finalTag.Commit != c.Target {
		return fmt.Errorf("final tag verification failed")
	}
	finalRelease, ok, e := p.Store.Release(ctx, name)
	if e != nil {
		return e
	}
	if !ok || finalRelease != want {
		return fmt.Errorf("final release verification failed")
	}
	return nil
}
