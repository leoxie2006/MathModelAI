# MathModelAI 完善 TODO

这份清单用于把当前三人小队框架完善成可实际跑完整数学建模比赛流程的工作台。

## P0 基础整理

- [x] 搭建“竞赛队长 + 建模手 + 编程手 + 论文手 + 专项代理”的三层 Agent 框架。
- [x] 增加三人协作协议 Skill。
- [x] 建立 `toolbox/` 工具箱根目录。
- [x] 将现有 `exec` 与 `execute-python-script` 工具放入 `toolbox/tools/`。
- [x] 将 `config.yaml` 的 `security.tools_dir` 指向 `toolbox/tools`。
- [x] 将项目黑板的网安 category 改造成数学建模 category：`problem`、`data`、`model`、`code`、`result`、`figure`、`paper`、`review`。
- [x] 为共享卡片建立第一版 Markdown body schema：`problem_card`、`data_card`、`model_card`、`coding_task_card`、`result_card`、`figure_card`、`paper_section_card`、`review_card`。
- [x] 在前端项目页进一步展示审核状态和 checkpoint。

## P1 三人小队协作流

- [x] 父代理实现标准会议模板：开题会、建模-编程对接、编程-建模复核、编程-论文同步、论文-建模复核、论文-编程复核、终审会。
- [x] Lead 代理输出 `handoff_request` 后，由父代理统一调度专项代理。
- [x] 建模手复核代码结果：题意、变量、约束、单位、假设、评价指标。
- [x] 编程手同步论文素材：代码逻辑、图表解释、核心数值、复现路径。
- [x] 论文手回查建模和编程：公式、假设、图表、结果、引用。
- [x] 增加返工规则和次数上限：数学错误回到 `model_card`，代码错误回到 `coding_task_card`，论文缺口回到 `result_card`/`figure_card`。

## P2 自建数学工具箱

- [x] 建立 `toolbox/methods/optimization/`：线性规划、整数规划、非线性规划、多目标优化、遗传算法、模拟退火。
- [x] 建立 `toolbox/methods/prediction/`：回归、XGBoost、ARIMA、指数平滑、GM(1,1)。
- [x] 建立 `toolbox/methods/evaluation/`：AHP、熵权法、TOPSIS、模糊综合评价、PCA、DEA。
- [x] 建立 `toolbox/methods/simulation/`：蒙特卡洛、排队论、元胞自动机、系统动力学。
- [x] 为每个方法补方法卡：适用条件、输入输出、假设、验证、常见错误、未来 CLI。
- [x] 将第一批成熟评价方法做成 `mathmodel-toolkit` CLI：TOPSIS、熵权法、AHP。
- [x] 将第一批稳定 CLI 包装为 MCP 工具 YAML，放入 `toolbox/tools/`。
- [x] 给已实现工具调用结果统一输出 `result_card` 与 `figure_card`。
- [x] 扩展 `mathmodel-toolkit` 到优化、预测、仿真方法。

## P3 编程与验证

- [x] 抽取持久化 Jupyter 执行器，形成独立 CLI。
- [x] 代码实现工具输出 notebook、脚本、图表、结果 JSON 和复现命令。
- [x] 代码审核工具检查数据泄露、指标计算、约束实现、随机种子和图表一致性。
- [x] 结果整理工具将原始输出转换成 `result_card`、`figure_card` 和论文素材。
- [x] 增加敏感性分析、边界测试、基线模型对照和稳健性检查工具。

## P4 LaTeX 论文产线

- [x] 官方 Word 模板导入：用户上传 `.docx` 模板到 `toolbox/report/official-word-templates/`。
- [x] Word 模板结构解析：页边距、页眉页脚、标题层级、字体、字号、行距、摘要、关键词、图表题注、参考文献。
- [x] 一比一转换为 LaTeX 模板：输出 `main.tex`、`.cls`/`.sty` 和 `template_fidelity_report.md`。
- [x] 正文模板拼接：从 `toolbox/report/body-templates/` 读取正文骨架，拼接经审核的 `paper_section_card`。
- [x] 编译前检查：图片路径、公式编号、引用编号、材料缺口、图表解释一致性。
- [x] 编译 PDF：优先支持 XeLaTeX / latexmk；失败时保存编译日志。
- [x] 导出 Word：从同一份结构化源内容生成 `paper.docx`，不能与 PDF 使用两套正文。
- [x] 输出 `compile_report.json`：模板差异、编译状态、缺失文件、引用检查、导出路径。

## P5 前端工作台

- [x] 增加三人小队视图：建模手、编程手、论文手的当前任务和交接卡。
- [x] 增加工具箱管理页：方法卡、CLI 状态、MCP 工具注册状态。
- [x] 增加论文产线页：上传官方 Word 模板、查看 LaTeX 转换差异、编译 PDF/Word。
- [x] 增加 checkpoint 页面：用户确认模型路线、核心结果、论文初稿和最终导出。

## P6 知识库

- [x] 将常用方法卡同步到 `knowledge_base/modeling/`，供 Agent 检索。
- [x] 增加优秀论文结构模板、摘要模板、图表 caption 模板。
- [x] 增加常见扣分点：数据泄露、单位错误、无约束优化、图文不一致、引用不实。
- [x] 建立案例复盘库：赛题、模型路线、代码结果、论文结构、评审意见。
