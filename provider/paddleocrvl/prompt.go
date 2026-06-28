package paddleocrvl

// SpottingSystemPrompt instructs the PaddleOCR-VL model to output each text
// block followed by its 4-point quadrilateral location in <|LOC_x|><|LOC_y|>
// format. Coordinates use a [0, 999] grid and are ordered: top-left, top-right,
// bottom-right, bottom-left.
const SpottingSystemPrompt = `OCR the image. Output each text block followed by its 4-point quadrilateral location in <|LOC_x|><|LOC_y|> format, with coordinates in [0, 999]. Use exactly 4 LOC pairs per block (8 tokens total): top-left, top-right, bottom-right, bottom-left.

Example output format:
标题<|LOC_100|><|LOC_50|><|LOC_500|><|LOC_50|><|LOC_500|><|LOC_150|><|LOC_100|><|LOC_150|>
内容<|LOC_200|><|LOC_300|><|LOC_700|><|LOC_300|><|LOC_700|><|LOC_400|><|LOC_200|><|LOC_400|>`

// DefaultSystemPrompt is an empty system prompt.
const DefaultSystemPrompt = ``
