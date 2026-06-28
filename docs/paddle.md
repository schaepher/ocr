实际上，**LM Studio 本身无法保证**。决定是否输出 `<|LOC_xxx|>` 的不是 LM Studio，而是 **Prompt + Sampling 参数 + PaddleOCR-VL 的任务模式**。

PaddleOCR-VL 有几种不同的输出模式：

| Prompt                 | 输出               | 是否带 LOC       |
| ---------------------- | ---------------- | ------------- |
| `OCR:`                 | 文本（有时带 LOC，有时没有） | 不稳定           |
| `Spotting:`            | 文本 + LOC         | **应该始终带 LOC** |
| `Table Recognition:`   | HTML/Table       | 不带 LOC        |
| `Formula Recognition:` | LaTeX            | 不带 LOC        |

如果你的目标是**做后处理**，那么应该始终使用 **Spotting 模式**。

---

## 1. Prompt 必须固定

建议不要写：

```text
Please OCR this image.
```

或者

```text
Extract all text.
```

而是直接：

```text
Spotting:
```

或者更严格一点：

```text
Spotting:
Output every detected text together with its location tokens.
Do not omit any location tokens.
```

甚至可以写：

```text
You are PaddleOCR-VL.

Output every text line followed immediately by exactly eight location tokens.

Example:

Hello<|LOC_10|><|LOC_20|><|LOC_30|><|LOC_20|><|LOC_30|><|LOC_40|><|LOC_10|><|LOC_40|>

Do not explain.
Do not summarize.
Only output spotting results.

Spotting:
```

虽然模型不一定完全遵循，但比单纯 `Spotting:` 更稳定。

---

## 2. Temperature 必须为 0

LM Studio：

```
Temperature = 0
```

否则模型可能觉得：

```
LOC 不重要
```

于是直接省略。

---

## 3. Top P

建议

```
Top P = 1
```

不要：

```
0.8
```

否则 LOC token 可能被采样掉。

---

## 4. Repeat Penalty

建议：

```
1.0
```

不要：

```
1.2
1.3
```

LOC token 本身就是大量重复：

```
LOC
LOC
LOC
LOC
...
```

Penalty 太高会让模型倾向于不输出。

---

## 5. 不要开启 JSON Mode

LM Studio 有：

```
Response Format
```

不要：

```
JSON
```

否则模型会努力输出：

```json
{
  "text":"..."
}
```

LOC 基本都会没了。

---

## 6. Max Tokens 要够

LOC token 数量其实很多。

例如一句：

```
Hello
```

其实会生成：

```
Hello
LOC
LOC
LOC
LOC
LOC
LOC
LOC
LOC
```

一句文字可能有十几个 token。

如果：

```
Max Tokens = 512
```

模型很容易后半截只输出文字。

建议：

```
2048
```

甚至：

```
4096
```

---

## 7. 不要开启 Thinking

如果 LM Studio 使用的是：

```
Reasoning Mode
```

最好关闭。

OCR 模型根本不需要思考。

---

# 能不能做到 100%？

答案是：**不能，仅靠 Prompt 做不到。**

原因是 PaddleOCR-VL 本质上仍是一个生成模型。

例如它完全可能输出：

```
微博正文
公开
最安神
```

而不是：

```
微博正文<LOC...>
公开<LOC...>
最安神<LOC...>
```

Prompt 无法硬性约束 Decoder。

---

# 官方为什么可以做到？

官方 **并不是靠 Prompt**。

官方 Python 推理实际上会在生成阶段限制可选 Token。

也就是类似：

```
Allowed Tokens

↓

文字

↓

LOC

↓

LOC

↓

LOC
```

Decoder 会控制：

```
Text

↓

必须 LOC

↓

必须 LOC

↓

必须 LOC
```

这属于 **Logits Processor / Constrained Decoding**。

llama.cpp 当前没有针对 PaddleOCR-VL 实现这一套，所以 LM Studio 无法保证。


