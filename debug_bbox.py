"""Generate HTML with visible bounding boxes for debugging OCR positions."""
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
    xs = [p[0] for p in bbox]
    ys = [p[1] for p in bbox]
    return min(xs), min(ys), max(xs), max(ys)


def generate_bbox_html(blocks, image_width, image_height, image_src):
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
        ".ocr-block { position: absolute; border: 1px solid red; "
        "transition: background 0.15s; }\n"
    )
    parts.append(
        ".ocr-block:hover { background: rgba(255,255,0,0.3); z-index: 1; }\n"
    )
    parts.append(
        ".ocr-label { display: none; position: absolute; top: 0; left: 0; "
        "background: rgba(255,0,0,0.85); color: white; font-size: 10px; "
        "padding: 1px 3px; white-space: nowrap; pointer-events: none; }\n"
    )
    parts.append(".ocr-block:hover .ocr-label { display: block; }\n")
    parts.append("</style>\n</head>\n<body>\n")

    w, h = float(image_width), float(image_height)

    parts.append(
        f"<div class=\"ocr-page\" style=\"width:{image_width}px;max-width:100%;\">\n"
    )
    parts.append(f"<img src=\"{escape(image_src)}\" alt=\"Background image\">\n")

    for i, blk in enumerate(blocks):
        text = blk["text"]
        bbox = blk["bbox"]
        conf = blk["confidence"]
        if not text or not bbox or len(bbox) != 4:
            continue

        min_x, min_y, max_x, max_y = bbox_bounds(bbox)
        bw = max_x - min_x
        bh = max_y - min_y
        if bw < 1:
            bw = 1
        if bh < 1:
            bh = 1

        left_pct = float(min_x) / w * 100.0
        top_pct = float(min_y) / h * 100.0
        width_pct = float(bw) / w * 100.0
        height_pct = float(bh) / h * 100.0

        parts.append(
            f"<div class=\"ocr-block\" "
            f"style=\"left:{left_pct:.4f}%;top:{top_pct:.4f}%;"
            f"width:{width_pct:.4f}%;height:{height_pct:.4f}%;\">"
            f"<span class=\"ocr-label\">[{i}] {escape(text[:20])} ({conf:.2f})</span>"
            f"</div>\n"
        )

    parts.append("</div>\n</body>\n</html>\n")
    return "".join(parts)


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    json_path = os.path.join(script_dir, "ocr_output.json")
    image_path = os.path.expanduser("~/Pictures/ocr/long.jpg")

    blocks = load_ocr_data(json_path)
    print(f"Loaded {len(blocks)} OCR blocks")

    print("Encoding image to base64...")
    mime = "image/jpeg"
    b64 = image_to_base64(image_path)
    image_src = f"data:{mime};base64,{b64}"

    html = generate_bbox_html(blocks, 1080, 22572, image_src)

    out_path = os.path.join(script_dir, "ocr_bbox_debug.html")
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(html)

    size_mb = os.path.getsize(out_path) / (1024 * 1024)
    print(f"Written {out_path} ({size_mb:.1f} MB)")


if __name__ == "__main__":
    main()
