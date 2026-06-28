package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schaepher/ocr"
	"github.com/schaepher/ocr/provider"
	"github.com/schaepher/ocr/provider/paddleocrvl"
	"github.com/schaepher/ocr/provider/qwen/qwen3vl"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".webp": true, ".tiff": true, ".tif": true,
}

func main() {
	imagePath := flag.String("image", "", "path to image file")
	imageDir := flag.String("image-dir", "", "path to directory of images (default: current dir if no --image)")
	baseURL := flag.String("base-url", "http://127.0.0.1:1234/v1", "LM Studio API base URL")
	provName := flag.String("provider", "paddleocrvl", "OCR provider: paddleocrvl, qwen3vl")
	model := flag.String("model", "", "model name (overrides provider default)")
	format := flag.String("format", "html", "output format: markdown, json, html, text")
	outputPath := flag.String("output", "", "output file path (--image only; default: same dir as image, auto extension)")
	parallel := flag.Int("parallel", 1, "max concurrent conversions (--image-dir only)")
	maxHeight := flag.Int("max-height", 0, "max image height before slicing (0=no slicing)")
	flag.Parse()

	var prov provider.Provider
	switch *provName {
	case "qwen3":
		prov = qwen3vl.New()
	default:
		prov = paddleocrvl.New()
	}
	if *model == "" {
		*model = prov.DefaultModel()
	}

	if *imagePath != "" && *imageDir != "" {
		fmt.Fprintln(os.Stderr, "Error: --image and --image-dir are mutually exclusive")
		os.Exit(1)
	}
	if *imagePath == "" && *imageDir == "" {
		*imageDir = "."
	}

	if *imagePath != "" {
		outPath := *outputPath
		if outPath == "" {
			outPath = deriveOutPath(*imagePath, *format)
		}
		if err := processImage(prov, *imagePath, outPath, *baseURL, *model, *format, *maxHeight); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Output written to %s\n", outPath)
		return
	}

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
	var done atomic.Int32
	total := len(images)

	for _, img := range images {
		wg.Add(1)
		sem <- struct{}{}
		go func(imgPath string) {
			defer wg.Done()
			defer func() { <-sem }()
			outPath := deriveOutPath(imgPath, *format)
			n := done.Add(1)
			fmt.Printf("Processing [%d/%d] %s\n", n, total, filepath.Base(imgPath))
			if err := processImage(prov, imgPath, outPath, *baseURL, *model, *format, *maxHeight); err != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			}
		}(img)
	}
	wg.Wait()
	fmt.Printf("Done: %d files processed.\n", total)
}

func processImage(prov provider.Provider, imagePath, outPath, baseURL, model, format string, maxHeight int) error {
	cli := ocr.New(prov).
		LMStudio(baseURL).
		Model(model).
		MaxHeight(maxHeight)
	if maxHeight > 0 {
		cli = cli.OnProgress(func(cur, total, y int) {
			fmt.Printf("  slice [%d/%d] y=%d\n", cur, total, y)
		})
	}
	doc, err := cli.ParseImage(context.Background(), imagePath)
	if err != nil {
		return fmt.Errorf("OCR: %w", err)
	}

	imageSrc := imagePath
	if format == "html" {
		imageSrc = resolveImageSrc(imagePath, outPath)
	}

	var out string
	switch format {
	case "json":
		out, err = ocr.JSON(doc)
	case "html":
		out, err = ocr.HTML(doc, imageSrc)
	case "text":
		out, err = ocr.Text(doc)
	default:
		out, err = ocr.Markdown(doc)
	}
	if err != nil {
		return fmt.Errorf("output: %w", err)
	}

	return os.WriteFile(outPath, []byte(out), 0644)
}

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

func deriveOutPath(imagePath, format string) string {
	ext := formatExt(format)
	base := strings.TrimSuffix(imagePath, filepath.Ext(imagePath))
	return base + ext
}

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
