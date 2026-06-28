#!/usr/bin/env python3
"""PaddleOCR helper: OCR an image with automatic slicing, output Document JSON.

Usage:
    python ocr_helper.py --image <path> [--max-height 3800] [--overlap 200]

Outputs JSON in document.Document format to stdout:
    {"width": 1080, "height": 22572, "blocks": [{"text": "...", "polygon": {...}}]}
"""
import argparse
import json
import os
import sys
import tempfile

from PIL import Image
from paddleocr import PaddleOCR


def bbox_to_polygon(bbox):
    """Convert PaddleOCR [[x1,y1],[x2,y2],[x3,y3],[x4,y4]] to Polygon Points."""
    return {"Points": [{"X": int(p[0]), "Y": int(p[1])} for p in bbox]}


def ocr_slice(ocr, img, y_start, y_end):
    """OCR a single vertical slice of the image."""
    crop = img.crop((0, y_start, img.width, y_end))
    # Save to temp file (PaddleOCR predict needs a file path)
    with tempfile.NamedTemporaryFile(suffix=".jpg", delete=False) as f:
        crop.save(f.name, "JPEG", quality=92)
        temp_path = f.name

    try:
        result = ocr.predict(temp_path)
        r = result[0]
        blocks = []
        for i in range(len(r["rec_texts"])):
            bbox = r["dt_polys"][i].tolist()
            # Adjust Y coordinates to global image space
            for pt in bbox:
                pt[1] += y_start
            blocks.append({
                "text": r["rec_texts"][i],
                "polygon": bbox_to_polygon(bbox),
                "confidence": r["rec_scores"][i],
            })
        return blocks
    finally:
        os.unlink(temp_path)


def main():
    parser = argparse.ArgumentParser(description="PaddleOCR helper")
    parser.add_argument("--image", required=True, help="Path to image file")
    parser.add_argument("--max-height", type=int, default=3800,
                        help="Max slice height (default: 3800, must be < 4000)")
    parser.add_argument("--overlap", type=int, default=200,
                        help="Vertical overlap between slices (default: 200)")
    args = parser.parse_args()

    if not os.path.exists(args.image):
        print(f"Error: image not found: {args.image}", file=sys.stderr)
        sys.exit(1)

    # Initialize PaddleOCR (disable MKL-DNN for Windows compatibility)
    ocr = PaddleOCR(
        use_textline_orientation=True,
        lang="ch",
        enable_mkldnn=False,
    )

    img = Image.open(args.image)
    img_w, img_h = img.width, img.height

    all_blocks = []

    if img_h <= args.max_height:
        # No slicing needed
        result = ocr.predict(args.image)
        r = result[0]
        for i in range(len(r["rec_texts"])):
            bbox = r["dt_polys"][i].tolist()
            all_blocks.append({
                "text": r["rec_texts"][i],
                "polygon": bbox_to_polygon(bbox),
                "confidence": r["rec_scores"][i],
            })
    else:
        # Slice and OCR
        y = 0
        slice_idx = 0
        while y < img_h:
            y_end = min(y + args.max_height + args.overlap, img_h)
            blocks = ocr_slice(ocr, img, y, y_end)
            print(f"  slice [{slice_idx+1}] y={y}-{y_end}: {len(blocks)} texts",
                  file=sys.stderr)
            all_blocks.extend(blocks)
            y += args.max_height
            slice_idx += 1

    # Output document.Document JSON
    doc = {
        "width": img_w,
        "height": img_h,
        "blocks": all_blocks,
    }
    json.dump(doc, sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()
