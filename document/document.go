package document

import (
	"fmt"
	"strings"
)

// Document is the unified OCR result structure.
type Document struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Blocks []Block `json:"blocks"`
}

// Processor transforms a Document. Used for layout post-processing in a pipeline.
type Processor func(doc *Document) (*Document, error)

// String returns a human-readable summary of the document.
func (d *Document) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Document %dx%d, %d blocks\n", d.Width, d.Height, len(d.Blocks))
	for i, blk := range d.Blocks {
		fmt.Fprintf(&b, "  [%d] %q at %v\n", i, blk.Text, blk.Polygon.Center())
	}
	return b.String()
}
