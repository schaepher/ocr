// Package provider defines the interface for OCR model providers.
// Each provider encapsulates the model name, system prompt, and decoder
// for a specific OCR model (e.g., PaddleOCR-VL, Qwen2.5-VL, etc.).
package provider

import (
	"github.com/schaepher/ocr/decoder"
)

// Provider is the interface that OCR model providers must implement.
type Provider interface {
	// DefaultModel returns the default model name (e.g. "PaddleOCR-VL-1.6").
	DefaultModel() string
	// SystemPrompt returns the system prompt for the OCR task.
	SystemPrompt() string
	// Decoder returns the decoder that parses this model's output format.
	Decoder() decoder.Decoder
}
