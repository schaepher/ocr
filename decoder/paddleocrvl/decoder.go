package paddleocrvl

import (
	"image"

	"github.com/schaepher/paddleocrvl/decoder"
	"github.com/schaepher/paddleocrvl/document"
)

// Decoder implements decoder.Decoder for PaddleOCR-VL models.
type Decoder struct{}

// NewDecoder creates a new PaddleOCR-VL decoder.
func NewDecoder() *Decoder {
	return &Decoder{}
}

// Decode parses PaddleOCR-VL raw output containing <|LOC_xxx|> tokens
// into a structured Document with pixel-accurate polygon coordinates.
func (d *Decoder) Decode(raw string, imageSize image.Point) (*document.Document, error) {
	return parseRaw(raw, imageSize)
}

// Ensure Decoder satisfies the decoder.Decoder interface.
var _ decoder.Decoder = (*Decoder)(nil)
