# OCR Go SDK — 设计文档

## 背景

基于 TD.md 中的详细设计，实现一个完整的 Go SDK，用于调用 LM Studio (OpenAI 兼容 API) 进行 VLM OCR 推理，解析 PaddleOCR-VL 的 `<|LOC_xxx|>` token 输出，并生成结构化 Document 及 Markdown/HTML/JSON 输出。

通过 **Provider 模式** 支持未来扩展更多 VLM 模型（Qwen2.5-VL、InternVL3 等）。

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
│   └── paddleocrvl/
│       ├── paddleocrvl.go         # PaddleOCR-VL Provider 实现
│       └── prompt.go              # PaddleOCR-VL 系统提示词
├── decoder/
│   └── decoder.go                 # Decoder 接口定义
├── decoder/paddleocrvl/
│   ├── loc.go                     # LOC Token 解析（正则、坐标缩放）
│   └── decoder.go                 # PaddleOCR-VL Decoder 实现
├── client/
│   └── client.go                  # OpenAI 兼容 HTTP 客户端 (含 Streaming SSE)
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

### Pipeline 流程

```
Image File
    │
    ▼
client.ImageToBase64()        # 读取并 base64 编码
    │
    ▼
client.Chat()                 # POST /v1/chat/completions (含 SSE streaming)
    │
    ▼
decoder.Decode(raw, size)     # 解析 <|LOC_xxx|> token → Document {Blocks}
    │
    ▼
layout.Sort()                 # Y 聚类 → X 排序 (阅读顺序)
    │
    ▼
output.Markdown/HTML/JSON()   # 渲染为指定格式
```

### HTML 输出设计

HTML 使用 SVG `<text>` 元素叠加在背景图上：

- `viewBox` 精确匹配 block 像素尺寸
- `preserveAspectRatio="none"` 拉伸填满 SVG 元素
- `textLength` + `lengthAdjust="spacingAndGlyphs"` 自动填满 block 宽度
- `font-size = block 高度`（横排）或 `font-size = block 宽度`（竖排）
- 竖排文字：`writing-mode="vertical-rl"`，从 block 中心展开 (`y=bh/2`)
- 零硬编码系数，纯几何推导

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
16. `pipeline/pipeline.go` — Pipeline 构建器（SystemPrompt 由调用方传入，无硬编码默认值）

### Phase 7: Provider 模式
17. `provider/provider.go` — Provider 接口定义
18. `provider/paddleocrvl/paddleocrvl.go` — PaddleOCR-VL Provider 实现
19. `provider/paddleocrvl/prompt.go` — 系统提示词（从独立 prompt/ 包移入）

### Phase 8: 根包 + CLI
20. `ocr.go` — 便捷 API（接受 Provider 参数）
21. `ocr_test.go` — 根包测试（mock Provider + mock Decoder）
22. `cmd/ocr/main.go` — CLI 工具（ocr 命令，支持 --image / --image-dir / --parallel）
23. `README.md` — 使用文档
24. `LICENSE` — MIT

## 关键设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| 模块名 | `github.com/schaepher/ocr` | 简短，不绑定特定模型 |
| 根包名 | `package ocr` | 用法自然：`ocr.New(...)` |
| Provider 模式 | `provider.Provider` 接口 | 解耦模型细节，扩展新模型只需实现接口 |
| 提示词位置 | 移入 `provider/paddleocrvl/` | 提示词是模型特定的，不属于通用 pipeline |
| CLI 位置 | `cmd/ocr/` | Go 标准布局，`go install` 生成 `ocr` 命令 |
| `Point` 类型 | 使用 `image.Point` | 标准库已有，避免自定义 |
| `Processor` 类型 | 定义在 `document/` 包 | 避免 layout/pipeline 循环依赖 |
| 坐标缩放 | `loc * imgSize / 1000` (整数运算) | 无需浮点，结果一致 |
| 阅读顺序 | Y 聚类 (10px 容差) → X 排序 | PaddleOCR Layout 标准做法 |
| Streaming | 在 Client 层拼接 delta → 统一给 Decoder | Decoder 无需关心是否 streaming |
| HTML 文本排版 | SVG `textLength` + `lengthAdjust` | 零系数自动填满 block，无需猜测字体 metrics |
| 竖排检测 | `block.Height > block.Width` | 纯粹几何判断，无阈值 |
| text 空白清理 | 仅在 decoder 层 TrimSpace | 上游统一清理，下游无需重复 |

## CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--image` | (必选，与 `--image-dir` 二选一) | 单张图片路径 |
| `--image-dir` | (必选，与 `--image` 二选一) | 图片目录路径，递归扫描 |
| `--format` | `markdown` | 输出格式: `markdown`, `json`, `html`, `text` |
| `--output` | 同图片目录，自动扩展名 | 输出文件路径（仅 `--image` 模式） |
| `--base-url` | `http://127.0.0.1:1234/v1` | LM Studio API 地址 |
| `--model` | Provider 默认值 | 模型名称（覆盖 Provider 默认值） |
| `--parallel` | `1` | 并发数（仅 `--image-dir` 模式） |

## 估算规模

- 源码：~1500 行
- 测试：~500 行
- 文档：~150 行
- **总计：~2150 行**

## 验证方式

1. `go build ./...` — 全部通过编译
2. `go vet ./...` — 无警告
3. `go test ./...` — 全部测试通过
4. 关键测试：LOC 解析（含坐标缩放）、Layout 排序、Output 格式

### 测试重点

- `decoder/paddleocrvl/loc_test.go` — 单元测试覆盖率 >95%
  - 单块解析、多块解析、无 LOC token、奇数 token 报错、坐标缩放
- `layout/sort_test.go` — 阅读顺序正确性
- `output/*_test.go` — 对比 golden file
- `ocr_test.go` — mock Provider + mock Decoder，测试便捷 API 行为
