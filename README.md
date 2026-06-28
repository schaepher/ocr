# PaddleOCR-VL Go SDK

A pure Go SDK for running PaddleOCR-VL and other VLM OCR models via LM Studio (OpenAI-compatible API).

## Features

- **No external dependencies** — uses only Go standard library
- **LM Studio / OpenAI-compatible** — works with any OpenAI API endpoint
- **LOC Token Parsing** — converts `<|LOC_xxx|>` tokens to pixel-accurate polygons
- **Layout Analysis** — reading order sorting (Y-cluster → X-sort)
- **Multiple Output Formats** — Markdown, JSON, HTML, Plain Text
- **Pipeline Architecture** — extensible for future VLM models (Qwen2.5-VL, InternVL3, etc.)
- **Streaming Support** — handles SSE streaming responses

## CLI Installation

```bash
go install github.com/schaepher/paddleocrvl/cmd/ocr@latest
```

## CLI Usage

```bash
# Basic OCR to Markdown (default)
ocr --image screenshot.png

# Output as HTML with overlays
ocr --image screenshot.png --format html

# Other formats
ocr --image screenshot.png --format json
ocr --image screenshot.png --format text

# Custom output path
ocr --image screenshot.png --format html --output result.html

# Custom LM Studio endpoint / model
ocr --image screenshot.png --base-url http://127.0.0.1:1234/v1 --model PaddleOCR-VL-1.6
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--image` | *(required)* | Path to image file |
| `--format` | `markdown` | Output format: `markdown`, `json`, `html`, `text` |
| `--output` | same dir as image, auto extension | Output file path |
| `--base-url` | `http://127.0.0.1:1234/v1` | LM Studio API base URL |
| `--model` | `PaddleOCR-VL-1.6` | Model name |

## SDK Usage

```go
package main

import (
    "context"
    "fmt"
    paddleocrvl "github.com/schaepher/paddleocrvl"
)

func main() {
    ctx := context.Background()

    doc, err := paddleocrvl.New().
        LMStudio("http://127.0.0.1:1234/v1").
        Model("PaddleOCR-VL-1.6").
        ParseImage(ctx, "screenshot.png")
    if err != nil {
        panic(err)
    }

    // Output as Markdown
    md, _ := paddleocrvl.Markdown(doc)
    fmt.Println(md)
}
```

## Package Structure

```
├── paddleocrvl.go          # Convenience API (New().LMStudio().Model().ParseImage())
├── client/                 # OpenAI-compatible HTTP client
├── cmd/
│   └── ocr/                # CLI binary
├── decoder/                # Decoder interface
│   └── paddleocrvl/        # PaddleOCR-VL token parser
├── document/               # Core data types (Document, Block, Polygon)
├── layout/                 # Layout analysis (sort, paragraph merge)
├── output/                 # Output renderers (Markdown, JSON, HTML, Text)
├── pipeline/               # Pipeline orchestrator
└── prompt/                 # System prompt constants
```

## How It Works

1. **Image** is read and base64-encoded
2. **LM Studio API** receives the image via OpenAI-compatible chat completion
3. **Raw output** contains text with `<|LOC_xxx|>` location tokens
4. **Decoder** parses LOC tokens into polygons and scales to pixel coordinates
5. **Layout** sorts blocks into natural reading order (Y-cluster → X-sort)
6. **Output** renders the structured Document as Markdown/JSON/HTML/Text

### Coordinate Conversion

PaddleOCR-VL uses a discrete 0–1000 grid for coordinates. The SDK converts:

```
pixelX = locX * imageWidth / 1000
pixelY = locY * imageHeight / 1000
```

## License

MIT
