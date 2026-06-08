# 构建输出目录

这里存放 `mathmodel-report` 生成的 `paper.tex`、`paper.pdf`、`paper.docx` 和 `compile_report.json`。

## 命令

```bash
toolbox/cli/mathmodel-report convert-template official.docx --out toolbox/report/latex-templates/official
toolbox/cli/mathmodel-report assemble \
  --template-dir toolbox/report/latex-templates/official \
  --body toolbox/report/body-templates/china-mathmodel-body.md \
  --out toolbox/report/build/current
toolbox/cli/mathmodel-report compile \
  --tex toolbox/report/build/current/paper.tex \
  --out toolbox/report/build/current
```
