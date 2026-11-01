package gitlab

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	api "gitlab.com/gitlab-org/api/client-go/v3"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	api     *api.Client
	project string
}

func New(baseURL, token, project string) (*Client, error) {
	if token == "" || project == "" {
		return nil, fmt.Errorf("GitLab token and project ID/path are required")
	}
	c, e := api.NewClient(token, api.WithBaseURL(baseURL), api.WithHTTPClient(&http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}), api.WithCustomRetryMax(2), api.WithCustomRetryWaitMinMax(time.Second, 5*time.Second), api.WithOnlyIdempotentRetries(), api.WithCustomLogger(nil))
	if e != nil {
		return nil, fmt.Errorf("invalid GitLab client configuration")
	}
	return &Client{c, project}, nil
}
func apiError(op string, r *api.Response, e error) error {
	if e == nil {
		return nil
	}
	if r != nil {
		return fmt.Errorf("GitLab %s failed: HTTP %d", op, r.StatusCode)
	}
	return fmt.Errorf("GitLab %s failed (network or timeout)", op)
}
func (c *Client) Project(ctx context.Context) (scm.Project, error) {
	p, r, e := c.api.Projects.GetProject(c.project, nil, api.WithContext(ctx))
	if e != nil {
		return scm.Project{}, apiError("project", r, e)
	}
	return scm.Project{ID: p.ID, Path: p.PathWithNamespace, DefaultBranch: p.DefaultBranch}, nil
}
func (c *Client) Branch(ctx context.Context, name string) (string, error) {
	b, r, e := c.api.Branches.GetBranch(c.project, name, api.WithContext(ctx))
	if e != nil {
		return "", apiError("branch", r, e)
	}
	if b.Commit == nil {
		return "", fmt.Errorf("GitLab branch has no commit")
	}
	return b.Commit.ID, nil
}
func (c *Client) Tags(ctx context.Context) ([]scm.Tag, error) {
	var result []scm.Tag
	opt := &api.ListTagsOptions{ListOptions: api.ListOptions{PerPage: 100, Page: 1}}
	for {
		tags, r, e := c.api.Tags.ListTags(c.project, opt, api.WithContext(ctx))
		if e != nil {
			return nil, apiError("tags", r, e)
		}
		for _, t := range tags {
			if t.Commit == nil {
				return nil, fmt.Errorf("GitLab tag has no commit")
			}
			result = append(result, scm.Tag{Name: t.Name, Commit: t.Commit.ID})
		}
		if r.NextPage == 0 {
			break
		}
		if r.NextPage <= opt.Page {
			return nil, fmt.Errorf("invalid GitLab pagination")
		}
		opt.Page = r.NextPage
	}
	return result, nil
}
func (c *Client) Tag(ctx context.Context, name string) (scm.Tag, bool, error) {
	t, r, e := c.api.Tags.GetTag(c.project, name, api.WithContext(ctx))
	if r != nil && r.StatusCode == 404 {
		return scm.Tag{}, false, nil
	}
	if e != nil {
		return scm.Tag{}, false, apiError("tag", r, e)
	}
	if t.Commit == nil {
		return scm.Tag{}, false, fmt.Errorf("GitLab tag has no commit")
	}
	return scm.Tag{Name: t.Name, Commit: t.Commit.ID}, true, nil
}
func (c *Client) CreateTag(ctx context.Context, t scm.Tag) error {
	_, r, e := c.api.Tags.CreateTag(c.project, &api.CreateTagOptions{TagName: &t.Name, Ref: &t.Commit}, api.WithContext(ctx))
	return apiError("create tag", r, e)
}
func (c *Client) Release(ctx context.Context, tag string) (scm.Release, bool, error) {
	v, r, e := c.api.Releases.GetRelease(c.project, tag, api.WithContext(ctx))
	if r != nil && r.StatusCode == 404 {
		return scm.Release{}, false, nil
	}
	if e != nil {
		return scm.Release{}, false, apiError("release", r, e)
	}
	if v.Commit.ID == "" {
		return scm.Release{}, false, fmt.Errorf("GitLab release has no commit")
	}
	return scm.Release{Tag: v.TagName, Name: v.Name, Description: v.Description, Commit: v.Commit.ID}, true, nil
}
func (c *Client) CreateRelease(ctx context.Context, v scm.Release) error {
	_, r, e := c.api.Releases.CreateRelease(c.project, &api.CreateReleaseOptions{TagName: &v.Tag, Name: &v.Name, Description: &v.Description}, api.WithContext(ctx))
	return apiError("create release", r, e)
}

type MR struct {
	Title, Description, URL, Base, Head, TargetBranch, Diff, Commits string
	Files                                                            []string
}

func (c *Client) MergeRequest(ctx context.Context, iid int64) (MR, error) {
	var result MR
	m, r, e := c.api.MergeRequests.GetMergeRequest(c.project, iid, nil, api.WithContext(ctx))
	if e != nil {
		return result, apiError("merge request", r, e)
	}
	result = MR{Title: m.Title, Description: m.Description, URL: m.WebURL, Base: m.DiffRefs.BaseSha, Head: m.DiffRefs.HeadSha, TargetBranch: m.TargetBranch}
	if result.Base == "" || result.Head == "" {
		return result, fmt.Errorf("merge request diff refs are not ready")
	}
	opt := &api.ListMergeRequestDiffsOptions{ListOptions: api.ListOptions{PerPage: 100, Page: 1}}
	for {
		diffs, r, e := c.api.MergeRequests.ListMergeRequestDiffs(c.project, iid, opt, api.WithContext(ctx))
		if e != nil {
			return result, apiError("MR diffs", r, e)
		}
		for _, d := range diffs {
			if d.TooLarge || d.Collapsed {
				return result, fmt.Errorf("GitLab MR diff is truncated or collapsed")
			}
			result.Files = append(result.Files, d.NewPath)
			result.Diff += "diff --git a/" + d.OldPath + " b/" + d.NewPath + "\n" + d.Diff + "\n"
			if len(result.Diff) > 10000000 {
				return result, fmt.Errorf("MR diff exceeds hard limit")
			}
		}
		if r.NextPage == 0 {
			break
		}
		if r.NextPage <= opt.Page {
			return result, fmt.Errorf("invalid GitLab pagination")
		}
		opt.Page = r.NextPage
	}
	co := &api.GetMergeRequestCommitsOptions{ListOptions: api.ListOptions{PerPage: 100, Page: 1}}
	for {
		commits, r, e := c.api.MergeRequests.GetMergeRequestCommits(c.project, iid, co, api.WithContext(ctx))
		if e != nil {
			return result, apiError("MR commits", r, e)
		}
		for _, commit := range commits {
			result.Commits += commit.ID + "\n" + commit.Message + "\n"
		}
		if len(result.Commits) > 1000000 {
			return result, fmt.Errorf("MR commit metadata exceeds hard limit")
		}
		if r.NextPage == 0 {
			break
		}
		if r.NextPage <= co.Page {
			return result, fmt.Errorf("invalid GitLab pagination")
		}
		co.Page = r.NextPage
	}
	return result, nil
}
func (c *Client) Comment(ctx context.Context, iid int64, body string) error {
	body = strings.TrimSpace(body)
	_, r, e := c.api.Notes.CreateMergeRequestNote(c.project, iid, &api.CreateMergeRequestNoteOptions{Body: &body}, api.WithContext(ctx))
	return apiError("MR comment", r, e)
}
