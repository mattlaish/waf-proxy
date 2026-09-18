package openaiapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxResponseBytes = 1 << 20

// ResponsesRequest is the small subset of the OpenAI Responses API used by the
// WAF. Provider-side storage is always disabled and Structured Outputs can be
// constrained with a strict JSON Schema.
type ResponsesRequest struct {
	Model           string      `json:"model"`
	MaxOutputTokens int         `json:"max_output_tokens"`
	Store           bool        `json:"store"`
	Input           []Message   `json:"input"`
	Text            *TextOutput `json:"text,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TextOutput struct {
	Format JSONSchemaFormat `json:"format"`
}

type JSONSchemaFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

func NewResponsesRequest(model string, maxOutputTokens int, system, user, schemaName string, schema map[string]any) ResponsesRequest {
	req := ResponsesRequest{
		Model:           model,
		MaxOutputTokens: maxOutputTokens,
		Store:           false,
		Input: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	if schema != nil {
		req.Text = &TextOutput{Format: JSONSchemaFormat{
			Type:   "json_schema",
			Name:   schemaName,
			Strict: true,
			Schema: schema,
		}}
	}
	return req
}

// CallResponses executes one synchronous Responses API request. apiKey is
// caller-owned; the caller remains responsible for zeroing it after return.
// Error responses deliberately report status only instead of reflecting remote
// response bodies into local logs.
func CallResponses(ctx context.Context, client *http.Client, baseURL string, apiKey []byte, reqBody ResponsesRequest) (string, error) {
	if client == nil {
		return "", errors.New("OpenAI HTTP client is not configured")
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal OpenAI Responses request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/responses", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if len(apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+string(apiKey))
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", err
	}
	if len(b) > maxResponseBytes {
		return "", errors.New("OpenAI Responses response exceeds size limit")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("OpenAI Responses HTTP %d", resp.StatusCode)
	}
	return ExtractResponseText(b)
}

func ExtractResponseText(b []byte) (string, error) {
	var out struct {
		Status            string `json:"status"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text"`
				Refusal string `json:"refusal"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("unexpected OpenAI Responses response: %w", err)
	}
	if out.Status == "incomplete" {
		reason := "unknown"
		if out.IncompleteDetails != nil && out.IncompleteDetails.Reason != "" {
			reason = out.IncompleteDetails.Reason
		}
		return "", fmt.Errorf("OpenAI Responses response incomplete: %s", reason)
	}
	if out.Status != "completed" {
		return "", fmt.Errorf("unexpected OpenAI Responses status %q", out.Status)
	}
	for _, item := range out.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			switch content.Type {
			case "refusal":
				return "", fmt.Errorf("OpenAI Responses refusal: %s", truncate(content.Refusal, 160))
			case "output_text":
				if strings.TrimSpace(content.Text) != "" {
					return content.Text, nil
				}
			}
		}
	}
	return "", errors.New("unexpected OpenAI Responses response: no output_text")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
