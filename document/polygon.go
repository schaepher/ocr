package document

import (
	"image"
)

// Polygon represents a quadrilateral (4 points) bounding region.
// Points are ordered clockwise: top-left, top-right, bottom-right, bottom-left.
type Polygon struct {
	Points []image.Point
}

// Bounds returns the axis-aligned bounding rectangle of the polygon.
func (p Polygon) Bounds() image.Rectangle {
	if len(p.Points) == 0 {
		return image.Rectangle{}
	}

	minX, minY := p.Points[0].X, p.Points[0].Y
	maxX, maxY := minX, minY

	for _, pt := range p.Points[1:] {
		if pt.X < minX {
			minX = pt.X
		}
		if pt.X > maxX {
			maxX = pt.X
		}
		if pt.Y < minY {
			minY = pt.Y
		}
		if pt.Y > maxY {
			maxY = pt.Y
		}
	}

	return image.Rect(minX, minY, maxX, maxY)
}

// Center returns the centroid of the polygon (average of all points).
func (p Polygon) Center() image.Point {
	if len(p.Points) == 0 {
		return image.Point{}
	}

	var sumX, sumY int
	for _, pt := range p.Points {
		sumX += pt.X
		sumY += pt.Y
	}

	n := len(p.Points)
	return image.Point{X: sumX / n, Y: sumY / n}
}

// IsZero reports whether the polygon has no points.
func (p Polygon) IsZero() bool {
	return len(p.Points) == 0
}
