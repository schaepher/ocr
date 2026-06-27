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

## Quick Start

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

    // Output as JSON
    json, _ := paddleocrvl.JSON(doc)
    fmt.Println(json)
}
```

## CLI Usage

```bash
go run examples/main.go --image demo.png --format markdown
go run examples/main.go --image demo.png --format json
go run examples/main.go --image demo.png --format html
go run examples/main.go --image demo.png --format text
```

## Pipeline API

For more control, use the pipeline builder directly:

```go
package main

import (
    "context"
    "fmt"
    "github.com/schaepher/paddleocrvl/client"
    "github.com/schaepher/paddleocrvl/decoder/paddleocrvl"
    "github.com/schaepher/paddleocrvl/layout"
    "github.com/schaepher/paddleocrvl/output"
    "github.com/schaepher/paddleocrvl/pipeline"
)

func main() {
    ctx := context.Background()

    cl := client.New(
        client.WithBaseURL("http://127.0.0.1:1234/v1"),
        client.WithModel("PaddleOCR-VL-1.6"),
    )

    doc, err := pipeline.New().
        Use(cl).
        Decode(paddleocrvl.NewDecoder()).
        PostProcess(
            layout.Sort(),
            layout.MergeParagraph(),
        ).
        Image("demo.png").
        Run(ctx)
    if err != nil {
        panic(err)
    }

    html, _ := output.HTML(doc)
    fmt.Println(html)
}
```

## Package Structure

```
├── paddleocrvl.go          # Convenience API (New().LMStudio().Model().ParseImage())
├── client/                 # OpenAI-compatible HTTP client
├── decoder/                # Decoder interface
│   └── paddleocrvl/        # PaddleOCR-VL token parser
├── document/               # Core data types (Document, Block, Polygon)
├── layout/                 # Layout analysis (sort, paragraph merge, table detect)
├── output/                 # Output renderers (Markdown, JSON, HTML, Text)
├── pipeline/               # Pipeline orchestrator
├── prompt/                 # System prompt constants
└── examples/               # CLI demo
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
