# 论文输出产线

数学建模论文默认采用 LaTeX 作为主格式，最后同时导出 PDF 和 Word。

## 目标流程

```text
官方 Word 模板
  -> 一比一转成 LaTeX 样式模板
  -> 拼接正文模板与 Agent 生成章节
  -> 注入图表、公式、引用和附录
  -> 编译 PDF
  -> 从同一份源内容导出 Word
```

## 目录规划

```text
toolbox/report/
  official-word-templates/   # 用户放官方 .docx 模板
  latex-templates/           # 转换后的 .tex/.cls/.sty 模板
  body-templates/            # 论文正文结构模板
  build/                     # 编译脚本、临时输出和导出说明
```

当前已通过 `toolbox/cli/mathmodel-report` 实现基础产线，并在 `toolbox/tools/` 中提供 MCP 工具包装。

## 转换要求

- 官方 Word 模板必须先被解析成结构化版式清单：页边距、页眉页脚、标题层级、字体、字号、行距、摘要、关键词、正文、图表题注、参考文献和附录。
- LaTeX 模板要尽量一比一复刻官方 Word 模板；无法完全复刻的地方必须记录差异。
- 正文内容来自库中的正文模板与三人小队产出的 `paper_section_card`，不直接拼接未经审核的自由文本。
- 图表路径、公式编号、引用编号必须在编译前检查。

## 目标产物

- `paper.tex`：最终 LaTeX 源文件。
- `paper.pdf`：正式提交 PDF。
- `paper.docx`：Word 版本，用于检查、修改或按赛事要求提交。
- `compile_report.json`：模板差异、编译日志、缺失图片、引用检查和导出状态。

## CLI 命令

```bash
toolbox/cli/mathmodel-report convert-template official.docx --out toolbox/report/latex-templates/official
toolbox/cli/mathmodel-report assemble \
  --template-dir toolbox/report/latex-templates/official \
  --body toolbox/report/body-templates/china-mathmodel-body.md \
  --sections-dir path/to/paper_section_cards \
  --out toolbox/report/build/current
toolbox/cli/mathmodel-report preflight --tex toolbox/report/build/current/paper.tex
toolbox/cli/mathmodel-report compile --tex toolbox/report/build/current/paper.tex --out toolbox/report/build/current
```

## MCP 工具

- `mathmodel-report-convert-template`
- `mathmodel-report-assemble`
- `mathmodel-report-preflight`
- `mathmodel-report-compile`
