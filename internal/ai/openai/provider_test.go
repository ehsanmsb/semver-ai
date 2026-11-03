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
	}{{"valid", good, "stop", 200, false}, {"gateway wrong root", `{"files":[]}`, "stop", 200, true}, {"malformed", "oops", "stop", 200, true}, {"unknown fields", `{"changes":[],"version":"1.0.0"}`, "stop", 200, true}, {"duplicate", `{"changes":[],"changes":[]}`, "stop", 200, true}, {"missing field", strings.Replace(good, `"uncertainty":""`, `"uncertainty":null`, 1), "stop", 200, true}, {"incomplete", good, "length", 200, true}, {"unavailable", "secret-value", "", 503, true}} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-key" || r.URL.Path != "/v1/chat/completions" {
					t.Error("request contract")
				}
				var request struct {
					Messages []struct {
						Content string `json:"content"`
					} `json:"messages"`
					Format struct {
						Schema struct {
							Value map[string]any `json:"schema"`
						} `json:"json_schema"`
					} `json:"response_format"`
				}
				if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
					t.Fatal(e)
				}
				if len(request.Messages) != 2 || !strings.Contains(request.Messages[0].Content, `"required":["changes"]`) || !strings.Contains(request.Messages[0].Content, `"additionalProperties":false`) || request.Format.Schema.Value["type"] != "object" {
					t.Error("schema missing from prompt or structured response format")
				}
				w.WriteHeader(tc.status)
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": tc.finish, "message": map[string]string{"content": tc.content}}}})
			}))
			defer srv.Close()
			p := Provider{BaseURL: srv.URL + "/v1", Model: "test", Key: "test-key", Timeout: time.Second, MaxTokens: 30000}
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

func TestCompletionDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, finish, refusal, content, want string }{
		{"length", "length", "", `{"changes":[]}`, "completion_tokens=4096, reasoning_tokens=4000"},
		{"refusal", "stop", "SECRET refusal body", "", "provider refused"},
		{"filter", "content_filter", "", "SECRET body", "finish_reason=content_filter"},
		{"tool", "tool_calls", "", "SECRET body", "unsupported tool/function"},
		{"unknown", "SECRET finish reason", "", "SECRET body", "unsupported finish_reason"},
		{"missing", "", "", "", "unsupported finish_reason"},
		{"empty", "stop", "", "", "empty content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if req["max_completion_tokens"] != float64(16384) {
					t.Errorf("budget not forwarded: %v", req["max_completion_tokens"])
				}
				json.NewEncoder(w).Encode(map[string]any{
					"choices": []any{map[string]any{"finish_reason": tc.finish, "message": map[string]string{"content": tc.content, "refusal": tc.refusal}}},
					"usage":   map[string]any{"completion_tokens": 4096, "completion_tokens_details": map[string]int{"reasoning_tokens": 4000}},
				})
			}))
			defer srv.Close()
			p := Provider{BaseURL: srv.URL, Model: "test", Key: "key", Timeout: time.Second, MaxTokens: 30000, MaxCompletionTokens: 16384}
			result, err := p.Classify(context.Background(), ai.Request{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if strings.Contains(err.Error(), "SECRET") || len(result.Changes) != 0 {
				t.Fatalf("untrusted response leaked/accepted: %v", err)
			}
		})
	}
}
