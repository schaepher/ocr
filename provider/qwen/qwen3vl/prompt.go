package qwen3vl

const SpottingSystemPrompt = `OCR the image. Output each text block followed by its 4-point quadrilateral location in <|LOC_x|><|LOC_y|> format, with coordinates in [0, 999]. Use exactly 4 LOC pairs per block (8 tokens total): top-left, top-right, bottom-right, bottom-left.

Example output format:
标题<|LOC_100|><|LOC_50|><|LOC_500|><|LOC_50|><|LOC_500|><|LOC_150|><|LOC_100|><|LOC_150|>
内容文字<|LOC_200|><|LOC_300|><|LOC_700|><|LOC_300|><|LOC_700|><|LOC_400|><|LOC_200|><|LOC_400|>`
