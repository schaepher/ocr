package document

// Block is a single text region detected by OCR.
type Block struct {
	Text    string  `json:"text"`
	Polygon Polygon `json:"polygon"`
	// Confidence score from the model, typically in [0, 1].
	Confidence float64 `json:"confidence,omitempty"`
}
