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
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"

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

type rawFile struct {
	Params rawParams  `json:"params"`
	Data   []rawEntry `json:"data"`
}
type rawParams struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	BaseURL   string `json:"baseURL"`
	MaxHeight int    `json:"maxHeight,omitempty"`
}

const locFormatReminder = "\nOutput with <|LOC_x|><|LOC_y|> location tokens."

type rawEntry struct {
	Index  int    `json:"index"`
	Y      int    `json:"y"`
	Height int    `json:"height"`
	Raw    string `json:"raw"`
}

type Client struct {
	provider     provider.Provider
	baseURL      string
	model        string
	systemPrompt string
	maxHeight    int // 0 means no slicing
	overlap      int
	onProgress   ProgressFunc
	debugPath    string
	maxRetries   int
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
		maxRetries:   3,
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

// MaxRetries sets the max retries when the model output has no LOC tokens.
// Default is 3.
func (c *Client) MaxRetries(n int) *Client {
	c.maxRetries = n
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

// Debug enables debug mode: raw model output is saved to path (JSON array).
// If path already exists, the model is skipped and cached raw output is replayed.
func (c *Client) Debug(path string) *Client {
	c.debugPath = path
	return c
}

// ParseImage runs OCR on an image file and returns a structured Document.
// If MaxHeight is set and the image exceeds it, the image is split into
// overlapping horizontal slices, each processed separately, then merged.
func (c *Client) ParseImage(ctx context.Context, imagePath string) (*document.Document, error) {
	// Debug replay: load cached raw outputs and decode them.
	if c.debugPath != "" {
		if data, err := os.ReadFile(c.debugPath); err == nil {
			var rf rawFile
			if err := json.Unmarshal(data, &rf); err == nil && len(rf.Data) > 0 {
				return c.replayRaws(rf)
			}
			var raws []string
			if err := json.Unmarshal(data, &raws); err == nil {
				return c.replayRawStrings(raws, imagePath)
			}
		}
	}

	if c.maxHeight <= 0 {
		doc, raw, err := c.parseOne(ctx, imagePath)
		if err != nil {
			return nil, err
		}
		c.saveDebug([]rawEntry{{Index: 0, Y: 0, Height: doc.Height, Raw: raw}})
		return doc, nil
	}

	slices, imgW, imgH, err := imageutil.SliceImage(imagePath, c.maxHeight, c.overlap)
	if err != nil {
		return nil, fmt.Errorf("slice image: %w", err)
	}
	if slices == nil {
		doc, raw, err := c.parseOne(ctx, imagePath)
		if err != nil {
			return nil, err
		}
		c.saveDebug([]rawEntry{{Index: 0, Y: 0, Height: doc.Height, Raw: raw}})
		return doc, nil
	}

	var allBlocks []document.Block
	var entries []rawEntry
	for i, sl := range slices {
		if c.onProgress != nil {
			c.onProgress(i+1, len(slices), sl.Y)
		}
		doc, raw, err := c.parseSlice(ctx, sl.Data, fmt.Sprintf("slice_%d.jpg", i))
		if err != nil {
			return nil, fmt.Errorf("slice %d: %w", i, err)
		}
		entries = append(entries, rawEntry{Index: i, Y: sl.Y, Height: sl.Height, Raw: raw})
		for _, blk := range doc.Blocks {
			if !blk.Polygon.IsZero() {
				for j := range blk.Polygon.Points {
					blk.Polygon.Points[j].Y += sl.Y
				}
			}
			allBlocks = append(allBlocks, blk)
		}
	}
	c.saveDebug(entries)

	doc := &document.Document{Width: imgW, Height: imgH, Blocks: allBlocks}
	doc, err = layout.Sort()(doc)
	return doc, err
}

func (c *Client) saveDebug(entries []rawEntry) {
	if c.debugPath == "" || len(entries) == 0 {
		return
	}
	rf := rawFile{
		Params: rawParams{
			Provider:  c.provider.Name(),
			Model:     c.model,
			BaseURL:   c.baseURL,
			MaxHeight: c.maxHeight,
		},
		Data: entries,
	}
	data, _ := json.MarshalIndent(rf, "", "  ")
	os.WriteFile(c.debugPath, data, 0644)
}

func (c *Client) replayRawStrings(raws []string, imagePath string) (*document.Document, error) {
	w, h := 1920, 1080
	if slices, _, imgH, err := imageutil.SliceImage(imagePath, c.maxHeight, c.overlap); err == nil && slices != nil {
		h = imgH
	}

	imgSize := image.Point{X: w, Y: h}
	dec := c.provider.Decoder()
	var allBlocks []document.Block
	for i, raw := range raws {
		doc, err := dec.Decode(raw, imgSize)
		if err != nil {
			return nil, fmt.Errorf("decode cached slice %d: %w", i, err)
		}
		allBlocks = append(allBlocks, doc.Blocks...)
	}
	doc := &document.Document{Width: w, Height: h, Blocks: allBlocks}
	return layout.Sort()(doc)
}

func (c *Client) replayRaws(rf rawFile) (*document.Document, error) {
	dec := c.provider.Decoder()
	imgSize := image.Point{X: 1920, Y: 1080}
	var allBlocks []document.Block
	for _, e := range rf.Data {
		doc, err := dec.Decode(e.Raw, imgSize)
		if err != nil {
			return nil, fmt.Errorf("decode cached slice %d: %w", e.Index, err)
		}
		for _, blk := range doc.Blocks {
			if !blk.Polygon.IsZero() {
				for j := range blk.Polygon.Points {
					blk.Polygon.Points[j].Y += e.Y
				}
			}
			allBlocks = append(allBlocks, blk)
		}
	}
	doc := &document.Document{Width: 1920, Height: 1080, Blocks: allBlocks}
	return layout.Sort()(doc)
}

func (c *Client) parseOne(ctx context.Context, imagePath string) (*document.Document, string, error) {
	var lastRaw string
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		cl := client.New(
			client.WithBaseURL(c.baseURL),
			client.WithModel(c.model),
		)
		pipe := pipeline.New().
			Use(cl).
			Decode(c.provider.Decoder()).
			PostProcess(layout.Sort()).
			Image(imagePath)
		prompt := c.systemPrompt
		if attempt > 0 {
			prompt += locFormatReminder
			fmt.Fprintf(os.Stderr, "Retrying with format reminder (attempt %d)...\n", attempt+1)
		}
		if prompt != "" {
			pipe.SystemPrompt(prompt)
		}
		doc, err := pipe.Run(ctx)
		lastRaw = pipe.Raw
		if err != nil {
			return nil, lastRaw, err
		}
		if hasLocTokens(doc) || attempt == c.maxRetries-1 {
			return doc, lastRaw, nil
		}
	}
	return nil, lastRaw, fmt.Errorf("no LOC tokens after %d retries", c.maxRetries)
}

func (c *Client) parseSlice(ctx context.Context, data []byte, name string) (*document.Document, string, error) {
	var lastRaw string
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		cl := client.New(
			client.WithBaseURL(c.baseURL),
			client.WithModel(c.model),
		)
		pipe := pipeline.New().
			Use(cl).
			Decode(c.provider.Decoder()).
			PostProcess(layout.Sort())
		prompt := c.systemPrompt
		if attempt > 0 {
			prompt += locFormatReminder
			fmt.Fprintf(os.Stderr, "Retrying with format reminder (attempt %d)...\n", attempt+1)
		}
		if prompt != "" {
			pipe.SystemPrompt(prompt)
		}
		doc, err := pipe.RunWithReader(ctx, bytes.NewReader(data), name)
		lastRaw = pipe.Raw
		if err != nil {
			return nil, lastRaw, err
		}
		if hasLocTokens(doc) || attempt == c.maxRetries-1 {
			return doc, lastRaw, nil
		}
	}
	return nil, lastRaw, fmt.Errorf("no LOC tokens after %d retries", c.maxRetries)
}

func hasLocTokens(doc *document.Document) bool {
	for _, blk := range doc.Blocks {
		if !blk.Polygon.IsZero() {
			return true
		}
	}
	return false
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
