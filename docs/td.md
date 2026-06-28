# OCR Go SDK — 设计文档

## 背景

基于 TD.md 中的详细设计，实现一个完整的 Go SDK，用于调用 LM Studio (OpenAI 兼容 API) 进行 VLM OCR 推理，解析 PaddleOCR-VL / Qwen3-VL 的 `<|LOC_xxx|>` token 输出，并生成结构化 Document 及 Markdown/HTML/JSON 输出。

通过 **Provider 模式** 支持未来扩展更多 VLM 模型。同时提供 **PaddleOCR Python 本地 provider**，绕过 VLM 直接调用本地 PaddleOCR 库，获得像素级位置精度。

## 目录结构

```
ocr/
├── go.mod                         # module github.com/schaepher/ocr
├── ocr.go                         # 根包：便捷 API (New(provider).ParseImage())
├── ocr_test.go                    # 根包测试
├── document/
│   ├── polygon.go                 # Polygon ([]image.Point) 及相关方法
│   ├── block.go                   # Block {Text, Polygon, Confidence}
│   └── document.go                # Document {Width, Height, Blocks}, Processor 类型
├── provider/
│   ├── provider.go                # Provider 接口 (Model/Prompt/Decoder)
│   ├── paddleocrvl/
│   │   ├── paddleocrvl.go         # PaddleOCR-VL Provider 实现
│   │   └── prompt.go              # PaddleOCR-VL 系统提示词
│   ├── qwen/
│   │   └── qwen3vl/
│   │       ├── qwen3vl.go         # Qwen3-VL Provider 实现
│   │       └── prompt.go          # Qwen3-VL 系统提示词
│   └── paddleocrpy/
│       ├── paddleocrpy.go         # 调用 Python PaddleOCR 子进程
│       └── ocr_helper.py          # Python OCR 脚本（自动切片 + enable_mkldnn=False）
├── decoder/
│   └── decoder.go                 # Decoder 接口定义
├── decoder/paddleocrvl/
│   ├── loc.go                     # LOC Token 解析（正则、坐标缩放）
│   └── decoder.go                 # PaddleOCR-VL Decoder 实现
├── client/
│   └── client.go                  # OpenAI 兼容 HTTP 客户端 (含 Streaming SSE)
├── imageutil/
│   └── slice.go                   # 图片切片（SliceImage, GetDimensions, SaveSlices）
├── layout/
│   ├── sort.go                    # Y 轴聚类 + X 轴排序（阅读顺序）
│   ├── paragraph.go               # 段落合并
│   └── table.go                   # 表格检测（骨架）
├── output/
│   ├── json.go                    # JSON 输出
│   ├── markdown.go                # Markdown 输出
│   ├── html.go                    # HTML 输出 (SVG textLength 自动填满 block)
│   └── text.go                    # 纯文本输出
├── pipeline/
│   └── pipeline.go                # Pipeline 编排 (Use → Decode → PostProcess → Run)
├── cmd/ocr/
│   └── main.go                    # CLI 工具 (ocr 命令)
├── docs/
│   └── td.md                      # 本文档
├── README.md
└── LICENSE
```

## 架构设计

### Provider 模式

```
┌──────────────────────────────────────┐
│             ocr.New(provider)         │  ← 根包 API
├──────────────────────────────────────┤
│  provider.Provider                    │
│  ├── DefaultModel() string           │
│  ├── SystemPrompt() string           │
│  └── Decoder() decoder.Decoder       │
├──────────────────────────────────────┤
│  pipeline.Pipeline                    │
│  ├── client.Client  (HTTP 请求)       │
│  ├── decoder.Decoder (解析输出)       │
│  └── []document.Processor (后处理)   │
└──────────────────────────────────────┘
```

Provider 封装了每个模型的特有信息（模型名、提示词、解码器）。根包 `ocr` 接受一个 Provider 参数，无需硬编码任何模型细节。扩展新模型只需实现 `provider.Provider` 接口。

### 三种 Provider

| Provider | 类型 | 引擎 | 位置精度 | 说明 |
|----------|------|------|---------|------|
| `paddleocrvl` | VLM | PaddleOCR-VL-1.6 via LM Studio | ~4px/LOC-step | 默认，需要切片避免大图 resize |
| `qwen3vl` | VLM | qwen/qwen3-vl-8b via LM Studio | ~4px/LOC-step | Qwen3 视觉语言模型 |
| `paddleocrpy` | 本地 | PaddleOCR Python 库 | 像素级 | 直接调用 Python，`enable_mkldnn=False` |

### Pipeline 流程

```
Image File
    │
    ▼
[图片超 maxHeight?]──是──► imageutil.SliceImage()  # 水平切片
    │                                                   │
    否                                                  ▼
    │                                       [--save-slices?]──是──► 保存切片 JPEG 到 {name}/ 目录
    │                                                   │
    ▼                                                   ▼
client.ImageToBase64()              pipeline.Run() × N slices
    │                                   │
    ▼                                   ▼
client.Chat()                     decoder.Decode() × N
    │                                   │
    ▼                                   ▼
decoder.Decode(raw, imgSize)      合并 Y-offset + layout.Sort()
    │
    ▼
layout.Sort()                 # Y 聚类 → X 排序 (阅读顺序)
    │
    ▼
output.Markdown/HTML/JSON()   # 渲染为指定格式
```

### 大图切片

当图片高度超过 `max-height`（默认不切片），自动水平切片。每片独立送 VLM 推理，坐标解码后加上 Y 偏移合并。

- **切片高度**：`max-height`（默认 4000，须 < 4000 避免 PaddleOCR resize 失真）
- **重叠**：`--overlap`（默认 200px），防止跨边界文本被截断
- **`--page N`**：只 OCR 第 N 页（1-based），调试用
- **`--save-slices`**：将切片保存为 `{图片名}/001.jpg … 00N.jpg`，同时生成对应的 `001.raw.json` 和 `001.html`

### Raw 缓存与回放

`--raw`（默认开启）将每次 VLM 原始输出保存为 `.raw.json`。再次运行时，如果 `.raw.json` 已存在，则跳过 API 调用直接回放。

- 单文件：`{图片名}.raw.json`
- 切片模式 + `--save-slices`：合并文件 `{图片名}.raw.json` + 每个切片的 `{图片名}/00N.raw.json`

### HTML 输出设计

HTML 使用 SVG `<text>` 元素叠加在背景图上：

- `viewBox` 精确匹配 block 像素尺寸
- `preserveAspectRatio="none"` 拉伸填满 SVG 元素
- `textLength` + `lengthAdjust="spacingAndGlyphs"` 自动填满 block 宽度
- `font-size = block 高度`（横排）或 `font-size = block 宽度`（竖排）
- 竖排文字：`writing-mode="vertical-rl"`，从 block 中心展开 (`y=bh/2`)
- 默认 `fill: transparent`，hover 时显示黑字白底

## 实现顺序

### Phase 1: 基础数据类型
1. `go.mod` — `module github.com/schaepher/ocr`
2. `document/polygon.go` — Polygon 结构体，Bounds()、Center() 方法
3. `document/block.go` — Block 结构体
4. `document/document.go` — Document 结构体 + Processor 类型定义

### Phase 2: LOC Token 解析
5. `decoder/decoder.go` — Decoder 接口
6. `decoder/paddleocrvl/loc.go` — 正则解析 `<\|LOC_(\d+)\|>`、坐标缩放 (loc * width / 1000)
7. `decoder/paddleocrvl/decoder.go` — Decoder 实现

### Phase 3: HTTP 客户端
8. `client/client.go` — OpenAI 兼容 API 客户端（Chat Completion、Streaming、图片 base64 编码）

### Phase 4: 布局分析
9. `layout/sort.go` — Y 轴聚类 (±10px) → X 轴排序
10. `layout/paragraph.go` — 同行块合并
11. `layout/table.go` — 表格检测（骨架）

### Phase 5: 输出格式
12. `output/json.go` — JSON 序列化
13. `output/markdown.go` — Markdown 渲染
14. `output/html.go` — HTML 渲染（SVG textLength 自动填满 block）
15. `output/text.go` — 纯文本

### Phase 6: Pipeline 编排
16. `pipeline/pipeline.go` — Pipeline 构建器

### Phase 7: Provider 模式
17. `provider/provider.go` — Provider 接口定义
18. `provider/paddleocrvl/paddleocrvl.go` — PaddleOCR-VL Provider 实现
19. `provider/paddleocrvl/prompt.go` — 系统提示词

### Phase 8: 根包 + CLI
20. `ocr.go` — 便捷 API（接受 Provider 参数）
21. `ocr_test.go` — 根包测试
22. `cmd/ocr/main.go` — CLI 工具

### Phase 9: 图片切片
23. `imageutil/slice.go` — SliceImage、GetDimensions、SaveSlices

### Phase 10: 多 Provider 扩展
24. `provider/qwen/qwen3vl/` — Qwen3-VL Provider
25. `provider/paddleocrpy/` — PaddleOCR Python 本地 Provider

## 关键设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 模块名 | `github.com/schaepher/ocr` | 简短，不绑定特定模型 |
| 根包名 | `package ocr` | 用法自然：`ocr.New(...)` |
| Provider 模式 | `provider.Provider` 接口 | 解耦模型细节，扩展新模型只需实现接口 |
| 提示词位置 | 移入 `provider/*/` | 提示词是模型特定的，不属于通用 pipeline |
| system/user 分离 | Pipeline.SystemPrompt + UserPrompt | PaddleOCR-VL 需要 system message，Qwen3 只需 user text |
| CLI 位置 | `cmd/ocr/` | Go 标准布局，`go install` 生成 `ocr` 命令 |
| `Point` 类型 | 使用 `image.Point` | 标准库已有，避免自定义 |
| `Processor` 类型 | 定义在 `document/` 包 | 避免 layout/pipeline 循环依赖 |
| 坐标缩放 | `loc * imgDim / 1000` (整数运算) | PaddleOCR-VL 在 [0, 999] 网格输出坐标 |
| 阅读顺序 | Y 聚类 (10px 容差) → X 排序 | PaddleOCR Layout 标准做法 |
| Streaming | 在 Client 层拼接 delta → 统一给 Decoder | Decoder 无需关心是否 streaming |
| HTML 文本排版 | SVG `textLength` + `lengthAdjust` | 零系数自动填满 block |
| 竖排检测 | `block.Height > block.Width` | 纯粹几何判断，无阈值 |
| 大图切片 | `imageutil.SliceImage` | 避免 VLM 内部 resize 导致坐标失真 |
| Raw 缓存 | `.raw.json` 文件 | 免重复 API 调用，支持回放修改 prompt 后结果 |
| API 固定参数 | temperature=0, top_p=1, max_tokens=4096 | 确保确定性输出，避免随机性影响坐标 |
| 图片尺寸传递 | Pipeline.ImgSize() | 解码器需要实际尺寸做坐标缩放，不能硬编码 1920×1080 |
| LOC 容差 | 移除上界检查 | Qwen3-VL 可能输出超过 [0, 999] 的 LOC 值 |
| PaddleOCR Python | 子进程调用 + 自动 venv 检测 | 绕过 VLM 精度限制，获得像素级坐标 |

## CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--image` | (与 `--image-dir` 二选一) | 单张图片路径 |
| `--image-dir` | (与 `--image` 二选一) | 图片目录路径 |
| `--provider` | `paddleocrvl` | OCR provider: `paddleocrvl`, `qwen3vl`, `paddleocrpy` |
| `--format` | `html` | 输出格式: `markdown`, `json`, `html`, `text` |
| `--output` | 同图片目录 | 输出文件路径（仅 `--image` 模式） |
| `--base-url` | `http://127.0.0.1:1234/v1` | LM Studio API 地址 |
| `--model` | Provider 默认值 | 模型名称（覆盖 Provider 默认值） |
| `--max-height` | `0` (不切片) | 切片最大高度（paddleocrpy 默认 3800） |
| `--page` | `0` (全部) | 只 OCR 第 N 页（1-based） |
| `--overlap` | `200` | 切片间垂直重叠像素 |
| `--save-slices` | `false` | 保存切片 JPEG + per-slice raw.json + per-slice HTML |
| `--raw` | `true` | 保存/回放 raw model output |
| `--parallel` | `1` | 并发数（仅 `--image-dir` 模式） |

### 典型用法

```bash
# 单图 OCR（默认 provider）
./ocr.exe --image photo.jpg

# VLM 切片 OCR + 保存切片文件
./ocr.exe --provider paddleocrvl --image long.jpg --max-height 4000 --save-slices

# 只 OCR 第 1 页（调试）
./ocr.exe --provider paddleocrvl --image long.jpg --max-height 4000 --page 1

# 本地 PaddleOCR（像素级精度）
./ocr.exe --provider paddleocrpy --image photo.jpg

# Qwen3-VL
./ocr.exe --provider qwen3 --image photo.jpg --max-height 4000

# JSON 输出
./ocr.exe --provider paddleocrvl --image photo.jpg --format json

# 批量处理
./ocr.exe --image-dir ./photos/ --parallel 4
```
