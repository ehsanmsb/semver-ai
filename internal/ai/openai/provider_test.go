package openai

import (
	"context"
	"encoding/json"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProvider(t *testing.T) {
	good := `{"changes":[{"classification":"FIX","explanation":"repair","evidence":"fixed","files":["a.go"],"confidence":0.95,"compatibility":"","uncertainty":""}]}`
	for _, tc := range []struct {
		name, content, finish string
		status                int
		wantErr               bool
	}{{"valid", good, "stop", 200, false}, {"malformed", "oops", "stop", 200, true}, {"unknown fields", `{"changes":[],"version":"1.0.0"}`, "stop", 200, true}, {"duplicate", `{"changes":[],"changes":[]}`, "stop", 200, true}, {"incomplete", good, "length", 200, true}, {"unavailable", "secret-value", "", 503, true}} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-key" || r.URL.Path != "/v1/chat/completions" {
					t.Error("request contract")
				}
				w.WriteHeader(tc.status)
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": tc.finish, "message": map[string]string{"content": tc.content}}}})
			}))
			defer srv.Close()
			p := Provider{srv.URL + "/v1", "test", "test-key", time.Second, 30000}
			_, e := p.Classify(context.Background(), ai.Request{})
			if (e != nil) != tc.wantErr {
				t.Fatal(e)
			}
			if e != nil && strings.Contains(e.Error(), "secret-value") {
				t.Fatal("leaked error body")
			}
		})
	}
}
