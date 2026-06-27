package output

import (
	"encoding/json"
	"fmt"

	"github.com/schaepher/paddleocrvl/document"
)

// JSON serializes a Document to indented JSON.
func JSON(doc *document.Document) (string, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(data), nil
}

// CompactJSON serializes a Document to compact (single-line) JSON.
func CompactJSON(doc *document.Document) (string, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(data), nil
}
