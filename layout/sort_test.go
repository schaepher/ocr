package layout

import (
	"image"
	"testing"

	"github.com/schaepher/paddleocrvl/document"
)

func TestSort(t *testing.T) {
	tests := []struct {
		name  string
		blocks []document.Block
		want  []string // expected text order
	}{
		{
			name: "top-to-bottom reading order",
			blocks: []document.Block{
				{Text: "bottom", Polygon: rectPoly(0, 200, 100, 250)},
				{Text: "top", Polygon: rectPoly(0, 0, 100, 50)},
				{Text: "middle", Polygon: rectPoly(0, 100, 100, 150)},
			},
			want: []string{"top", "middle", "bottom"},
		},
		{
			name: "left-to-right within same line",
			blocks: []document.Block{
				{Text: "right", Polygon: rectPoly(200, 0, 300, 50)},
				{Text: "left", Polygon: rectPoly(0, 0, 100, 50)},
			},
			want: []string{"left", "right"},
		},
		{
			name: "mixed: Y clusters then X order",
			blocks: []document.Block{
				{Text: "C2", Polygon: rectPoly(200, 100, 300, 150)},
				{Text: "A1", Polygon: rectPoly(0, 0, 100, 50)},
				{Text: "A2", Polygon: rectPoly(200, 0, 300, 50)},
				{Text: "B1", Polygon: rectPoly(0, 50, 100, 100)},
				{Text: "C1", Polygon: rectPoly(0, 100, 100, 150)},
				{Text: "B2", Polygon: rectPoly(200, 50, 300, 100)},
			},
			want: []string{"A1", "A2", "B1", "B2", "C1", "C2"},
		},
		{
			name: "floating blocks (no polygon) placed last",
			blocks: []document.Block{
				{Text: "bottom", Polygon: rectPoly(0, 100, 100, 150)},
				{Text: "floating"},
				{Text: "top", Polygon: rectPoly(0, 0, 100, 50)},
			},
			want: []string{"top", "bottom", "floating"},
		},
		{
			name:   "empty block list",
			blocks: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &document.Document{Width: 500, Height: 500, Blocks: tt.blocks}
			sortProc := Sort()

			result, err := sortProc(doc)
			if err != nil {
				t.Fatalf("Sort() error: %v", err)
			}

			if len(result.Blocks) != len(tt.want) {
				t.Fatalf("len(Blocks) = %d, want %d\ngot:  %v\nwant: %v",
					len(result.Blocks), len(tt.want), textList(result.Blocks), tt.want)
			}

			for i, w := range tt.want {
				if result.Blocks[i].Text != w {
					t.Errorf("Blocks[%d].Text = %q, want %q\nfull order: %v",
						i, result.Blocks[i].Text, w, textList(result.Blocks))
				}
			}
		})
	}
}

func TestSortPreservesOtherFields(t *testing.T) {
	blocks := []document.Block{
		{Text: "b", Confidence: 0.9, Polygon: rectPoly(100, 0, 200, 50)},
		{Text: "a", Confidence: 0.8, Polygon: rectPoly(0, 0, 100, 50)},
	}

	doc := &document.Document{Width: 500, Height: 500, Blocks: blocks}
	result, _ := Sort()(doc)

	if result.Blocks[0].Confidence != 0.8 {
		t.Errorf("Confidence not preserved: got %v", result.Blocks[0].Confidence)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct{ in, want int }{
		{5, 5}, {-5, 5}, {0, 0},
	}
	for _, tt := range tests {
		if got := abs(tt.in); got != tt.want {
			t.Errorf("abs(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// Helpers

func rectPoly(x1, y1, x2, y2 int) document.Polygon {
	return document.Polygon{
		Points: []image.Point{
			{X: x1, Y: y1},
			{X: x2, Y: y1},
			{X: x2, Y: y2},
			{X: x1, Y: y2},
		},
	}
}

func textList(blocks []document.Block) []string {
	out := make([]string, len(blocks))
	for i, b := range blocks {
		out[i] = b.Text
	}
	return out
}
