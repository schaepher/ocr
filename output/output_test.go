package output

import (
	"image"
	"strings"
	"testing"

	"github.com/schaepher/ocr/document"
)

func TestJSON(t *testing.T) {
	doc := testDoc()
	got, err := JSON(doc)
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}

	if !strings.Contains(got, `"width": 1920`) {
		t.Errorf("JSON missing width: %s", got)
	}
	if !strings.Contains(got, `"text": "hello"`) {
		t.Errorf("JSON missing text: %s", got)
	}
	if !strings.Contains(got, `"polygon"`) {
		t.Errorf("JSON missing polygon: %s", got)
	}
}

func TestCompactJSON(t *testing.T) {
	doc := testDoc()
	got, err := CompactJSON(doc)
	if err != nil {
		t.Fatalf("CompactJSON() error: %v", err)
	}

	if strings.Contains(got, "\n") {
		t.Errorf("CompactJSON should be single line: %s", got)
	}
}

func TestMarkdown(t *testing.T) {
	doc := testDoc()
	got, err := Markdown(doc)
	if err != nil {
		t.Fatalf("Markdown() error: %v", err)
	}

	if !strings.Contains(got, "hello") {
		t.Errorf("Markdown missing text: %s", got)
	}
	if !strings.Contains(got, "world") {
		t.Errorf("Markdown missing second block: %s", got)
	}
}

func TestHTML(t *testing.T) {
	doc := testDoc()
	got, err := HTML(doc, "test.png")
	if err != nil {
		t.Fatalf("HTML() error: %v", err)
	}

	if !strings.Contains(got, "<!DOCTYPE html>") {
		t.Errorf("HTML missing doctype: %s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("HTML missing text: %s", got)
	}
	if !strings.Contains(got, "ocr-block") {
		t.Errorf("HTML missing ocr-block class: %s", got)
	}
}

func TestText(t *testing.T) {
	doc := testDoc()
	got, err := Text(doc)
	if err != nil {
		t.Fatalf("Text() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %s", len(lines), got)
	}
}

func TestEmptyDocJSON(t *testing.T) {
	doc := &document.Document{Width: 100, Height: 100}
	got, err := JSON(doc)
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}
	if !strings.Contains(got, `"blocks": []`) && !strings.Contains(got, `"blocks": null`) {
		t.Errorf("unexpected empty JSON: %s", got)
	}
}

func TestEmptyDocMarkdown(t *testing.T) {
	doc := &document.Document{}
	got, _ := Markdown(doc)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestEmptyDocText(t *testing.T) {
	doc := &document.Document{}
	got, _ := Text(doc)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func testDoc() *document.Document {
	return &document.Document{
		Width:  1920,
		Height: 1080,
		Blocks: []document.Block{
			{
				Text: "hello",
				Polygon: document.Polygon{
					Points: []image.Point{{0, 0}, {50, 0}, {50, 20}, {0, 20}},
				},
			},
			{
				Text: "world",
				Polygon: document.Polygon{
					Points: []image.Point{{60, 0}, {110, 0}, {110, 20}, {60, 20}},
				},
			},
		},
	}
}
