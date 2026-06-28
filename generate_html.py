"""Generate an HTML overlay for OCR results, matching output/html.go style."""
import base64
import json
import os
import sys
from html import escape


def load_ocr_data(json_path):
    with open(json_path, "r", encoding="utf-8") as f:
        return json.load(f)


def image_to_base64(image_path):
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode("ascii")


def bbox_bounds(bbox):
    """Return (min_x, min_y, max_x, max_y) from 4-point polygon."""
    xs = [p[0] for p in bbox]
    ys = [p[1] for p in bbox]
    return min(xs), min(ys), max(xs), max(ys)


def generate_html(blocks, image_width, image_height, image_src):
    """Generate an HTML overlay matching output/html.go behavior."""

    parts = []
    parts.append("<!DOCTYPE html>\n<html lang=\"zh\">\n<head>\n")
    parts.append("<meta charset=\"utf-8\">\n")
    parts.append(
        "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1.0\">\n"
    )
    parts.append("<style>\n")
    parts.append(
        "body { margin: 0; background: #222; display: flex; justify-content: center; }\n"
    )
    parts.append(
        ".ocr-page { position: relative; overflow: hidden; background: #fff; "
        "box-shadow: 0 2px 12px rgba(0,0,0,0.4); }\n"
    )
    parts.append(".ocr-page img { display: block; width: 100%; height: auto; }\n")
    parts.append(
        ".ocr-block { position: absolute; overflow: hidden; "
        "transition: background 0.15s, box-shadow 0.15s; }\n"
    )
    parts.append(
        ".ocr-block text { fill: rgba(255,0,0,0.7); transition: fill 0.15s; "
        "font-family: sans-serif; }\n"
    )
    parts.append(
        ".ocr-block { background: rgba(0,255,0,0.08); }"
    )
    parts.append(
        ".ocr-block:hover { background: rgba(255,255,255,0.9); "
        "box-shadow: 0 1px 6px rgba(0,0,0,0.25); z-index: 1; }\n"
    )
    parts.append(".ocr-block:hover text { fill: #000; }\n")
    parts.append("</style>\n</head>\n<body>\n")

    w, h = float(image_width), float(image_height)

    parts.append(
        f"<div class=\"ocr-page\" style=\"width:{image_width}px;max-width:100%;\">\n"
    )
    parts.append(f"<img src=\"{escape(image_src)}\" alt=\"Background image\">\n")

    for blk in blocks:
        text = blk["text"]
        bbox = blk["bbox"]
        if not text:
            continue

        if not bbox or len(bbox) != 4:
            # No bbox — render as static block
            parts.append(
                "<div class=\"ocr-block\" style=\"position:static;padding:4px 8px;\">"
            )
            parts.append(
                "<svg width=\"100%\" height=\"1.2em\" viewBox=\"0 0 1 1\" "
                "preserveAspectRatio=\"none\">"
            )
            parts.append(
                f"<text x=\"0\" y=\"0.9\" textLength=\"1\" "
                f"lengthAdjust=\"spacingAndGlyphs\" fill=\"#000\">{escape(text)}</text>"
            )
            parts.append("</svg></div>\n")
            continue

        min_x, min_y, max_x, max_y = bbox_bounds(bbox)
        bw = max_x - min_x
        bh = max_y - min_y
        if bw < 1:
            bw = 1
        if bh < 1:
            bh = 1

        # Slightly smaller so text doesn't overflow
        bh_pad = bh - 2
        if bh_pad < 1:
            bh_pad = 1
        fh = bh - 1
        if fh < 1:
            fh = 1
        fw = bw - 1
        if fw < 1:
            fw = 1

        left_pct = float(min_x) / w * 100.0
        top_pct = float(min_y) / h * 100.0
        width_pct = float(bw) / w * 100.0
        height_pct = float(bh) / h * 100.0

        vertical = bh > bw

        parts.append(
            "<svg class=\"ocr-block\" overflow=\"hidden\" "
            f"style=\"left:{left_pct:.4f}%;top:{top_pct:.4f}%;"
            f"width:{width_pct:.4f}%;height:{height_pct:.4f}%;\" "
            f"viewBox=\"0 0 {bw} {bh}\" preserveAspectRatio=\"none\">"
        )

        if vertical:
            parts.append(
                f"<text x=\"{bw // 2}\" y=\"{bh // 2}\" "
                f"writing-mode=\"vertical-rl\" "
                f"textLength=\"{bh}\" lengthAdjust=\"spacingAndGlyphs\" "
                f"font-size=\"{bw}\" text-anchor=\"middle\">{escape(text)}</text>"
            )
        else:
            parts.append(
                f"<text x=\"0\" y=\"{bh_pad}\" "
                f"textLength=\"{bw}\" lengthAdjust=\"spacingAndGlyphs\" "
                f"font-size=\"{fh}\">{escape(text)}</text>"
            )

        parts.append("</svg>\n")

    parts.append("</div>\n</body>\n</html>\n")
    return "".join(parts)


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    json_path = os.path.join(script_dir, "ocr_output.json")
    image_path = os.path.expanduser("~/Pictures/ocr/long.jpg")

    if not os.path.exists(json_path):
        print(f"Error: {json_path} not found. Run OCR first.", file=sys.stderr)
        sys.exit(1)
    if not os.path.exists(image_path):
        print(f"Error: {image_path} not found.", file=sys.stderr)
        sys.exit(1)

    blocks = load_ocr_data(json_path)
    print(f"Loaded {len(blocks)} OCR blocks")

    # Use base64 data URI for self-contained HTML
    print("Encoding image to base64...")
    mime = "image/jpeg"
    b64 = image_to_base64(image_path)
    image_src = f"data:{mime};base64,{b64}"

    html = generate_html(blocks, 1080, 22572, image_src)

    out_path = os.path.join(script_dir, "ocr_output.html")
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(html)

    size_mb = os.path.getsize(out_path) / (1024 * 1024)
    print(f"Written {out_path} ({size_mb:.1f} MB)")


if __name__ == "__main__":
    main()
