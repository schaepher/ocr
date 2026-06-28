package output

import (
	"strings"

	"github.com/schaepher/ocr/document"
)

// Markdown renders a Document as plain Markdown text.
// Blocks are separated by blank lines, preserving reading order.
func Markdown(doc *document.Document) (string, error) {
	var b strings.Builder

	for i, blk := range doc.Blocks {
		if blk.Text == "" {
			continue
		}
		b.WriteString(blk.Text)
		if i < len(doc.Blocks)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String(), nil
}
