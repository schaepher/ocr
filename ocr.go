// Package ocr is a Go SDK for running OCR via VLM models (PaddleOCR-VL,
// Qwen2.5-VL, etc.) through LM Studio or any OpenAI-compatible API.
//
// Usage:
//
//	doc, err := ocr.New(paddleocrvl.New()).
//	    LMStudio("http://127.0.0.1:1234/v1").
//	    ParseImage(ctx, "demo.png")
//
//	md, _ := ocr.Markdown(doc)
package ocr

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/schaepher/ocr/client"
	"github.com/schaepher/ocr/document"
	"github.com/schaepher/ocr/imageutil"
	"github.com/schaepher/ocr/layout"
	"github.com/schaepher/ocr/output"
	"github.com/schaepher/ocr/pipeline"
	"github.com/schaepher/ocr/provider"
)

// Client is the convenient top-level API for OCR.
// ProgressFunc is called with the current slice index and total during slicing.
type ProgressFunc func(current, total int, y int)

type Client struct {
	provider     provider.Provider
	baseURL      string
	model        string
	systemPrompt string
	maxHeight    int // 0 means no slicing
	overlap      int
	onProgress   ProgressFunc
}

// New creates a new Client with the given provider.
// The provider supplies the default model name, system prompt, and decoder.
func New(p provider.Provider) *Client {
	return &Client{
		provider:     p,
		baseURL:      "http://127.0.0.1:1234/v1",
		model:        p.DefaultModel(),
		systemPrompt: p.SystemPrompt(),
		overlap:      200,
	}
}

// LMStudio sets the LM Studio API base URL.
func (c *Client) LMStudio(url string) *Client {
	c.baseURL = url
	return c
}

// Model overrides the default model name.
func (c *Client) Model(name string) *Client {
	c.model = name
	return c
}

// SystemPrompt overrides the default system prompt.
func (c *Client) SystemPrompt(prompt string) *Client {
	c.systemPrompt = prompt
	return c
}

// MaxHeight sets the maximum image height before slicing.
// 0 (default) means no slicing. Overlap between slices is 200px by default.
func (c *Client) MaxHeight(h int) *Client {
	c.maxHeight = h
	return c
}

// Overlap sets the vertical overlap between adjacent slices.
func (c *Client) Overlap(px int) *Client {
	c.overlap = px
	return c
}

// OnProgress sets a callback invoked for each slice during slicing.
func (c *Client) OnProgress(fn ProgressFunc) *Client {
	c.onProgress = fn
	return c
}

// ParseImage runs OCR on an image file and returns a structured Document.
// If MaxHeight is set and the image exceeds it, the image is split into
// overlapping horizontal slices, each processed separately, then merged.
func (c *Client) ParseImage(ctx context.Context, imagePath string) (*document.Document, error) {
	if c.maxHeight <= 0 {
		return c.parseOne(ctx, imagePath)
	}

	slices, imgW, imgH, err := imageutil.SliceImage(imagePath, c.maxHeight, c.overlap)
	if err != nil {
		return nil, fmt.Errorf("slice image: %w", err)
	}
	if slices == nil {
		// Image fits within maxHeight.
		return c.parseOne(ctx, imagePath)
	}

	var allBlocks []document.Block
	for i, sl := range slices {
		if c.onProgress != nil {
			c.onProgress(i+1, len(slices), sl.Y)
		}
		doc, err := c.parseSlice(ctx, sl.Data, fmt.Sprintf("slice_%d.jpg", i))
		if err != nil {
			return nil, fmt.Errorf("slice %d: %w", i, err)
		}
		// Offset block Y coordinates by this slice's position.
		for _, blk := range doc.Blocks {
			if !blk.Polygon.IsZero() {
				for j := range blk.Polygon.Points {
					blk.Polygon.Points[j].Y += sl.Y
				}
			}
			allBlocks = append(allBlocks, blk)
		}
	}

	doc := &document.Document{
		Width:  imgW,
		Height: imgH,
		Blocks: allBlocks,
	}
	// Re-sort merged blocks.
	doc, err = layout.Sort()(doc)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (c *Client) parseOne(ctx context.Context, imagePath string) (*document.Document, error) {
	cl := client.New(
		client.WithBaseURL(c.baseURL),
		client.WithModel(c.model),
	)
	pipe := pipeline.New().
		Use(cl).
		Decode(c.provider.Decoder()).
		PostProcess(layout.Sort()).
		Image(imagePath)
	if c.systemPrompt != "" {
		pipe.SystemPrompt(c.systemPrompt)
	}
	return pipe.Run(ctx)
}

func (c *Client) parseSlice(ctx context.Context, data []byte, name string) (*document.Document, error) {
	cl := client.New(
		client.WithBaseURL(c.baseURL),
		client.WithModel(c.model),
	)
	pipe := pipeline.New().
		Use(cl).
		Decode(c.provider.Decoder()).
		PostProcess(layout.Sort())
	if c.systemPrompt != "" {
		pipe.SystemPrompt(c.systemPrompt)
	}
	return pipe.RunWithReader(ctx, bytes.NewReader(data), name)
}

// ParseImageReader runs OCR on an image from an io.Reader.
func (c *Client) ParseImageReader(ctx context.Context, r io.Reader, imagePath string) (*document.Document, error) {
	cl := client.New(
		client.WithBaseURL(c.baseURL),
		client.WithModel(c.model),
	)
	pipe := pipeline.New().
		Use(cl).
		Decode(c.provider.Decoder()).
		PostProcess(layout.Sort())
	if c.systemPrompt != "" {
		pipe.SystemPrompt(c.systemPrompt)
	}
	return pipe.RunWithReader(ctx, r, imagePath)
}

// Convenience output functions.

// Markdown renders the document as Markdown.
func Markdown(doc *document.Document) (string, error) {
	return output.Markdown(doc)
}

// JSON renders the document as indented JSON.
func JSON(doc *document.Document) (string, error) {
	return output.JSON(doc)
}

// HTML renders the document as positioned HTML.
// imageSrc is used as the src attribute of the background <img> tag.
func HTML(doc *document.Document, imageSrc string) (string, error) {
	return output.HTML(doc, imageSrc)
}

// Text renders the document as plain text.
func Text(doc *document.Document) (string, error) {
	return output.Text(doc)
}
