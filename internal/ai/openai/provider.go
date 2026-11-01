package openai

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/ehsanmsb/semver-ai/internal/ai"
	"github.com/ehsanmsb/semver-ai/internal/safejson"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed classify-v1.txt
var prompt string

type Provider struct {
	BaseURL, Model, Key string
	Timeout             time.Duration
	MaxTokens           int
}

func schema() map[string]any {
	str := map[string]any{"type": "string"}
	props := map[string]any{"classification": map[string]any{"type": "string", "enum": []string{"BREAKING", "FEATURE", "FIX", "PERFORMANCE", "REFACTOR", "DOCS", "CHORE", "TEST", "UNKNOWN"}}, "explanation": str, "evidence": str, "files": map[string]any{"type": "array", "items": str}, "confidence": map[string]any{"type": "number"}, "compatibility": str, "uncertainty": str}
	finding := map[string]any{"type": "object", "additionalProperties": false, "properties": props, "required": []string{"classification", "explanation", "evidence", "files", "confidence", "compatibility", "uncertainty"}}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"changes": map[string]any{"type": "array", "items": finding}}, "required": []string{"changes"}}
}
func (p Provider) Classify(ctx context.Context, input ai.Request) (ai.Result, error) {
	var result ai.Result
	if p.Key == "" {
		return result, fmt.Errorf("AI credential is missing")
	}
	data, e := json.Marshal(input)
	if e != nil {
		return result, e
	}
	body, e := json.Marshal(map[string]any{"model": p.Model, "messages": []map[string]string{{"role": "system", "content": prompt}, {"role": "user", "content": string(data)}}, "response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "semantic_analysis_v1", "strict": true, "schema": schema()}}, "max_completion_tokens": 4096})
	if e != nil {
		return result, e
	}
	// One byte per input token is a deliberately conservative provider-independent bound.
	if len(body) > p.MaxTokens {
		return result, fmt.Errorf("serialized AI request exceeds conservative context budget; increase max_context_tokens")
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if e != nil {
		return result, fmt.Errorf("invalid AI endpoint")
	}
	req.Header.Set("Authorization", "Bearer "+p.Key)
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: p.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := client.Do(req)
	if e != nil {
		return result, fmt.Errorf("AI request failed (network or timeout)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("AI request failed: HTTP %d", resp.StatusCode)
	}
	data, e = io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if e != nil || len(data) > 2*1024*1024 {
		return result, fmt.Errorf("AI response unreadable or too large")
	}
	var envelope struct {
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if e = json.Unmarshal(data, &envelope); e != nil || len(envelope.Choices) != 1 {
		return result, fmt.Errorf("invalid AI completion envelope")
	}
	choice := envelope.Choices[0]
	if choice.Finish != "stop" || choice.Message.Refusal != "" {
		return result, fmt.Errorf("AI response refused or incomplete")
	}
	if e = safejson.Decode([]byte(choice.Message.Content), &result); e != nil {
		return result, fmt.Errorf("invalid AI analysis JSON")
	}
	var object map[string][]map[string]json.RawMessage
	if e = json.Unmarshal([]byte(choice.Message.Content), &object); e != nil {
		return result, fmt.Errorf("invalid AI schema")
	}
	for _, finding := range object["changes"] {
		for _, key := range []string{"classification", "explanation", "evidence", "files", "confidence", "compatibility", "uncertainty"} {
			raw, ok := finding[key]
			if !ok || string(raw) == "null" {
				return result, fmt.Errorf("missing or null AI field")
			}
		}
	}
	return result, nil
}
