package output

import (
	"fmt"
	"html"
	"strings"

	"github.com/schaepher/paddleocrvl/document"
)

// HTML renders a Document as an HTML overlay.
// Each OCR block is rendered as an SVG <text> element positioned with
// percentage coordinates. SVG's textLength + lengthAdjust="spacingAndGlyphs"
// auto-stretches the text to fill the block — no font-size heuristics needed.
func HTML(doc *document.Document, imageSrc string) (string, error) {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html lang=\"zh\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n")
	b.WriteString("<style>\n")
	b.WriteString("body { margin: 0; background: #222; display: flex; justify-content: center; }\n")
	b.WriteString(".ocr-page { position: relative; overflow: hidden; background: #fff; ")
	b.WriteString("box-shadow: 0 2px 12px rgba(0,0,0,0.4); }\n")
	b.WriteString(".ocr-page img { display: block; width: 100%; height: auto; }\n")
	b.WriteString(".ocr-block { position: absolute; overflow: hidden; ")
	b.WriteString("transition: background 0.15s, box-shadow 0.15s; }\n")
	b.WriteString(".ocr-block text { fill: transparent; transition: fill 0.15s; ")
	b.WriteString("font-family: sans-serif; }\n")
	b.WriteString(".ocr-block:hover { background: rgba(255,255,255,0.85); ")
	b.WriteString("box-shadow: 0 1px 6px rgba(0,0,0,0.25); z-index: 1; }\n")
	b.WriteString(".ocr-block:hover text { fill: #000; }\n")
	b.WriteString("</style>\n</head>\n<body>\n")

	w, h := float64(doc.Width), float64(doc.Height)

	fmt.Fprintf(&b, "<div class=\"ocr-page\" style=\"width:%dpx;max-width:100%%;\">\n", doc.Width)
	fmt.Fprintf(&b, "<img src=\"%s\" alt=\"Background image\">\n",
		html.EscapeString(imageSrc))

	for _, blk := range doc.Blocks {
		if blk.Text == "" {
			continue
		}

		if blk.Polygon.IsZero() {
			fmt.Fprintf(&b, "<div class=\"ocr-block\" style=\"position:static;padding:4px 8px;\">")
			fmt.Fprintf(&b, "<svg width=\"100%%\" height=\"1.2em\" viewBox=\"0 0 1 1\" preserveAspectRatio=\"none\">")
			fmt.Fprintf(&b, "<text x=\"0\" y=\"0.9\" textLength=\"1\" lengthAdjust=\"spacingAndGlyphs\" fill=\"#000\">%s</text>",
				html.EscapeString(blk.Text))
			fmt.Fprintf(&b, "</svg></div>\n")
			continue
		}

		bounds := blk.Polygon.Bounds()
		bw, bh := bounds.Dx(), bounds.Dy()
		if bw < 1 {
			bw = 1
		}
		if bh < 1 {
			bh = 1
		}
		// Slightly smaller than the viewBox so text doesn't overflow.
		bhPad := bh - 2
		if bhPad < 1 {
			bhPad = 1
		}
		fh := bh - 1
		if fh < 1 {
			fh = 1
		}
		fw := bw - 1
		if fw < 1 {
			fw = 1
		}

		leftPct := float64(bounds.Min.X) / w * 100
		topPct := float64(bounds.Min.Y) / h * 100
		widthPct := float64(bw) / w * 100
		heightPct := float64(bh) / h * 100

		vertical := bh > bw

		fmt.Fprintf(&b,
			"<svg class=\"ocr-block\" overflow=\"hidden\" style=\"left:%.4f%%;top:%.4f%%;width:%.4f%%;height:%.4f%%;\" ",
			leftPct, topPct, widthPct, heightPct)
		fmt.Fprintf(&b,
			"viewBox=\"0 0 %d %d\" preserveAspectRatio=\"none\">", bw, bh)

		if vertical {
			fmt.Fprintf(&b,
				"<text x=\"%d\" y=\"%d\" writing-mode=\"vertical-rl\" textLength=\"%d\" lengthAdjust=\"spacingAndGlyphs\" font-size=\"%d\" text-anchor=\"middle\">%s</text>",
				bw/2, bh/2, bh, bw, html.EscapeString(blk.Text))
		} else {
			fmt.Fprintf(&b,
				"<text x=\"0\" y=\"%d\" textLength=\"%d\" lengthAdjust=\"spacingAndGlyphs\" font-size=\"%d\">%s</text>",
				bhPad, bw, fh, html.EscapeString(blk.Text))
		}

		b.WriteString("</svg>\n")
	}

	b.WriteString("</div>\n</body>\n</html>\n")

	return b.String(), nil
}
