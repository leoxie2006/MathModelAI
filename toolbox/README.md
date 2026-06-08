# MathModelAI 工具箱

这里是自建数学建模工具箱的根目录。工具箱分成三层：

- `tools/`：可被系统加载的 MCP 工具 YAML。现有通用工具、评价/优化/预测/仿真方法工具、P3 编程验证工具和 P4 报告工具已放在这里。
- `methods/`：常用数学建模方法模板，已覆盖优化、预测、评价、仿真四类 21 张方法卡。
- `cli/`：稳定 CLI 入口，当前提供 `mathmodel-toolkit` 的评价、线性规划、回归、GM(1,1)、指数平滑、蒙特卡洛，以及 `mathmodel-pipeline` 的执行、审核、整理和验证流程。
- `report/`：论文输出产线，包括官方 Word 模板转 LaTeX、正文模板拼接、PDF 编译和 Word 导出。

## 设计原则

- 方法模板先文档化，再工具化。
- 稳定、可复用、确定性的流程做成 CLI，再包成 MCP 工具。
- Agent 只负责选择和解释，工具负责计算、检查和导出。
- 每个工具必须有输入、输出、适用条件、失败模式和验证方法。

## 目录规划

```text
toolbox/
  tools/                 # MCP 工具 YAML
  methods/               # 方法模板
  cli/                   # mathmodel-toolkit CLI
  examples/              # CLI 示例数据
  report/                # LaTeX/Word/PDF 输出产线
```
