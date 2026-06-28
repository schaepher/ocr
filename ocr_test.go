package ocr

import (
	"image"
	"testing"

	"github.com/schaepher/ocr/decoder"
	"github.com/schaepher/ocr/document"
	"github.com/schaepher/ocr/provider"
)

// mockProvider implements provider.Provider for tests.
type mockProvider struct{}

func (p *mockProvider) DefaultModel() string     { return "test-model" }
func (p *mockProvider) SystemPrompt() string     { return "test-prompt" }
func (p *mockProvider) Decoder() decoder.Decoder { return &mockDecoder{} }

type mockDecoder struct{}

func (d *mockDecoder) Decode(raw string, imgSize image.Point) (*document.Document, error) {
	return &document.Document{}, nil
}

var _ provider.Provider = (*mockProvider)(nil)

func TestNew(t *testing.T) {
	c := New(&mockProvider{})
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.model != "test-model" {
		t.Errorf("model = %q, want %q", c.model, "test-model")
	}
	if c.systemPrompt != "test-prompt" {
		t.Errorf("systemPrompt = %q, want %q", c.systemPrompt, "test-prompt")
	}
}

func TestLMStudio(t *testing.T) {
	c := New(&mockProvider{}).LMStudio("http://localhost:8080/v1")
	if c.baseURL != "http://localhost:8080/v1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:8080/v1")
	}
}

func TestModel(t *testing.T) {
	c := New(&mockProvider{}).Model("override-model")
	if c.model != "override-model" {
		t.Errorf("model = %q, want %q", c.model, "override-model")
	}
}

func TestSystemPrompt(t *testing.T) {
	c := New(&mockProvider{}).SystemPrompt("custom prompt")
	if c.systemPrompt != "custom prompt" {
		t.Errorf("systemPrompt = %q, want %q", c.systemPrompt, "custom prompt")
	}
}

func TestDefaults(t *testing.T) {
	c := New(&mockProvider{})
	if c.baseURL == "" {
		t.Error("default baseURL should not be empty")
	}
	if c.model == "" {
		t.Error("default model should not be empty")
	}
}
