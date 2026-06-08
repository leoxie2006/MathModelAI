# 内置工具

`toolbox/tools/` 下的 YAML 文件会被加载为 MCP 工具模板。当前包含通用工具和第一批稳定数学建模工具：

- `execute-python-script.yaml`：在虚拟环境中运行 Python 片段，适合建模计算、数据处理、仿真、优化和可视化。
- `exec.yaml`：执行系统命令，适合文件检查、依赖安装、LaTeX/Word 导出和工程辅助操作。
- `mathmodel-topsis.yaml`：调用 `mathmodel-toolkit topsis`，输出 `result_card`、`figure_card` 和排名表。
- `mathmodel-entropy-weight.yaml`：调用 `mathmodel-toolkit entropy-weight`，输出熵权、综合得分和卡片。
- `mathmodel-ahp.yaml`：调用 `mathmodel-toolkit ahp`，输出权重、一致性检验和卡片。
- `mathmodel-linear-programming.yaml`：调用 `mathmodel-toolkit linear-programming`，输出最优解、候选顶点和卡片。
- `mathmodel-regression.yaml`：调用 `mathmodel-toolkit regression`，输出回归系数、预测误差和卡片。
- `mathmodel-gm11.yaml`：调用 `mathmodel-toolkit gm11`，输出灰色预测结果和卡片。
- `mathmodel-exponential-smoothing.yaml`：调用 `mathmodel-toolkit exponential-smoothing`，输出指数平滑预测和卡片。
- `mathmodel-monte-carlo.yaml`：调用 `mathmodel-toolkit monte-carlo`，输出仿真统计、可行率和卡片。
- `mathmodel-run-python.yaml`：运行 Python 建模脚本，输出 notebook、复现命令、`code_card` 和 `result_card`。
- `mathmodel-review-code.yaml`：检查数据泄露、指标、边界条件、随机种子和图表一致性。
- `mathmodel-structure-results.yaml`：把原始结果整理成 `result_card`、`figure_card` 和论文素材。
- `mathmodel-validate-results.yaml`：做结果级边界、基线和敏感性检查，输出 `review_card`。
- `mathmodel-report-convert-template.yaml`：解析官方 Word 模板并转换成 LaTeX 版式模板。
- `mathmodel-report-assemble.yaml`：拼接正文模板和章节卡片，生成 `paper.tex`。
- `mathmodel-report-preflight.yaml`：编译前检查图片路径、材料缺口和引用。
- `mathmodel-report-compile.yaml`：编译 PDF 并从同一 LaTeX 源导出 Word。

新增工具时保持 `name`、`command`、`enabled`、`description` 和 `parameters` 字段完整即可。方法模板和更稳定的 CLI 工具放在 `toolbox/methods/` 与 `toolbox/report/` 下沉淀。
