package scm

import "context"

type Tag struct{ Name, Commit string }
type Release struct{ Tag, Name, Description, Commit string }
type Project struct {
	ID                  int64
	Path, DefaultBranch string
}
type Store interface {
	Project(context.Context) (Project, error)
	Branch(context.Context, string) (string, error)
	Tags(context.Context) ([]Tag, error)
	Tag(context.Context, string) (Tag, bool, error)
	CreateTag(context.Context, Tag) error
	Release(context.Context, string) (Release, bool, error)
	CreateRelease(context.Context, Release) error
}
