package layout

import (
	"testing"

	"github.com/schaepher/ocr/document"
)

func TestMergeParagraph(t *testing.T) {
	tests := []struct {
		name   string
		blocks []document.Block
		want   []string
	}{
		{
			name: "merge adjacent blocks on same line",
			blocks: []document.Block{
				{Text: "Hello", Polygon: rectPoly(0, 0, 50, 20)},
				{Text: "world", Polygon: rectPoly(60, 0, 110, 20)},
			},
			want: []string{"Hello world"},
		},
		{
			name: "do not merge different lines",
			blocks: []document.Block{
				{Text: "line1", Polygon: rectPoly(0, 0, 50, 20)},
				{Text: "line2", Polygon: rectPoly(0, 100, 50, 120)},
			},
			want: []string{"line1", "line2"},
		},
		{
			name:   "empty blocks",
			blocks: nil,
			want:   nil,
		},
		{
			name: "single block unchanged",
			blocks: []document.Block{
				{Text: "alone", Polygon: rectPoly(0, 0, 50, 20)},
			},
			want: []string{"alone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &document.Document{Width: 500, Height: 500, Blocks: tt.blocks}
			mergeProc := MergeParagraph()

			result, err := mergeProc(doc)
			if err != nil {
				t.Fatalf("MergeParagraph() error: %v", err)
			}

			if len(result.Blocks) != len(tt.want) {
				t.Fatalf("len(Blocks) = %d, want %d", len(result.Blocks), len(tt.want))
			}

			for i, w := range tt.want {
				if result.Blocks[i].Text != w {
					t.Errorf("Blocks[%d].Text = %q, want %q", i, result.Blocks[i].Text, w)
				}
			}
		})
	}
}
