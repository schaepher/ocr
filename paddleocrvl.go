package paddleocrvl

import (
	"context"
	"io"

	"github.com/schaepher/paddleocrvl/client"
	"github.com/schaepher/paddleocrvl/decoder/paddleocrvl"
	"github.com/schaepher/paddleocrvl/document"
	"github.com/schaepher/paddleocrvl/layout"
	"github.com/schaepher/paddleocrvl/output"
	"github.com/schaepher/paddleocrvl/pipeline"
)

// Client is the convenient top-level API for PaddleOCR-VL.
//
// Usage:
//
//	doc, err := paddleocrvl.New().
//	    LMStudio("http://127.0.0.1:1234/v1").
//	    Model("PaddleOCR-VL-1.6").
//	    ParseImage(ctx, "demo.png")
//
//	fmt.Println(doc.Markdown())
type Client struct {
	baseURL      string
	model        string
	systemPrompt string
}

// New creates a new Client with default settings.
func New() *Client {
	return &Client{
		baseURL: client.DefaultBaseURL,
		model:   client.DefaultModel,
	}
}

// LMStudio sets the LM Studio API base URL.
func (c *Client) LMStudio(url string) *Client {
	c.baseURL = url
	return c
}

// Model sets the model name to use.
func (c *Client) Model(name string) *Client {
	c.model = name
	return c
}

// SystemPrompt sets a custom system prompt for the model.
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
		Decode(paddleocrvl.NewDecoder()).
		PostProcess(
			layout.Sort(),
		).
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
		Decode(paddleocrvl.NewDecoder()).
		PostProcess(
			layout.Sort(),
		)

	if c.systemPrompt != "" {
		pipe.SystemPrompt(c.systemPrompt)
	}

	return pipe.RunWithReader(ctx, r, imagePath)
}

// Convenience output methods on Document are provided via the output package.
// Users can call output.Markdown(doc), output.JSON(doc), etc. directly.

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
