package output

import (
	"strings"

	"github.com/schaepher/ocr/document"
)

// Text renders a Document as plain text with newline separators.
func Text(doc *document.Document) (string, error) {
	var b strings.Builder

	for i, blk := range doc.Blocks {
		if blk.Text == "" {
			continue
		}
		b.WriteString(blk.Text)
		if i < len(doc.Blocks)-1 {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}
