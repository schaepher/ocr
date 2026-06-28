package paddleocrvl

import (
	"image"
	"regexp"
	"strconv"
	"strings"

	"github.com/schaepher/ocr/decoder"
	"github.com/schaepher/ocr/document"
)

const (
	// LocGridSize is the discrete coordinate grid size used by PaddleOCR-VL.
	// LOC tokens range from 0 to LocGridSize-1.
	LocGridSize = 1000

	// pointsPerBlock is the number of points in a quadrilateral bounding box.
	pointsPerBlock = 4
)

var locRE = regexp.MustCompile(`<\|LOC_(\d+)\|>`)

// parseRaw parses the raw model output into a list of Blocks.
// It extracts LOC token sequences and their associated text.
func parseRaw(raw string, imgSize image.Point) (*document.Document, error) {
	locs := locRE.FindAllStringSubmatchIndex(raw, -1)
	if len(locs) == 0 {
		// No LOC tokens — treat entire output as a single text block without location.
		return &document.Document{
			Width:  imgSize.X,
			Height: imgSize.Y,
			Blocks: []document.Block{
				{Text: strings.TrimSpace(raw)},
			},
		}, nil
	}

	tokens, err := extractLocValues(raw, locs)
	if err != nil {
		return nil, err
	}

	segments := splitText(raw, locs)
	blocks := buildBlocks(segments, tokens, imgSize)

	return &document.Document{
		Width:  imgSize.X,
		Height: imgSize.Y,
		Blocks: blocks,
	}, nil
}

// extractLocValues collects all LOC numeric values from the match indices.
func extractLocValues(raw string, locs [][]int) ([]int, error) {
	var vals []int
	for _, m := range locs {
		if len(m) < 4 {
			continue
		}
		v, err := strconv.Atoi(raw[m[2]:m[3]])
		if err != nil {
			return nil, decoder.ErrInvalidLocToken
		}
		if v < 0 || v >= LocGridSize {
			return nil, decoder.ErrInvalidLocToken
		}
		vals = append(vals, v)
	}
	return vals, nil
}

// splitText splits the raw string into text segments.
// Each segment corresponds to the text before each LOC sequence.
func splitText(raw string, locs [][]int) []string {
	var segments []string
	pos := 0
	for _, m := range locs {
		if pos < m[0] {
			segments = append(segments, raw[pos:m[0]])
		}
		pos = m[1]
	}
	if pos < len(raw) {
		segments = append(segments, raw[pos:])
	}
	return segments
}

// buildBlocks constructs Blocks from text segments and LOC token values.
// Tokens are paired (x, y) to form Points; 4 Points form one Block's Polygon.
func buildBlocks(segments []string, tokens []int, imgSize image.Point) []document.Block {
	if len(tokens)%2 != 0 {
		// Odd token count — drop the last token to avoid panic.
		tokens = tokens[:len(tokens)-1]
	}

	var blocks []document.Block
	tokenPos := 0

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		remaining := len(tokens) - tokenPos
		if remaining < pointsPerBlock*2 {
			// Not enough tokens for a full polygon — attach remaining as text-only block.
			blocks = append(blocks, document.Block{Text: seg})
			continue
		}

		pts := make([]image.Point, 0, pointsPerBlock)
		for i := 0; i < pointsPerBlock; i++ {
			if tokenPos+1 >= len(tokens) {
				break
			}
			x := scaleCoord(tokens[tokenPos], imgSize.X)
			y := scaleCoord(tokens[tokenPos+1], imgSize.Y)
			pts = append(pts, image.Point{X: x, Y: y})
			tokenPos += 2
		}

		if len(pts) == pointsPerBlock {
			blocks = append(blocks, document.Block{
				Text:    seg,
				Polygon: document.Polygon{Points: pts},
			})
		} else {
			blocks = append(blocks, document.Block{Text: seg})
		}
	}

	return blocks
}

// scaleCoord converts a discrete LOC coordinate (0–LocGridSize) to a pixel value.
func scaleCoord(loc, imgDim int) int {
	return loc * imgDim / LocGridSize
}
