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
	"context"
	"io"

	"github.com/schaepher/ocr/client"
	"github.com/schaepher/ocr/document"
	"github.com/schaepher/ocr/layout"
	"github.com/schaepher/ocr/output"
	"github.com/schaepher/ocr/pipeline"
	"github.com/schaepher/ocr/provider"
)

// Client is the convenient top-level API for OCR.
type Client struct {
	provider     provider.Provider
	baseURL      string
	model        string
	systemPrompt string
}

// New creates a new Client with the given provider.
// The provider supplies the default model name, system prompt, and decoder.
func New(p provider.Provider) *Client {
	return &Client{
		provider:     p,
		baseURL:      "http://127.0.0.1:1234/v1",
		model:        p.DefaultModel(),
		systemPrompt: p.SystemPrompt(),
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

// ParseImage runs OCR on an image file and returns a structured Document.
func (c *Client) ParseImage(ctx context.Context, imagePath string) (*document.Document, error) {
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
