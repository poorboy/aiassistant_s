package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"aiass/src/config"
)

type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

type deepseekReq struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	Stream      bool              `json:"stream"`
	Temperature float64           `json:"temperature"`
	Tools       []DeepSeekTool    `json:"tools,omitempty"`
}

type DeepSeekTool struct {
	Type     string          `json:"type"`
	Function DeepSeekFunc    `json:"function"`
}

type DeepSeekFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type DeepSeekMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []DeepSeekToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type DeepSeekToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function DeepSeekFuncCall `json:"function"`
}

type DeepSeekFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type DeepSeekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content    string             `json:"content"`
			ToolCalls  []DeepSeekToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func NewDeepSeekClient(cfg *config.Config) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey:  cfg.DeepSeekKey,
		baseURL: cfg.DeepSeekURL,
		model:   cfg.DeepSeekModel,
		http:    &http.Client{},
	}
}

func NewDeepSeekClientFromSettings(apiKey, baseURL, model string) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{},
	}
}

func (d *DeepSeekClient) ChatStream(messages []DeepSeekMessage, onChunk func(string)) (string, error) {
	return d.ChatStreamWithTools(messages, nil, func(chunk string, _ []DeepSeekToolCall) {
		onChunk(chunk)
	})
}

func (d *DeepSeekClient) ChatStreamWithTools(messages []DeepSeekMessage, tools []DeepSeekTool, onEvent func(string, []DeepSeekToolCall)) (string, error) {
	body := deepseekReq{
		Model:       d.model,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.7,
		Tools:       tools,
	}

	if len(tools) == 0 {
		body.Tools = nil
	}

	reqBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", d.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("deepseek api error %d: %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	var pendingToolCalls []DeepSeekToolCall
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk DeepSeekStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			content := choice.Delta.Content
			if content != "" {
				fullContent.WriteString(content)
				onEvent(content, nil)
			}
			if calls := choice.Delta.ToolCalls; len(calls) > 0 {
				for _, tc := range calls {
					mergeOrAppendToolCall(&pendingToolCalls, tc)
				}
			}
		}
	}

	if len(pendingToolCalls) > 0 {
		onEvent("", pendingToolCalls)
	}
	return fullContent.String(), nil
}

func mergeOrAppendToolCall(calls *[]DeepSeekToolCall, tc DeepSeekToolCall) {
	for i, c := range *calls {
		if c.ID == tc.ID {
			if tc.Function.Name != "" {
				(*calls)[i].Function.Name = tc.Function.Name
			}
			(*calls)[i].Function.Arguments += tc.Function.Arguments
			return
		}
	}
	*calls = append(*calls, tc)
}

func (d *DeepSeekClient) TestConnection() error {
	body := deepseekReq{
		Model:    d.model,
		Messages: []DeepSeekMessage{{Role: "user", Content: "ping"}},
		Stream:   false,
	}
	reqBody, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", d.baseURL+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("api returned %d", resp.StatusCode)
	}
	return nil
}
