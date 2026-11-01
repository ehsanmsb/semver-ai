package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v %s", args, e, b)
	}
	return string(b)
}
func TestHistory(t *testing.T) {
	d := t.TempDir()
	gitCmd(t, d, "init", "-b", "main")
	gitCmd(t, d, "config", "user.name", "test")
	gitCmd(t, d, "config", "user.email", "test@example.com")
	write := func(s string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(d, "api.txt"), []byte(s), 0600); e != nil {
			t.Fatal(e)
		}
		gitCmd(t, d, "add", ".")
		gitCmd(t, d, "commit", "-m", s)
	}
	write("first")
	r := Repository{d}
	ctx := context.Background()
	s, e := r.Snapshot(ctx, "v")
	if e != nil || s.BaseTag != "" || len(s.Files) != 1 {
		t.Fatal(s, e)
	}
	gitCmd(t, d, "tag", "v1.9.0")
	write("feature")
	gitCmd(t, d, "tag", "-a", "v1.10.0", "-m", "annotated")
	gitCmd(t, d, "tag", "v9.0.0-rc.1")
	write("fix")
	s, e = r.Snapshot(ctx, "v")
	if e != nil || s.BaseTag != "v1.10.0" || s.BaseCommit == s.Target || s.Diff == "" {
		t.Fatal(s, e)
	}
	gitCmd(t, d, "checkout", "-b", "other")
	write("unreleased elsewhere")
	gitCmd(t, d, "tag", "v99.0.0")
	gitCmd(t, d, "checkout", "main")
	s, e = r.Snapshot(ctx, "v")
	if e != nil || s.BaseTag != "v1.10.0" {
		t.Fatal(s, e)
	}
	clone := filepath.Join(t.TempDir(), "shallow")
	gitCmd(t, d, "clone", "--depth=1", "file://"+d, clone)
	if e = (Repository{clone}).Complete(ctx); e == nil {
		t.Fatal("accepted shallow history")
	}
}
