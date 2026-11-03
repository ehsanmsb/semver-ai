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
	MaxCompletionTokens int
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
	limit := p.MaxCompletionTokens
	if limit == 0 {
		limit = 4096
	}
	if limit < 1 || limit > 131072 {
		return result, fmt.Errorf("invalid AI completion token limit")
	}
	data, e := json.Marshal(input)
	if e != nil {
		return result, e
	}
	responseSchema := schema()
	schemaJSON, e := json.Marshal(responseSchema)
	if e != nil {
		return result, e
	}
	systemPrompt := prompt + "\nThe exact output schema is below. Return one object whose only top-level key is changes, never files. Do not wrap it in Markdown. Every finding must have all required fields.\n" + string(schemaJSON)
	body, e := json.Marshal(map[string]any{"model": p.Model, "messages": []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": string(data)}}, "response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "semantic_analysis_v1", "strict": true, "schema": responseSchema}}, "max_completion_tokens": limit})
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
		Usage struct {
			CompletionTokens *int `json:"completion_tokens"`
			Details          struct {
				ReasoningTokens *int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
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
	if choice.Message.Refusal != "" {
		return result, fmt.Errorf("AI provider refused classification; no analysis accepted (refusal text omitted)")
	}
	switch choice.Finish {
	case "stop":
	case "length":
		usage := ""
		if n := envelope.Usage.CompletionTokens; n != nil && *n >= 0 {
			usage += fmt.Sprintf(", completion_tokens=%d", *n)
		}
		if n := envelope.Usage.Details.ReasoningTokens; n != nil && *n >= 0 {
			usage += fmt.Sprintf(", reasoning_tokens=%d", *n)
		}
		return result, fmt.Errorf("AI response incomplete: finish_reason=length, max_completion_tokens=%d%s; increase ai.max_completion_tokens within model limits (budget includes reasoning); no analysis accepted", limit, usage)
	case "content_filter":
		return result, fmt.Errorf("AI response blocked: finish_reason=content_filter; no analysis accepted")
	case "tool_calls", "function_call":
		return result, fmt.Errorf("AI returned an unsupported tool/function call instead of classification JSON")
	default:
		return result, fmt.Errorf("AI response has missing or unsupported finish_reason; check OpenAI-compatible gateway behavior")
	}
	if strings.TrimSpace(choice.Message.Content) == "" {
		return result, fmt.Errorf("AI returned empty content with finish_reason=stop")
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
