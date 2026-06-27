package paddleocrvl

import (
	"image"
	"testing"

	"github.com/schaepher/paddleocrvl/decoder"
	"github.com/schaepher/paddleocrvl/document"
)

func TestParseRaw(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		imgSize  image.Point
		want     *document.Document
		wantErr  bool
		errIs    error
	}{
		{
			name:    "single block with LOC tokens",
			raw:     "微博正文<|LOC_412|><|LOC_53|><|LOC_584|><|LOC_53|><|LOC_584|><|LOC_75|><|LOC_412|><|LOC_75|>",
			imgSize: image.Point{X: 1920, Y: 1080},
			want: &document.Document{
				Width:  1920,
				Height: 1080,
				Blocks: []document.Block{
					{
						Text: "微博正文",
						Polygon: document.Polygon{
							Points: []image.Point{
								{X: 791, Y: 57},
								{X: 1121, Y: 57},
								{X: 1121, Y: 81},
								{X: 791, Y: 81},
							},
						},
					},
				},
			},
		},
		{
			name:    "no LOC tokens",
			raw:     "hello world",
			imgSize: image.Point{X: 1920, Y: 1080},
			want: &document.Document{
				Width:  1920,
				Height: 1080,
				Blocks: []document.Block{
					{Text: "hello world"},
				},
			},
		},
		{
			name:    "empty string",
			raw:     "",
			imgSize: image.Point{X: 1920, Y: 1080},
			want: &document.Document{
				Width:  1920,
				Height: 1080,
				Blocks: []document.Block{
					{Text: ""},
				},
			},
		},
		{
			name:    "multiple blocks",
			raw:     "公开<|LOC_100|><|LOC_200|><|LOC_300|><|LOC_200|><|LOC_300|><|LOC_400|><|LOC_100|><|LOC_400|>最安神<|LOC_500|><|LOC_600|><|LOC_700|><|LOC_600|><|LOC_700|><|LOC_800|><|LOC_500|><|LOC_800|>",
			imgSize: image.Point{X: 1000, Y: 1000},
			want: &document.Document{
				Width:  1000,
				Height: 1000,
				Blocks: []document.Block{
					{
						Text: "公开",
						Polygon: document.Polygon{
							Points: []image.Point{
								{X: 100, Y: 200},
								{X: 300, Y: 200},
								{X: 300, Y: 400},
								{X: 100, Y: 400},
							},
						},
					},
					{
						Text: "最安神",
						Polygon: document.Polygon{
							Points: []image.Point{
								{X: 500, Y: 600},
								{X: 700, Y: 600},
								{X: 700, Y: 800},
								{X: 500, Y: 800},
							},
						},
					},
				},
			},
		},
		{
			name:    "odd number of LOC tokens",
			raw:     "text<|LOC_100|><|LOC_200|><|LOC_300|>",
			imgSize: image.Point{X: 1000, Y: 1000},
			want: &document.Document{
				Width:  1000,
				Height: 1000,
				Blocks: []document.Block{
					{Text: "text"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRaw(tt.raw, tt.imgSize)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errIs != nil && err != tt.errIs {
					t.Errorf("expected error %v, got %v", tt.errIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Width != tt.want.Width {
				t.Errorf("Width = %d, want %d", got.Width, tt.want.Width)
			}
			if got.Height != tt.want.Height {
				t.Errorf("Height = %d, want %d", got.Height, tt.want.Height)
			}

			if len(got.Blocks) != len(tt.want.Blocks) {
				t.Fatalf("len(Blocks) = %d, want %d\ngot:  %+v\nwant: %+v",
					len(got.Blocks), len(tt.want.Blocks), got.Blocks, tt.want.Blocks)
			}

			for i := range tt.want.Blocks {
				if got.Blocks[i].Text != tt.want.Blocks[i].Text {
					t.Errorf("Blocks[%d].Text = %q, want %q", i, got.Blocks[i].Text, tt.want.Blocks[i].Text)
				}
				if !polygonsEqual(got.Blocks[i].Polygon, tt.want.Blocks[i].Polygon) {
					t.Errorf("Blocks[%d].Polygon = %v, want %v", i, got.Blocks[i].Polygon, tt.want.Blocks[i].Polygon)
				}
			}
		})
	}
}

func TestScaleCoord(t *testing.T) {
	tests := []struct {
		loc    int
		dim    int
		want   int
	}{
		{0, 1920, 0},
		{500, 1920, 960},
		{1000, 1920, 1920},
		{412, 1920, 791},
		{53, 1080, 57},
	}

	for _, tt := range tests {
		got := scaleCoord(tt.loc, tt.dim)
		if got != tt.want {
			t.Errorf("scaleCoord(%d, %d) = %d, want %d", tt.loc, tt.dim, got, tt.want)
		}
	}
}

func TestLocValuesOutOfRange(t *testing.T) {
	// LOC value >= LocGridSize should trigger ErrInvalidLocToken
	_, err := parseRaw("text<|LOC_1000|><|LOC_200|><|LOC_300|><|LOC_200|><|LOC_300|><|LOC_400|><|LOC_100|><|LOC_400|>",
		image.Point{X: 1920, Y: 1080})
	if err != decoder.ErrInvalidLocToken {
		t.Errorf("expected ErrInvalidLocToken, got %v", err)
	}
}

func TestDecoderInterface(t *testing.T) {
	d := NewDecoder()
	if d == nil {
		t.Fatal("NewDecoder() returned nil")
	}

	doc, err := d.Decode("hello world", image.Point{X: 100, Y: 100})
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if len(doc.Blocks) != 1 || doc.Blocks[0].Text != "hello world" {
		t.Errorf("unexpected result: %+v", doc)
	}
}

func polygonsEqual(a, b document.Polygon) bool {
	if len(a.Points) != len(b.Points) {
		return false
	}
	for i := range a.Points {
		if a.Points[i] != b.Points[i] {
			return false
		}
	}
	return true
}
