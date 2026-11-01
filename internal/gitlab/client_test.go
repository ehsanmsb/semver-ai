package gitlab

import (
	"context"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/scm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient(t *testing.T) {
	posts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Error("token missing")
		}
		switch {
		case r.Method == "POST":
			posts++
			w.WriteHeader(503)
			fmt.Fprint(w, "test-token must not leak")
		case strings.HasSuffix(r.URL.Path, "/repository/tags/missing"):
			w.WriteHeader(404)
		case strings.HasSuffix(r.URL.Path, "/repository/tags"):
			if r.URL.Query().Get("page") == "1" {
				w.Header().Set("X-Next-Page", "2")
				fmt.Fprint(w, `[{"name":"v1.0.0","commit":{"id":"one"}}]`)
			} else {
				fmt.Fprint(w, `[{"name":"v1.1.0","commit":{"id":"two"}}]`)
			}
		default:
			w.WriteHeader(403)
			fmt.Fprint(w, "test-token")
		}
	}))
	defer srv.Close()
	c, e := New(srv.URL+"/api/v4", "test-token", "42")
	if e != nil {
		t.Fatal(e)
	}
	tags, e := c.Tags(context.Background())
	if e != nil || len(tags) != 2 {
		t.Fatal(tags, e)
	}
	_, exists, e := c.Tag(context.Background(), "missing")
	if e != nil || exists {
		t.Fatal(exists, e)
	}
	e = c.CreateTag(context.Background(), scm.Tag{Name: "v1", Commit: "one"})
	if e == nil || posts != 1 || strings.Contains(e.Error(), "test-token") {
		t.Fatal(e, posts)
	}
	_, e = c.Branch(context.Background(), "main")
	if e == nil || strings.Contains(e.Error(), "test-token") {
		t.Fatal(e)
	}
}
