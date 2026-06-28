package layout

import (
	"image"
	"sort"

	"github.com/schaepher/ocr/document"
)

// cellBounds stores a block's index and its bounding rectangle.
type cellBounds struct {
	index int
	rect  image.Rectangle
}

// DetectTables returns a Processor that identifies table-like structures
// where blocks form a grid pattern.
//
// This is a basic heuristic detector. It looks for blocks whose bounding
// boxes align on both X and Y axes, suggesting a table/cell layout.
func DetectTables() document.Processor {
	return func(doc *document.Document) (*document.Document, error) {
		if len(doc.Blocks) < 4 {
			return doc, nil
		}

		// Only consider blocks with a valid polygon.
		var cells []document.Block
		for _, b := range doc.Blocks {
			if !b.Polygon.IsZero() {
				cells = append(cells, b)
			}
		}

		if len(cells) < 4 {
			return doc, nil
		}

		bounds := make([]cellBounds, len(cells))
		for i, c := range cells {
			bounds[i] = cellBounds{index: i, rect: c.Polygon.Bounds()}
		}

		xPositions := collectUniqueXPositions(bounds)
		yPositions := collectUniqueYPositions(bounds)

		// If we find at least 2 columns and 2 rows with aligned positions,
		// it's likely a table.
		if len(xPositions) >= 2 && len(yPositions) >= 2 {
			potentialCells := countAlignedCells(bounds, xPositions, yPositions)
			if potentialCells >= len(cells)*3/4 {
				// Mark blocks as table cells by grouping them.
				// Full table reconstruction is v2.
				_ = potentialCells
			}
		}

		return doc, nil
	}
}

// collectUniqueXPositions finds distinct X-axis column positions.
func collectUniqueXPositions(bounds []cellBounds) []int {
	xSet := make(map[int]bool)
	for _, b := range bounds {
		xSet[b.rect.Min.X] = true
		xSet[b.rect.Max.X] = true
	}
	xs := make([]int, 0, len(xSet))
	for x := range xSet {
		xs = append(xs, x)
	}
	sort.Ints(xs)
	return xs
}

// collectUniqueYPositions finds distinct Y-axis row positions.
func collectUniqueYPositions(bounds []cellBounds) []int {
	ySet := make(map[int]bool)
	for _, b := range bounds {
		ySet[b.rect.Min.Y] = true
		ySet[b.rect.Max.Y] = true
	}
	ys := make([]int, 0, len(ySet))
	for y := range ySet {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	return ys
}

// countAlignedCells counts how many cells align with the grid.
func countAlignedCells(bounds []cellBounds, xPositions, yPositions []int) int {
	count := 0
	for _, b := range bounds {
		if isAligned(b.rect.Min.X, xPositions) && isAligned(b.rect.Min.Y, yPositions) &&
			isAligned(b.rect.Max.X, xPositions) && isAligned(b.rect.Max.Y, yPositions) {
			count++
		}
	}
	return count
}

// isAligned checks if a position matches any grid position within tolerance.
func isAligned(pos int, positions []int) bool {
	for _, p := range positions {
		if abs(pos-p) <= YTolerance {
			return true
		}
	}
	return false
}
