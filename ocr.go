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
	"path/filepath"

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
	ImgWidth  int    `json:"imgWidth,omitempty"`
	ImgHeight int    `json:"imgHeight,omitempty"`
}

// const locFormatReminder = "\nOutput with <|LOC_x|><|LOC_y|> location tokens."

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
	page         int // 1-based page to OCR, 0 means all
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
		maxRetries:   5,
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

// Page sets which page (1-based slice index) to OCR. 0 means all pages.
func (c *Client) Page(n int) *Client {
	c.page = n
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

	// When maxHeight is 0, process as single image (no slicing).
	if c.maxHeight <= 0 {
		imgW, imgH, err := imageutil.GetDimensions(imagePath)
		if err != nil {
			return nil, fmt.Errorf("get image dimensions: %w", err)
		}
		imgSize := image.Point{X: imgW, Y: imgH}
		doc, raw, err := c.parseOne(ctx, imagePath, imgSize)
		if err != nil {
			return nil, err
		}
		c.saveDebug([]rawEntry{{Index: 0, Y: 0, Height: doc.Height, Raw: raw}}, imgW, imgH)
		return doc, nil
	}

	slices, imgW, imgH, err := imageutil.SliceImage(imagePath, c.maxHeight, c.overlap)
	if err != nil {
		return nil, fmt.Errorf("slice image: %w", err)
	}

	if slices == nil {
		// Image fits within maxHeight, no slicing needed.
		imgSize := image.Point{X: imgW, Y: imgH}
		doc, raw, err := c.parseOne(ctx, imagePath, imgSize)
		if err != nil {
			return nil, err
		}
		c.saveDebug([]rawEntry{{Index: 0, Y: 0, Height: doc.Height, Raw: raw}}, imgW, imgH)
		return doc, nil
	}

	// Save slices to subdirectory named after the image (without extension).
	sliceDir, err := imageutil.SaveSlices(slices, imagePath, c.maxHeight, c.overlap)
	if err != nil {
		return nil, fmt.Errorf("save slices: %w", err)
	}

	// Determine which slice indices to process.
	start, end := 0, len(slices)
	if c.page > 0 {
		// --page is 1-based.
		start = c.page - 1
		end = c.page
		if start >= len(slices) {
			return nil, fmt.Errorf("page %d out of range (1-%d)", c.page, len(slices))
		}
	}

	var allBlocks []document.Block
	var entries []rawEntry
	combinedDebugPath := c.debugPath // original combined raw.json path
	for i := start; i < end; i++ {
		sl := slices[i]
		pageNum := i + 1
		if c.onProgress != nil {
			c.onProgress(pageNum, len(slices), sl.Y)
		}
		slicePath := filepath.Join(sliceDir, fmt.Sprintf("%03d.jpg", pageNum))

		// Per-slice debug path: each slice gets its own .raw.json.
		sliceDebugPath := slicePath[:len(slicePath)-4] + ".raw.json"

		sliceSize := image.Point{X: imgW, Y: sl.Height}
		doc, raw, err := c.parseOne(ctx, slicePath, sliceSize)

		// Save per-slice raw.json (independent replay unit).
		if sliceDebugPath != "" {
			origPath := c.debugPath
			c.debugPath = sliceDebugPath
			c.saveDebug([]rawEntry{{Index: 0, Y: 0, Height: sl.Height, Raw: raw}}, imgW, sl.Height)
			c.debugPath = origPath
		}

		if err != nil {
			entries = append(entries, rawEntry{Index: i, Y: sl.Y, Height: sl.Height, Raw: raw})
			c.debugPath = combinedDebugPath
			c.saveDebug(entries, imgW, imgH)
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
	c.debugPath = combinedDebugPath
	c.saveDebug(entries, imgW, imgH)

	// For single-page, return only that page's blocks without Y adjustment.
	// The coordinates are relative to the slice, which is correct for the page.
	doc := &document.Document{Width: imgW, Height: imgH, Blocks: allBlocks}
	doc, err = layout.Sort()(doc)
	return doc, err
}

func (c *Client) saveDebug(entries []rawEntry, imgW, imgH int) {
	if c.debugPath == "" || len(entries) == 0 {
		return
	}
	rf := rawFile{
		Params: rawParams{
			Provider:  c.provider.Name(),
			Model:     c.model,
			BaseURL:   c.baseURL,
			MaxHeight: c.maxHeight,
			ImgWidth:  imgW,
			ImgHeight: imgH,
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
	imgW := rf.Params.ImgWidth
	imgH := rf.Params.ImgHeight
	if imgW == 0 {
		imgW = 1920
	}
	if imgH == 0 {
		imgH = 1080
	}
	imgSize := image.Point{X: imgW, Y: imgH}
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
	doc := &document.Document{Width: imgW, Height: imgH, Blocks: allBlocks}
	return layout.Sort()(doc)
}

func (c *Client) parseOne(ctx context.Context, imagePath string, imgSize image.Point) (*document.Document, string, error) {
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
			Image(imagePath).
			ImgSize(imgSize).
			SystemPrompt(c.systemPrompt).
			UserPrompt("Spotting:")
		if attempt > 0 {
			pipe.UserPrompt("Spotting:\nOutput with <|LOC_x|><|LOC_y|> location tokens.")
			fmt.Fprintf(os.Stderr, "Retrying with format reminder (attempt %d)...\n", attempt)
		}
		doc, err := pipe.Run(ctx)
		lastRaw = pipe.Raw
		if err != nil {
			return nil, lastRaw, err
		}
		if hasLocTokens(doc) {
			return doc, lastRaw, nil
		}
	}
	return nil, lastRaw, fmt.Errorf("no LOC tokens after %d retries", c.maxRetries)
}

func (c *Client) parseSlice(ctx context.Context, data []byte, name string, imgSize image.Point) (*document.Document, string, error) {
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
			ImgSize(imgSize).
			SystemPrompt(c.systemPrompt).
			UserPrompt("Spotting:")
		if attempt > 0 {
			pipe.UserPrompt("Spotting:\nOutput with <|LOC_x|><|LOC_y|> location tokens.")
			fmt.Fprintf(os.Stderr, "Retrying with format reminder (attempt %d)...\n", attempt)
		}
		doc, err := pipe.RunWithReader(ctx, bytes.NewReader(data), name)
		lastRaw = pipe.Raw
		if err != nil {
			return nil, lastRaw, err
		}
		if hasLocTokens(doc) {
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
