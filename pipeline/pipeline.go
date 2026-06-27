package pipeline

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/schaepher/paddleocrvl/client"
	"github.com/schaepher/paddleocrvl/decoder"
	"github.com/schaepher/paddleocrvl/document"
	"github.com/schaepher/paddleocrvl/prompt"
)

// Pipeline orchestrates the full OCR flow:
// Image → Client → Decoder → PostProcessors → Document
type Pipeline struct {
	cl      *client.Client
	dec     decoder.Decoder
	procs   []document.Processor
	prompt  string
	imgPath string
}

// New creates an empty Pipeline.
func New() *Pipeline {
	return &Pipeline{}
}

// Use sets the HTTP client for API communication.
func (p *Pipeline) Use(c *client.Client) *Pipeline {
	p.cl = c
	return p
}

// Decode sets the decoder for parsing raw model output.
func (p *Pipeline) Decode(d decoder.Decoder) *Pipeline {
	p.dec = d
	return p
}

// PostProcess appends layout post-processors (Sort, MergeParagraph, etc.).
func (p *Pipeline) PostProcess(procs ...document.Processor) *Pipeline {
	p.procs = append(p.procs, procs...)
	return p
}

// Image sets the image file path.
func (p *Pipeline) Image(path string) *Pipeline {
	p.imgPath = path
	return p
}

// SystemPrompt sets a custom system prompt.
func (p *Pipeline) SystemPrompt(s string) *Pipeline {
	p.prompt = s
	return p
}

// Run executes the full pipeline:
//  1. Read and encode the image.
//  2. Send to LM Studio via client.
//  3. Decode raw output into Document.
//  4. Apply post-processors.
func (p *Pipeline) Run(ctx context.Context) (*document.Document, error) {
	if p.cl == nil {
		return nil, fmt.Errorf("pipeline: no client set (call .Use())")
	}
	if p.dec == nil {
		return nil, fmt.Errorf("pipeline: no decoder set (call .Decode())")
	}
	if p.imgPath == "" {
		return nil, fmt.Errorf("pipeline: no image path set (call .Image())")
	}

	// 1. Encode image.
	imageURI, err := client.ImageToBase64(p.imgPath)
	if err != nil {
		return nil, fmt.Errorf("pipeline: encode image: %w", err)
	}

	userPrompt := p.prompt
	if userPrompt == "" {
		userPrompt = prompt.SpottingSystemPrompt
	}

	// 2. Send API request.
	req := client.BuildVisionRequest("", "", userPrompt, imageURI)
	raw, err := p.cl.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pipeline: api call: %w", err)
	}
	if raw == "" {
		return nil, fmt.Errorf("pipeline: empty response from API")
	}

	// 3. Decode.
	// Estimate image size from file. This is a best-effort dimension;
	// the client doesn't have access to actual pixel dimensions.
	// Users should explicitly set dimensions via the decoder if needed.
	imgSize := image.Point{X: 1920, Y: 1080} // fallback

	doc, err := p.dec.Decode(raw, imgSize)
	if err != nil {
		return nil, fmt.Errorf("pipeline: decode: %w", err)
	}

	// 4. Run post-processors.
	for _, proc := range p.procs {
		doc, err = proc(doc)
		if err != nil {
			return nil, fmt.Errorf("pipeline: postprocess: %w", err)
		}
	}

	return doc, nil
}

// RunWithReader executes the pipeline using an io.Reader for the image.
// The imagePath is used for MIME type detection only.
func (p *Pipeline) RunWithReader(ctx context.Context, r io.Reader, imagePath string) (*document.Document, error) {
	if p.cl == nil {
		return nil, fmt.Errorf("pipeline: no client set (call .Use())")
	}
	if p.dec == nil {
		return nil, fmt.Errorf("pipeline: no decoder set (call .Decode())")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("pipeline: read image: %w", err)
	}

	mimeType := detectMimeType(imagePath, data)
	encoded := encodeBase64(data)
	imageURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	userPrompt := p.prompt
	if userPrompt == "" {
		userPrompt = prompt.SpottingSystemPrompt
	}

	req := client.BuildVisionRequest("", "", userPrompt, imageURI)
	raw, err := p.cl.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pipeline: api call: %w", err)
	}
	if raw == "" {
		return nil, fmt.Errorf("pipeline: empty response from API")
	}

	imgSize := image.Point{X: 1920, Y: 1080}
	doc, err := p.dec.Decode(raw, imgSize)
	if err != nil {
		return nil, fmt.Errorf("pipeline: decode: %w", err)
	}

	for _, proc := range p.procs {
		doc, err = proc(doc)
		if err != nil {
			return nil, fmt.Errorf("pipeline: postprocess: %w", err)
		}
	}

	return doc, nil
}

// detectMimeType guesses the MIME type from the file extension or magic bytes.
func detectMimeType(path string, data []byte) string {
	lower := strings.ToLower(path)

	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	default:
		if len(data) >= 8 {
			if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
				return "image/png"
			}
			if bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
				return "image/jpeg"
			}
		}
		return "image/png"
	}
}

// encodeBase64 encodes data to a base64 string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
