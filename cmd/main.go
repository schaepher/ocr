package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	paddleocrvl "github.com/schaepher/paddleocrvl"
)

func main() {
	imagePath := flag.String("image", "", "path to image file (required)")
	baseURL := flag.String("base-url", "http://127.0.0.1:1234/v1", "LM Studio API base URL")
	model := flag.String("model", "PaddleOCR-VL-1.6", "model name")
	format := flag.String("format", "markdown", "output format: markdown, json, html, text")
	outputPath := flag.String("output", "", "output file path (default: same directory as image, auto extension)")
	flag.Parse()

	if *imagePath == "" {
		fmt.Fprintln(os.Stderr, "Usage: paddleocrvl --image <path> [--format markdown|json|html|text] [--output <path>]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Derive output path if not specified.
	outPath := *outputPath
	if outPath == "" {
		ext := formatExt(*format)
		base := strings.TrimSuffix(*imagePath, filepath.Ext(*imagePath))
		outPath = base + ext
	}

	ctx := context.Background()

	doc, err := paddleocrvl.New().
		LMStudio(*baseURL).
		Model(*model).
		ParseImage(ctx, *imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Compute the image src for HTML output.
	// Use a relative path when output and source are in the same directory,
	// otherwise use the absolute path.
	imageSrc := *imagePath
	if *format == "html" {
		imageSrc = resolveImageSrc(*imagePath, outPath)
	}

	var out string
	switch *format {
	case "json":
		out, err = paddleocrvl.JSON(doc)
	case "html":
		out, err = paddleocrvl.HTML(doc, imageSrc)
	case "text":
		out, err = paddleocrvl.Text(doc)
	default:
		out, err = paddleocrvl.Markdown(doc)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Output error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, []byte(out), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Output written to %s\n", outPath)
}

// formatExt returns the file extension for a given output format.
func formatExt(format string) string {
	switch format {
	case "json":
		return ".json"
	case "html":
		return ".html"
	case "text":
		return ".txt"
	default:
		return ".md"
	}
}

// resolveImageSrc returns the path to use in the HTML img src attribute.
// If the output file and the source image are in the same directory,
// it returns just the base name (relative); otherwise it returns the
// absolute path.
func resolveImageSrc(imagePath, outPath string) string {
	absImage, err := filepath.Abs(imagePath)
	if err != nil {
		return imagePath
	}
	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return absImage
	}
	if filepath.Dir(absImage) == filepath.Dir(absOut) {
		return filepath.Base(imagePath)
	}
	return absImage
}
