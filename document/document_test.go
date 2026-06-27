package document

import (
	"image"
	"testing"
)

func TestPolygonBounds(t *testing.T) {
	tests := []struct {
		name string
		pts  []image.Point
		want image.Rectangle
	}{
		{
			name: "empty polygon",
			pts:  nil,
			want: image.Rectangle{},
		},
		{
			name: "axis-aligned rectangle",
			pts:  []image.Point{{10, 20}, {100, 20}, {100, 80}, {10, 80}},
			want: image.Rect(10, 20, 100, 80),
		},
		{
			name: "single point",
			pts:  []image.Point{{5, 5}},
			want: image.Rect(5, 5, 5, 5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Polygon{Points: tt.pts}
			got := p.Bounds()
			if got != tt.want {
				t.Errorf("Bounds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolygonCenter(t *testing.T) {
	tests := []struct {
		name string
		pts  []image.Point
		want image.Point
	}{
		{
			name: "empty polygon",
			pts:  nil,
			want: image.Point{},
		},
		{
			name: "axis-aligned rectangle",
			pts:  []image.Point{{10, 20}, {100, 20}, {100, 80}, {10, 80}},
			want: image.Point{X: 55, Y: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Polygon{Points: tt.pts}
			got := p.Center()
			if got != tt.want {
				t.Errorf("Center() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolygonIsZero(t *testing.T) {
	var empty Polygon
	if !empty.IsZero() {
		t.Error("empty polygon should be zero")
	}
	nonEmpty := Polygon{Points: []image.Point{{1, 2}}}
	if nonEmpty.IsZero() {
		t.Error("non-empty polygon should not be zero")
	}
}

func TestDocumentString(t *testing.T) {
	doc := &Document{
		Width:  1920,
		Height: 1080,
		Blocks: []Block{
			{Text: "hello", Polygon: Polygon{Points: []image.Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}}},
		},
	}
	s := doc.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}
