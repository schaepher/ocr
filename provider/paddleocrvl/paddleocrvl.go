// Package paddleocrvl implements the provider.Provider interface for
// PaddleOCR-VL models, which output text with <|LOC_xxx|> location tokens.
package paddleocrvl

import (
	"github.com/schaepher/ocr/decoder"
	decoderPVL "github.com/schaepher/ocr/decoder/paddleocrvl"
	"github.com/schaepher/ocr/provider"
)

// Provider implements provider.Provider for PaddleOCR-VL.
type Provider struct{}

// New creates a new PaddleOCR-VL provider.
func New() *Provider {
	return &Provider{}
}

func (p *Provider) DefaultModel() string     { return "PaddleOCR-VL-1.6" }
func (p *Provider) SystemPrompt() string     { return SpottingSystemPrompt }
func (p *Provider) Decoder() decoder.Decoder { return decoderPVL.NewDecoder() }

var _ provider.Provider = (*Provider)(nil)
