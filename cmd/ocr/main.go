package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	paddleocrvl "github.com/schaepher/paddleocrvl"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".webp": true, ".tiff": true, ".tif": true,
}

func main() {
	imagePath := flag.String("image", "", "path to image file")
	imageDir := flag.String("image-dir", "", "path to directory of images")
	baseURL := flag.String("base-url", "http://127.0.0.1:1234/v1", "LM Studio API base URL")
	model := flag.String("model", "PaddleOCR-VL-1.6", "model name")
	format := flag.String("format", "markdown", "output format: markdown, json, html, text")
	outputPath := flag.String("output", "", "output file path (--image only; default: same dir as image, auto extension)")
	parallel := flag.Int("parallel", 1, "max concurrent conversions (--image-dir only)")
	flag.Parse()

	if *imagePath == "" && *imageDir == "" {
		fmt.Fprintln(os.Stderr, "Usage: ocr --image <path> [flags]")
		fmt.Fprintln(os.Stderr, "       ocr --image-dir <path> [--parallel N] [flags]")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *imagePath != "" && *imageDir != "" {
		fmt.Fprintln(os.Stderr, "Error: --image and --image-dir are mutually exclusive")
		os.Exit(1)
	}

	if *imagePath != "" {
		outPath := *outputPath
		if outPath == "" {
			outPath = deriveOutPath(*imagePath, *format)
		}
		if err := processImage(*imagePath, outPath, *baseURL, *model, *format); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Output written to %s\n", outPath)
		return
	}

	// --image-dir mode
	images, err := walkImages(*imageDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}
	if len(images) == 0 {
		fmt.Fprintln(os.Stderr, "No supported image files found in", *imageDir)
		os.Exit(1)
	}

	sem := make(chan struct{}, *parallel)
	var wg sync.WaitGroup

	for _, img := range images {
		wg.Add(1)
		sem <- struct{}{}
		go func(imgPath string) {
			defer wg.Done()
			defer func() { <-sem }()
			outPath := deriveOutPath(imgPath, *format)
			fmt.Printf("Processing %s ...\n", imgPath)
			if err := processImage(imgPath, outPath, *baseURL, *model, *format); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", imgPath, err)
				return
			}
			fmt.Printf("  -> %s\n", outPath)
		}(img)
	}
		wg.Wait()
}

// processImage runs OCR on a single image and writes the output file.
func processImage(imagePath, outPath, baseURL, model, format string) error {
	doc, err := paddleocrvl.New().
		LMStudio(baseURL).
		Model(model).
		ParseImage(context.Background(), imagePath)
	if err != nil {
		return fmt.Errorf("OCR: %w", err)
	}

	// Compute imageSrc for HTML output.
	imageSrc := imagePath
	if format == "html" {
		imageSrc = resolveImageSrc(imagePath, outPath)
	}

	var out string
	switch format {
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
		return fmt.Errorf("output: %w", err)
	}

	return os.WriteFile(outPath, []byte(out), 0644)
}

// walkImages recursively walks dir and returns paths to supported image files.
func walkImages(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if imageExts[strings.ToLower(filepath.Ext(path))] {
			images = append(images, path)
		}
		return nil
	})
	return images, err
}

// deriveOutPath returns the output path: same dir, same base name, format extension.
func deriveOutPath(imagePath, format string) string {
	ext := formatExt(format)
	base := strings.TrimSuffix(imagePath, filepath.Ext(imagePath))
	return base + ext
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
