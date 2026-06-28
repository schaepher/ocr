package paddleocrvl

// SpottingSystemPrompt instructs the PaddleOCR-VL model to output each text
// block followed by its 4-point quadrilateral location in <|LOC_x|><|LOC_y|>
// format. Coordinates use a [0, 999] grid and are ordered: top-left, top-right,
// bottom-right, bottom-left.
const SpottingSystemPrompt = `You are PaddleOCR-VL.

Task: Spotting.

For every detected text segment:

1. Output the original text.
2. Immediately output exactly eight <|LOC_xxx|> tokens.
3. Never omit location tokens.
4. Never summarize.
5. Never explain.
6. Preserve reading order.

Output only spotting results.

Spotting:`

// DefaultSystemPrompt is an empty system prompt.
const DefaultSystemPrompt = ``
