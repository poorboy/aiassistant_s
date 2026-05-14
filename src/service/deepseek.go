package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"aiass/src/config"
)

type DeepSeekClient struct {
	apiKey   string
	baseURL  string
	model    string
	proxyURL string
	provider string
	http     *http.Client
}

type deepseekReq struct {
	Model       string            `json:"model"`
	Messages    []DeepSeekMessage `json:"messages"`
	Stream      bool              `json:"stream"`
	Temperature float64           `json:"temperature"`
	Tools       []DeepSeekTool    `json:"tools,omitempty"`
}

type DeepSeekTool struct {
	Type     string       `json:"type"`
	Function DeepSeekFunc `json:"function"`
}

type DeepSeekFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type DeepSeekMessage struct {
	Role             string             `json:"role"`
	Content          string             `json:"content"`
	ReasoningContent string             `json:"reasoning_content,omitempty"`
	ToolCalls        []DeepSeekToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string             `json:"tool_call_id,omitempty"`
}

type DeepSeekToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function DeepSeekFuncCall `json:"function"`
}

type DeepSeekFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type DeepSeekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string             `json:"content"`
			ReasoningContent string             `json:"reasoning_content"`
			ToolCalls        []DeepSeekToolCall `json:"tool_calls,omitempty"`
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

func NewDeepSeekClientFromSettings(apiKey, baseURL, model, proxyURL string) *DeepSeekClient {
	client := &DeepSeekClient{
		apiKey:   apiKey,
		baseURL:  baseURL,
		model:    model,
		proxyURL: proxyURL,
		provider: detectProvider(baseURL),
	}
	client.http = newHTTPClient(proxyURL)
	return client
}

func newHTTPClient(proxyURL string) *http.Client {
	if proxyURL == "" {
		return &http.Client{}
	}
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return &http.Client{}
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
	}
}

func detectProvider(baseURL string) string {
	baseURL = strings.ToLower(baseURL)
	switch {
	case strings.Contains(baseURL, "generativelanguage.googleapis.com"):
		return "gemini"
	case strings.Contains(baseURL, "api.anthropic.com"):
		return "anthropic"
	case strings.Contains(baseURL, "localhost:11434"):
		return "ollama"
	case strings.Contains(baseURL, "localhost:8080"):
		return "llamacpp"
	case strings.Contains(baseURL, "aip.baidubce.com"):
		return "baidu"
	case strings.Contains(baseURL, "api.hunyuan.cloud.tencent.com"):
		return "tencent"
	case strings.Contains(baseURL, "open.bigmodel.cn"):
		return "zhipu"
	case strings.Contains(baseURL, "dashscope.aliyuncs.com"):
		return "aliyun"
	default:
		return "openai"
	}
}

func (d *DeepSeekClient) ChatStream(messages []DeepSeekMessage, onChunk func(string)) (string, error) {
	return d.ChatStreamWithTools(messages, nil, func(chunk string, _ []DeepSeekToolCall) {
		onChunk(chunk)
	}, nil)
}

func (d *DeepSeekClient) ChatStreamWithTools(messages []DeepSeekMessage, tools []DeepSeekTool, onEvent func(string, []DeepSeekToolCall), reasoningContent *string) (string, error) {
	provider := d.provider
	isGemini := provider == "gemini"
	isAnthropic := provider == "anthropic"
	isOllama := provider == "ollama"
	isBaidu := provider == "baidu"

	var reqBody []byte
	var err error
	var fullURL string

	if isGemini {
		fullURL = d.baseURL + "/v1beta/models/" + d.model + ":streamGenerateContent?alt=sse"
		geminiBody := buildGeminiRequestBody(messages)
		reqBody, _ = json.Marshal(geminiBody)
	} else if isAnthropic {
		fullURL = d.baseURL + "/v1/messages"
		anthropicBody := buildAnthropicRequestBody(messages, d.model, tools)
		reqBody, _ = json.Marshal(anthropicBody)
	} else if isOllama {
		fullURL = d.baseURL + "/api/chat"
		ollamaBody := buildOllamaRequestBody(messages, d.model, tools)
		reqBody, _ = json.Marshal(ollamaBody)
	} else if isBaidu {
		token, err := getBaiduAccessToken(d.apiKey, d.http)
		if err != nil {
			return "", fmt.Errorf("baidu access token: %w", err)
		}
		fullURL = fmt.Sprintf("%s/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions_pro?access_token=%s", strings.TrimRight(d.baseURL, "/"), token)
		baiduBody := buildBaiduRequestBody(messages, tools)
		reqBody, _ = json.Marshal(baiduBody)
	} else {
		fullURL = d.baseURL + "/v1/chat/completions"
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
		reqBody, _ = json.Marshal(body)
	}

	log.Printf("[DeepSeek] POST %s (model=%s, provider=%s, proxy=%q)", fullURL, d.model, provider, d.proxyURL)
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	switch {
	case isGemini:
		req.Header.Set("X-goog-api-key", d.apiKey)
	case isAnthropic:
		req.Header.Set("x-api-key", d.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case isBaidu:
	default:
		req.Header.Set("Authorization", "Bearer "+d.apiKey)
	}

	resp, err := d.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("[DeepSeek] status=%d body=%s", resp.StatusCode, string(respBody))
		return "", fmt.Errorf("deepseek api error %d: %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	var pendingToolCalls []DeepSeekToolCall

	switch {
	case isAnthropic:
		fullContent.WriteString(parseAnthropicStream(resp.Body, onEvent))
		return fullContent.String(), nil
	case isOllama:
		fullContent.WriteString(parseOllamaStream(resp.Body, onEvent))
		return fullContent.String(), nil
	case isGemini:
		fullContent.WriteString(parseGeminiStream(resp.Body))
		onEvent(fullContent.String(), nil)
		return fullContent.String(), nil
	default:
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
				if reasoningContent != nil && choice.Delta.ReasoningContent != "" {
					*reasoningContent += choice.Delta.ReasoningContent
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
}

func mergeOrAppendToolCall(calls *[]DeepSeekToolCall, tc DeepSeekToolCall) {
	if tc.ID != "" {
		for i, c := range *calls {
			if c.ID == tc.ID {
				if tc.Function.Name != "" && c.Function.Name == "" {
					(*calls)[i].Function.Name = tc.Function.Name
				}
				(*calls)[i].Function.Arguments += tc.Function.Arguments
				return
			}
		}
		*calls = append(*calls, tc)
		return
	}
	if tc.Function.Arguments != "" && len(*calls) > 0 {
		lastIdx := len(*calls) - 1
		(*calls)[lastIdx].Function.Arguments += tc.Function.Arguments
		return
	}
}

func (d *DeepSeekClient) TestConnection() error {
	provider := d.provider
	isGemini := provider == "gemini"
	isAnthropic := provider == "anthropic"
	isOllama := provider == "ollama"
	isBaidu := provider == "baidu"

	var reqBody []byte
	var fullURL string

	if isGemini {
		fullURL = d.baseURL + "/v1beta/models/" + d.model + ":generateContent"
		reqBody, _ = json.Marshal(map[string]interface{}{
			"contents": []map[string]interface{}{
				{"parts": []map[string]string{{"text": "ping"}}},
			},
		})
	} else if isAnthropic {
		fullURL = d.baseURL + "/v1/messages"
		reqBody, _ = json.Marshal(map[string]interface{}{
			"model":      d.model,
			"max_tokens": 10,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		})
	} else if isOllama {
		fullURL = d.baseURL + "/api/generate"
		reqBody, _ = json.Marshal(map[string]interface{}{
			"model":  d.model,
			"prompt": "ping",
			"stream": false,
		})
	} else if isBaidu {
		token, err := getBaiduAccessToken(d.apiKey, d.http)
		if err != nil {
			return fmt.Errorf("baidu access token: %w", err)
		}
		fullURL = fmt.Sprintf("%s/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions_pro?access_token=%s", strings.TrimRight(d.baseURL, "/"), token)
		reqBody, _ = json.Marshal(map[string]interface{}{
			"messages": []map[string]string{{"role": "user", "content": "ping"}},
		})
	} else {
		fullURL = d.baseURL + "/v1/chat/completions"
		body := deepseekReq{
			Model:    d.model,
			Messages: []DeepSeekMessage{{Role: "user", Content: "ping"}},
			Stream:   false,
		}
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	switch {
	case isGemini:
		req.Header.Set("X-goog-api-key", d.apiKey)
	case isAnthropic:
		req.Header.Set("x-api-key", d.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case isBaidu:
	default:
		req.Header.Set("Authorization", "Bearer "+d.apiKey)
	}

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

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct {
	Text string `json:"text"`
}
type geminiReq struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
}

func buildGeminiRequestBody(messages []DeepSeekMessage) geminiReq {
	var contents []geminiContent
	var systemText string
	for _, m := range messages {
		if m.Role == "system" {
			if systemText != "" {
				systemText += "\n" + m.Content
			} else {
				systemText = m.Content
			}
			continue
		}
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}
	req := geminiReq{Contents: contents}
	if systemText != "" {
		req.SystemInstruction = &geminiContent{Parts: []geminiPart{{Text: systemText}}}
	}
	return req
}

type geminiStreamResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func parseGeminiStream(r io.Reader) string {
	var full strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk geminiStreamResp
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, c := range chunk.Candidates {
			for _, p := range c.Content.Parts {
				full.WriteString(p.Text)
			}
		}
	}
	return full.String()
}

type anthropicReq struct {
	Model       string                      `json:"model"`
	MaxTokens   int                         `json:"max_tokens"`
	System      string                      `json:"system,omitempty"`
	Messages    []anthropicMessage          `json:"messages"`
	Stream      bool                        `json:"stream"`
	Tools       []anthropicTool             `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

func buildAnthropicRequestBody(messages []DeepSeekMessage, model string, tools []DeepSeekTool) anthropicReq {
	var systemText string
	var msgs []anthropicMessage
	for _, m := range messages {
		if m.Role == "system" {
			if systemText != "" {
				systemText += "\n" + m.Content
			} else {
				systemText = m.Content
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		}
		msgs = append(msgs, anthropicMessage{Role: role, Content: m.Content})
	}
	anthropicTools := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		anthropicTools = append(anthropicTools, anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}
	req := anthropicReq{
		Model:     model,
		MaxTokens: 4096,
		Messages:  msgs,
		Stream:    true,
	}
	if systemText != "" {
		req.System = systemText
	}
	if len(anthropicTools) > 0 {
		req.Tools = anthropicTools
	}
	return req
}

type anthropicStreamChunk struct {
	Type  string `json:"type"`
	Delta struct {
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	ContentBlock struct {
		Text string `json:"text"`
	} `json:"content_block"`
}

func parseAnthropicStream(r io.Reader, onEvent func(string, []DeepSeekToolCall)) string {
	var full strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var chunk anthropicStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		switch chunk.Type {
		case "content_block_delta":
			if chunk.Delta.Text != "" {
				full.WriteString(chunk.Delta.Text)
				onEvent(chunk.Delta.Text, nil)
			}
		}
	}
	return full.String()
}

type ollamaReq struct {
	Model    string            `json:"model"`
	Messages []ollamaMessage   `json:"messages"`
	Stream   bool              `json:"stream"`
	Tools    []DeepSeekTool    `json:"tools,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func buildOllamaRequestBody(messages []DeepSeekMessage, model string, tools []DeepSeekTool) ollamaReq {
	var msgs []ollamaMessage
	for _, m := range messages {
		msgs = append(msgs, ollamaMessage{Role: m.Role, Content: m.Content})
	}
	req := ollamaReq{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	}
	if len(tools) > 0 {
		req.Tools = tools
	}
	return req
}

type ollamaStreamChunk struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func parseOllamaStream(r io.Reader, onEvent func(string, []DeepSeekToolCall)) string {
	var full strings.Builder
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		var chunk ollamaStreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Message.Content != "" {
			full.WriteString(chunk.Message.Content)
			onEvent(chunk.Message.Content, nil)
		}
		if chunk.Done {
			break
		}
	}
	return full.String()
}

type baiduReq struct {
	Messages []baiduMessage `json:"messages"`
	Stream   bool           `json:"stream"`
}

type baiduMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func buildBaiduRequestBody(messages []DeepSeekMessage, tools []DeepSeekTool) baiduReq {
	var msgs []baiduMessage
	for _, m := range messages {
		role := m.Role
		if role == "system" {
			continue
		}
		msgs = append(msgs, baiduMessage{Role: role, Content: m.Content})
	}
	return baiduReq{
		Messages: msgs,
		Stream:   true,
	}
}

func getBaiduAccessToken(apiKey string, httpClient *http.Client) (string, error) {
	parts := strings.SplitN(apiKey, "|", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("baidu api_key must be in format: client_id|client_secret")
	}
	clientID := parts[0]
	clientSecret := parts[1]
	urlStr := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
	resp, err := httpClient.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	json.Unmarshal(body, &result)
	if result.Error != "" {
		return "", fmt.Errorf("baidu oauth error: %s", result.Error)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("baidu oauth: empty access_token")
	}
	return result.AccessToken, nil
}
