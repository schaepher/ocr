package paddleocrvl

import (
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLMStudio(t *testing.T) {
	c := New().LMStudio("http://localhost:8080/v1")
	if c.baseURL != "http://localhost:8080/v1" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:8080/v1")
	}
}

func TestModel(t *testing.T) {
	c := New().Model("test-model")
	if c.model != "test-model" {
		t.Errorf("model = %q, want %q", c.model, "test-model")
	}
}

func TestSystemPrompt(t *testing.T) {
	c := New().SystemPrompt("custom prompt")
	if c.systemPrompt != "custom prompt" {
		t.Errorf("systemPrompt = %q, want %q", c.systemPrompt, "custom prompt")
	}
}

func TestDefaults(t *testing.T) {
	c := New()
	if c.baseURL == "" {
		t.Error("default baseURL should not be empty")
	}
	if c.model == "" {
		t.Error("default model should not be empty")
	}
}
