package output

import (
	"fmt"
	"html"
	"strings"

	"github.com/schaepher/paddleocrvl/document"
)

// HTML renders a Document as HTML with positioned div elements.
// Each block becomes a <div> with absolute positioning matching its
// bounding box, preserving the original layout.
func HTML(doc *document.Document) (string, error) {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<style>\n")
	b.WriteString(".ocr-page { position: relative; ")
	fmt.Fprintf(&b, "width: %dpx; height: %dpx; ", doc.Width, doc.Height)
	b.WriteString("overflow: hidden; }\n")
	b.WriteString(".ocr-block { position: absolute; overflow: hidden; ")
	b.WriteString("white-space: pre-wrap; font-size: 14px; ")
	b.WriteString("font-family: sans-serif; }\n")
	b.WriteString("</style>\n</head>\n<body>\n")
	fmt.Fprintf(&b, "<div class=\"ocr-page\" style=\"width:%dpx;height:%dpx;\">\n", doc.Width, doc.Height)

	for _, blk := range doc.Blocks {
		if blk.Text == "" {
			continue
		}

		if blk.Polygon.IsZero() {
			fmt.Fprintf(&b, "<div class=\"ocr-block\">%s</div>\n", html.EscapeString(blk.Text))
			continue
		}

		bounds := blk.Polygon.Bounds()
		fmt.Fprintf(&b, "<div class=\"ocr-block\" style=\"left:%dpx;top:%dpx;width:%dpx;height:%dpx;\">%s</div>\n",
			bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy(),
			html.EscapeString(blk.Text))
	}

	b.WriteString("</div>\n</body>\n</html>\n")

	return b.String(), nil
}
