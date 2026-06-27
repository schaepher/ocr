# PaddleOCR-VL Go SDK — 实现计划

## 背景

基于 TD.md 中的详细设计，实现一个完整的 Go SDK，用于调用 LM Studio (OpenAI 兼容 API) 进行 VLM OCR 推理，解析 PaddleOCR-VL 的 `<|LOC_xxx|>` token 输出，并生成结构化 Document 及 Markdown/HTML/JSON 输出。

## 目录结构

```
ocr/
├── go.mod                         # module github.com/schaepher/paddleocrvl
├── paddleocrvl.go                 # 根包：便捷 API (New().LMStudio().Model().ParseImage())
├── document/
│   ├── polygon.go                 # Polygon ([]image.Point) 及相关方法
│   ├── block.go                   # Block {Text, Polygon, Confidence}
│   └── document.go                # Document {Width, Height, Blocks}, Processor 类型
├── prompt/
│   └── prompt.go                  # 系统提示词常量
├── decoder/
│   └── decoder.go                 # Decoder 接口定义
├── decoder/paddleocrvl/
│   ├── loc.go                     # LOC Token 解析（正则、坐标缩放）—— 未导出
│   └── decoder.go                 # PaddleOCRVL Decoder 实现
├── client/
│   └── client.go                  # OpenAI 兼容 HTTP 客户端 (含 Streaming SSE)
├── layout/
│   ├── sort.go                    # Y 轴聚类 + X 轴排序（阅读顺序）
│   ├── paragraph.go               # 段落合并
│   └── table.go                   # 表格检测（骨架）
├── output/
│   ├── json.go                    # JSON 输出
│   ├── markdown.go                # Markdown 输出
│   ├── html.go                    # HTML 输出 (含 position div)
│   └── text.go                    # 纯文本输出
├── pipeline/
│   └── pipeline.go                # Pipeline 编排 (Use → Decode → PostProcess → Run)
├── examples/
│   └── main.go                    # CLI 示例
├── README.md
└── LICENSE
```

## 实现顺序（按编译依赖）

### Phase 1: 基础数据类型
1. `go.mod` — 改为 `module github.com/schaepher/paddleocrvl` + `go 1.25.0`
2. `document/polygon.go` — Polygon 结构体，Bounds()、Center() 方法
3. `document/block.go` — Block 结构体
4. `document/document.go` — Document 结构体 + Processor 类型定义
5. `prompt/prompt.go` — 系统提示词常量

### Phase 2: LOC Token 解析
6. `decoder/decoder.go` — Decoder 接口
7. `decoder/paddleocrvl/loc.go` — 正则解析 `<\|LOC_(\d+)\|>`、坐标缩放 (loc * width / 1000)
8. `decoder/paddleocrvl/decoder.go` — Decoder 实现

### Phase 3: HTTP 客户端
9. `client/client.go` — OpenAI 兼容 API 客户端（Chat Completion、Streaming、图片 base64 编码）

### Phase 4: 布局分析
10. `layout/sort.go` — Y 轴聚类 (±10px) → X 轴排序
11. `layout/paragraph.go` — 同行块合并
12. `layout/table.go` — 表格检测（基础）

### Phase 5: 输出格式
13. `output/json.go` — JSON 序列化
14. `output/markdown.go` — Markdown 渲染
15. `output/html.go` — HTML 渲染（含定位 div）
16. `output/text.go` — 纯文本

### Phase 6: Pipeline 编排
17. `pipeline/pipeline.go` — Pipeline 构建器

### Phase 7: 根包 + 示例
18. `paddleocrvl.go` — 便捷 API
19. `paddleocrvl_test.go` — 根包测试
20. `examples/main.go` — CLI 工具
21. `README.md` — 使用文档
22. `LICENSE` — MIT

## 关键设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| `Point` 类型 | 使用 `image.Point` | 标准库已有，避免自定义 |
| `Processor` 类型 | 定义在 `document/` 包 | 避免 layout/pipeline 循环依赖 |
| 坐标缩放 | `loc * imgSize / 1000` (整数运算) | 无需浮点，结果一致 |
| 阅读顺序 | Y 聚类 (10px 容差) → X 排序 | PaddleOCR Layout 标准做法 |
| Streaming | 在 Client 层拼接 delta → 统一给 Decoder | Decoder 无需关心是否 streaming |

## 估算规模

- 源码：~1200 行
- 测试：~500 行  
- 文档：~100 行
- **总计：~1800 行**

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
