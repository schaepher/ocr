package layout

import (
	"sort"

	"github.com/schaepher/paddleocrvl/document"
)

// YTolerance is the maximum vertical distance (in pixels) for two blocks
// to be considered on the same text line / cluster.
const YTolerance = 10

// blockWithCenter pairs a Block with its vertical center for Y-clustering.
type blockWithCenter struct {
	block   document.Block
	centerY int
}

// Sort returns a Processor that sorts blocks into natural reading order.
// The algorithm:
//  1. Compute the center Y for each block.
//  2. Group blocks whose center Ys are within YTolerance of each other.
//  3. Sort clusters by their average Y (top to bottom).
//  4. Within each cluster, sort blocks by X (left to right).
func Sort() document.Processor {
	return func(doc *document.Document) (*document.Document, error) {
		if len(doc.Blocks) == 0 {
			return doc, nil
		}

		// Filter out blocks with no polygon for sorting — put them at the end.
		var positioned []document.Block
		var floating []document.Block
		for _, b := range doc.Blocks {
			if b.Polygon.IsZero() {
				floating = append(floating, b)
			} else {
				positioned = append(positioned, b)
			}
		}

		if len(positioned) == 0 {
			return doc, nil
		}

		items := make([]blockWithCenter, len(positioned))
		for i, b := range positioned {
			items[i] = blockWithCenter{
				block:   b,
				centerY: b.Polygon.Center().Y,
			}
		}

		// Sort by center Y first.
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].centerY < items[j].centerY
		})

		// Cluster by Y.
		var clusters [][]blockWithCenter
		for _, item := range items {
			placed := false
			for ci, cluster := range clusters {
				for _, existing := range cluster {
					if abs(item.centerY-existing.centerY) <= YTolerance {
						clusters[ci] = append(cluster, item)
						placed = true
						break
					}
				}
				if placed {
					break
				}
			}
			if !placed {
				clusters = append(clusters, []blockWithCenter{item})
			}
		}

		// Sort clusters by their average Y.
		sort.SliceStable(clusters, func(i, j int) bool {
			return avgClusterY(clusters[i]) < avgClusterY(clusters[j])
		})

		// Within each cluster, sort by X (left to right).
		sorted := make([]document.Block, 0, len(doc.Blocks))
		for _, cluster := range clusters {
			sort.SliceStable(cluster, func(i, j int) bool {
				return cluster[i].block.Polygon.Bounds().Min.X < cluster[j].block.Polygon.Bounds().Min.X
			})
			for _, item := range cluster {
				sorted = append(sorted, item.block)
			}
		}

		sorted = append(sorted, floating...)
		doc.Blocks = sorted

		return doc, nil
	}
}

// avgClusterY returns the average Y coordinate of a cluster of blocks.
func avgClusterY(items []blockWithCenter) int {
	if len(items) == 0 {
		return 0
	}
	sum := 0
	for _, item := range items {
		sum += item.centerY
	}
	return sum / len(items)
}

// abs returns the absolute value of n.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
