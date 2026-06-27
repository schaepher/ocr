package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Defaults
const (
	DefaultBaseURL = "http://127.0.0.1:1234/v1"
	DefaultModel   = "PaddleOCR-VL-1.6"
)

// Message represents a single message in the chat completion request.
type Message struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// ContentPart is a part of a message content (text or image_url).
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents a base64-encoded image or URL.
type ImageURL struct {
	URL string `json:"url"`
}

// ChatRequest is the request body for POST /v1/chat/completions.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// ChatResponse is the non-streaming response from /v1/chat/completions.
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Message ResponseMessage `json:"message"`
}

// ResponseMessage is the assistant's response message.
type ResponseMessage struct {
	Content string `json:"content"`
}

// ChatStreamChunk is a single SSE chunk from a streaming response.
type ChatStreamChunk struct {
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice represents a choice in a streaming chunk.
type StreamChoice struct {
	Delta  StreamDelta `json:"delta"`
	Finish string      `json:"finish_reason"`
}

// StreamDelta contains the incremental content delta.
type StreamDelta struct {
	Content string `json:"content"`
}

// APIError represents an API error response.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s (type=%s, code=%d)", e.Message, e.Type, e.Code)
}

// Client is an OpenAI-compatible HTTP client for LM Studio / Ollama / etc.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// Option is a functional option for the Client.
type Option func(*Client)

// WithBaseURL sets the API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(url, "/") }
}

// WithModel sets the model name.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new Client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		model:      DefaultModel,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Chat sends a chat completion request and returns the response text.
// If stream is true, it reads the SSE stream and concatenates deltas.
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (string, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	if req.Stream {
		return c.chatStream(ctx, req)
	}
	return c.chatOnce(ctx, req)
}

// chatOnce sends a non-streaming chat completion request.
func (c *Client) chatOnce(ctx context.Context, req *ChatRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if chatResp.Error != nil {
		return "", chatResp.Error
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response: no choices returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// chatStream sends a streaming request and concatenates SSE deltas.
func (c *Client) chatStream(ctx context.Context, req *ChatRequest) (string, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		var chunk ChatStreamChunk
		data := line[6:] // strip "data: "
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", fmt.Errorf("decode stream chunk: %w", err)
		}

		for _, choice := range chunk.Choices {
			full.WriteString(choice.Delta.Content)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read stream: %w", err)
	}

	return full.String(), nil
}

// ImageToBase64 reads an image file and returns its base64 data URI.
func ImageToBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read image file: %w", err)
	}

	mimeType := detectMimeType(path, data)
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
}

// detectMimeType guesses the MIME type from the file extension or magic bytes.
func detectMimeType(path string, data []byte) string {
	lower := strings.ToLower(path)

	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	default:
		// Fall back to detecting magic bytes
		if len(data) >= 8 {
			if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
				return "image/png"
			}
			if bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
				return "image/jpeg"
			}
		}
		return "image/png" // safest default for VLM models
	}
}

// BuildVisionRequest creates a chat request with an image and optional text prompt.
func BuildVisionRequest(model, systemPrompt, userPrompt, imageURI string) *ChatRequest {
	msg := Message{Role: "user", Content: []ContentPart{
		{Type: "text", Text: userPrompt},
		{Type: "image_url", ImageURL: &ImageURL{URL: imageURI}},
	}}

	if systemPrompt != "" {
		return &ChatRequest{
			Model: model,
			Messages: []Message{
				{Role: "system", Content: []ContentPart{{Type: "text", Text: systemPrompt}}},
				msg,
			},
		}
	}

	return &ChatRequest{
		Model:    model,
		Messages: []Message{msg},
	}
}
