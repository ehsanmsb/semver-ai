package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/config"
	"go.yaml.in/yaml/v3"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIEndToEnd(t *testing.T) {
	d := t.TempDir()
	t.Chdir(d)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatal(e, string(out))
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.name", "test")
	git("config", "user.email", "test@example.com")
	if e := os.WriteFile("api.go", []byte("new API"), 0600); e != nil {
		t.Fatal(e)
	}
	git("add", "api.go")
	git("commit", "-m", "feat: new API")
	head := git("rev-parse", "HEAD")
	class := "FEATURE"
	tagExists := false
	releaseExists := false
	notes := ""
	mutations := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/chat/completions":
			content := fmt.Sprintf(`{"changes":[{"classification":%q,"explanation":"Add API","evidence":"new API","files":["api.go"],"confidence":0.95,"compatibility":"","uncertainty":""}]}`, class)
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]string{"content": content}}}})
		case r.URL.Path == "/api/v4/projects/42":
			fmt.Fprint(w, `{"id":42,"path_with_namespace":"group/project","default_branch":"main"}`)
		case strings.HasSuffix(r.URL.Path, "/repository/branches/main"):
			fmt.Fprintf(w, `{"commit":{"id":%q}}`, head)
		case strings.HasSuffix(r.URL.Path, "/repository/tags"):
			if r.Method == "POST" {
				tagExists = true
				mutations++
				fmt.Fprintf(w, `{"name":"v0.1.0","commit":{"id":%q}}`, head)
			} else if tagExists {
				fmt.Fprintf(w, `[{"name":"v0.1.0","commit":{"id":%q}}]`, head)
			} else {
				fmt.Fprint(w, `[]`)
			}
		case strings.HasSuffix(r.URL.Path, "/repository/tags/v0.1.0"):
			if !tagExists {
				w.WriteHeader(404)
			} else {
				fmt.Fprintf(w, `{"name":"v0.1.0","commit":{"id":%q}}`, head)
			}
		case strings.HasSuffix(r.URL.Path, "/releases") && r.Method == "POST":
			var body struct {
				Description string `json:"description"`
			}
			if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
				t.Error(e)
			}
			notes = body.Description
			releaseExists = true
			mutations++
			fmt.Fprint(w, `{}`)
		case strings.HasSuffix(r.URL.Path, "/releases/v0.1.0"):
			if !releaseExists {
				w.WriteHeader(404)
			} else {
				json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.0", "name": "v0.1.0", "description": notes, "commit": map[string]string{"id": head}})
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	cfg := config.Defaults()
	cfg.AI.Model = "test"
	cfg.AI.BaseURL = srv.URL + "/v1"
	cfg.GitLab.APIURL = srv.URL + "/api/v4"
	data, e := yaml.Marshal(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(".release-manager.yml", data, 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("SEMVER_AI_API_KEY", "ai-key")
	t.Setenv("GITLAB_TOKEN", "gitlab-key")
	t.Setenv("CI_PROJECT_ID", "42")
	t.Setenv("CI_COMMIT_SHA", head)
	command := func(want int, args ...string) {
		t.Helper()
		var out, err bytes.Buffer
		if got := run(args, &out, &err); got != want {
			t.Fatalf("%v exit=%d want=%d\n%s\n%s", args, got, want, out.String(), err.String())
		}
	}
	command(0, "predict", "--json")
	command(0, "prepare-release", "--dry-run")
	if _, e = os.Stat("semver-release-candidate.json"); !os.IsNotExist(e) {
		t.Fatal("dry run wrote artifact")
	}
	command(0, "prepare-release", "--json")
	if _, e = os.Stat("semver-release-candidate.json"); e != nil {
		t.Fatal(e)
	}
	class = "DOCS"
	command(0, "prepare-release", "--json")
	if _, e = os.Stat("semver-release-candidate.json"); !os.IsNotExist(e) {
		t.Fatal("stale artifact survived NO_RELEASE")
	}
	class = "FEATURE"
	command(0, "prepare-release", "--json")
	command(1, "publish-release")
	if mutations != 0 {
		t.Fatal("unauthorized mutation")
	}
	t.Setenv("GITLAB_CI", "true")
	t.Setenv("CI_JOB_MANUAL", "true")
	t.Setenv("CI_COMMIT_REF_PROTECTED", "true")
	t.Setenv("CI_ENVIRONMENT_NAME", "production-release")
	command(0, "publish-release", "--dry-run")
	if mutations != 0 {
		t.Fatal("dry run mutated remote")
	}
	command(0, "publish-release", "--json")
	command(0, "publish-release", "--json")
	if mutations != 2 {
		t.Fatal("duplicate mutation", mutations)
	}
}
