package layout

import (
	"image"
	"strings"

	"github.com/schaepher/paddleocrvl/document"
)

// MergeParagraph returns a Processor that merges adjacent blocks
// that are on the same text line (same Y cluster) and close horizontally
// into a single paragraph block.
func MergeParagraph() document.Processor {
	return func(doc *document.Document) (*document.Document, error) {
		if len(doc.Blocks) < 2 {
			return doc, nil
		}

		merged := make([]document.Block, 0, len(doc.Blocks))
		current := doc.Blocks[0]

		for i := 1; i < len(doc.Blocks); i++ {
			next := doc.Blocks[i]

			if canMerge(current, next) {
				// Merge text with a space.
				current.Text = strings.TrimSpace(current.Text) + " " + strings.TrimSpace(next.Text)

				// Expand polygon to encompass both blocks.
				if !current.Polygon.IsZero() && !next.Polygon.IsZero() {
					r := current.Polygon.Bounds().Union(next.Polygon.Bounds())
					current.Polygon = document.Polygon{
						Points: []image.Point{
							{X: r.Min.X, Y: r.Min.Y},
							{X: r.Max.X, Y: r.Min.Y},
							{X: r.Max.X, Y: r.Max.Y},
							{X: r.Min.X, Y: r.Max.Y},
						},
					}
				}
			} else {
				merged = append(merged, current)
				current = next
			}
		}
		merged = append(merged, current)

		doc.Blocks = merged
		return doc, nil
	}
}

// canMerge checks if two blocks are on the same text line and close enough
// horizontally to be merged into a single paragraph.
func canMerge(a, b document.Block) bool {
	if a.Polygon.IsZero() || b.Polygon.IsZero() {
		return false
	}

	aBounds := a.Polygon.Bounds()
	bBounds := b.Polygon.Bounds()

	// Same line? Vertical center must be within YTolerance.
	aCenterY := (aBounds.Min.Y + aBounds.Max.Y) / 2
	bCenterY := (bBounds.Min.Y + bBounds.Max.Y) / 2

	if abs(aCenterY-bCenterY) > YTolerance {
		return false
	}

	// Close horizontally? Gap between blocks should be small.
	// a is to the left of b (assumes blocks are sorted).
	if aBounds.Max.X > bBounds.Min.X {
		// Overlapping or b is left of a — only merge if significant overlap.
		return false
	}

	gap := bBounds.Min.X - aBounds.Max.X
	avgHeight := (aBounds.Dy() + bBounds.Dy()) / 2
	if avgHeight == 0 {
		return gap < 50 // fallback threshold
	}

	// Gap should be less than the height of a typical character (≈ average height).
	return gap < avgHeight*2
}
