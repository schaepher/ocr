// Command paddle2html converts PaddleOCR JSON output to an HTML overlay
// using the project's own document types and output.HTML renderer.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/schaepher/ocr/document"
	"github.com/schaepher/ocr/output"
)

type paddleBlock struct {
	BBox       [][]int `json:"bbox"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

func main() {
	jsonPath := flag.String("json", "ocr_output.json", "path to PaddleOCR JSON output")
	imagePath := flag.String("image", "", "path to original image (for HTML img src)")
	outPath := flag.String("output", "ocr_output_go.html", "output HTML path")
	flag.Parse()

	if *imagePath == "" {
		fmt.Fprintln(os.Stderr, "Error: --image is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", *jsonPath, err)
		os.Exit(1)
	}

	var blocks []paddleBlock
	if err := json.Unmarshal(data, &blocks); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", *jsonPath, err)
		os.Exit(1)
	}

	// Image dimensions - hardcoded from the known image size
	// TODO: read from image metadata
	imgW, imgH := 1080, 22572

	doc := &document.Document{
		Width:  imgW,
		Height: imgH,
	}

	for _, blk := range blocks {
		if blk.Text == "" || len(blk.BBox) != 4 {
			continue
		}

		var points []image.Point
		for _, pt := range blk.BBox {
			if len(pt) != 2 {
				continue
			}
			points = append(points, image.Point{X: pt[0], Y: pt[1]})
		}
		if len(points) != 4 {
			continue
		}

		doc.Blocks = append(doc.Blocks, document.Block{
			Text:       blk.Text,
			Polygon:    document.Polygon{Points: points},
			Confidence: blk.Confidence,
		})
	}

	// Resolve image src relative to output
	absImage, _ := filepath.Abs(*imagePath)
	absOut, _ := filepath.Abs(*outPath)
	imageSrc := absImage
	if filepath.Dir(absImage) == filepath.Dir(absOut) {
		imageSrc = filepath.Base(*imagePath)
	}

	html, err := output.HTML(doc, imageSrc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Written %s (%d blocks)\n", *outPath, len(doc.Blocks))
}
