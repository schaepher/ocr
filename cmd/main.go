package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	paddleocrvl "github.com/schaepher/paddleocrvl"
)

func main() {
	imagePath := flag.String("image", "", "path to image file (required)")
	baseURL := flag.String("base-url", "http://127.0.0.1:1234/v1", "LM Studio API base URL")
	model := flag.String("model", "PaddleOCR-VL-1.6", "model name")
	format := flag.String("format", "markdown", "output format: markdown, json, html, text")
	flag.Parse()

	if *imagePath == "" {
		fmt.Fprintln(os.Stderr, "Usage: paddleocrvl --image <path> [--format markdown|json|html|text]")
		flag.PrintDefaults()
		os.Exit(1)
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

	var out string
	switch *format {
	case "json":
		out, err = paddleocrvl.JSON(doc)
	case "html":
		out, err = paddleocrvl.HTML(doc)
	case "text":
		out, err = paddleocrvl.Text(doc)
	default:
		out, err = paddleocrvl.Markdown(doc)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Output error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(out)
}
