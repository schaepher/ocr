# PaddleOCR 官方版安装与使用记录

> 测试环境：Windows 11 Pro, Python 3.13.7, AMD Ryzen (Radeon 780M 核显), 无 NVIDIA GPU

## 安装步骤

### 1. 创建虚拟环境

```bash
cd ~/Documents/ocr
python -m venv paddleocr_env
source paddleocr_env/Scripts/activate
```

### 2. 安装 PaddlePaddle (CPU 版本)

AMD 显卡在 Windows 下无官方 GPU 加速支持，使用 CPU 版本：

```bash
pip install paddlepaddle==3.3.1
```

安装的其他依赖：`numpy`, `httpx`, `protobuf`, `Pillow`, `opt-einsum`, `networkx`, `safetensors` 等。

### 3. 安装 PaddleOCR

```bash
pip install paddleocr
```

自动安装 `paddlex>=3.7.0`、`opencv-contrib-python`、`shapely`、`pyclipper` 等 OCR 相关依赖。

实际安装版本：`paddleocr-3.7.0`, `paddlex-3.7.2`, `paddlepaddle-3.3.1`

## ⚠️ 关键踩坑

### oneDNN 兼容性 Bug

在 Windows + AMD CPU 环境下，PaddlePaddle 3.3.x 默认启用 oneDNN，但 oneDNN 的 PIR 属性转换有 bug，会报错：

```
NotImplementedError: ConvertPirAttribute2RuntimeAttribute not support
[pir::ArrayAttribute<pir::DoubleAttribute>]
(at paddle\fluid\framework\new_executor\instruction\onednn\onednn_instruction.cc:118)
```

**解决方案**：初始化时显式设置 `enable_mkldnn=False`。

环境变量 `FLAGS_use_mkldnn=0` 和 `paddle.set_flags({'FLAGS_use_mkldnn': 0})` **均无效**，必须在 PaddleOCR 构造参数中传入。

### API 变更 (v3.7)

PaddleOCR 3.7 相比旧版有以下变化：

| 旧 API | 新 API |
|--------|--------|
| `use_angle_cls=True` | `use_textline_orientation=True` |
| `ocr.ocr(img_path, cls=True)` | `ocr.predict(img_path)` |

### 返回结构变化

`predict()` 返回 `list[OCRResult]`，每个 `OCRResult` 是 dict-like 对象，关键字段：

- `rec_texts` — 识别文本列表
- `rec_scores` — 置信度列表
- `dt_polys` — 文本框坐标列表 `[[x1,y1],[x2,y2],[x3,y3],[x4,y4]]`

## 正确用法

```python
import os
from paddleocr import PaddleOCR

# enable_mkldnn=False 是 Windows 下的关键参数
ocr = PaddleOCR(use_textline_orientation=True, lang='ch', enable_mkldnn=False)

result = ocr.predict('image.jpg')

# 提取结构化结果
r = result[0]
output = []
for i in range(len(r['rec_texts'])):
    output.append({
        'bbox': r['dt_polys'][i].tolist(),
        'text': r['rec_texts'][i],
        'confidence': r['rec_scores'][i]
    })
```

## 验证记录

测试图片：`~/Pictures/ocr/long.jpg`（1080×22572 超长微信公众号文章截图）

- 检测文本区域：174 个
- 识别准确率：大部分 0.99+，中文识别效果良好
- 首次运行自动下载模型到 `~/.paddlex/official_models/`（约需下载 5 个模型文件）

## 模型文件

首次运行自动下载的模型（缓存在 `C:\Users\<user>\.paddlex\official_models\`）：

| 模型 | 用途 |
|------|------|
| `PP-LCNet_x1_0_doc_ori` | 文档方向分类 |
| `UVDoc` | 文档矫正/展平 |
| `PP-LCNet_x1_0_textline_ori` | 文本行方向分类 |
| `PP-OCRv6_medium_det` | 文本检测 |
| `PP-OCRv6_medium_rec` | 文本识别 |
