package qwen3vl

import (
	"github.com/schaepher/ocr/decoder"
	decoderPVL "github.com/schaepher/ocr/decoder/paddleocrvl"
	"github.com/schaepher/ocr/provider"
)

// Provider implements provider.Provider for Qwen3-VL models.
// Reuses PaddleOCR-VL decoder since both output <|LOC_xxx|> tokens.
type Provider struct{}

func New() *Provider { return &Provider{} }

func (p *Provider) DefaultModel() string     { return "qwen/qwen3-vl-8b" }
func (p *Provider) SystemPrompt() string     { return SpottingSystemPrompt }
func (p *Provider) Decoder() decoder.Decoder { return decoderPVL.NewDecoder() }

var _ provider.Provider = (*Provider)(nil)
