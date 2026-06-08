# 数学建模三人小队 Agent 框架

本框架模拟真实数学建模比赛中的三人协作：建模手、编程手、论文手并行推进、反复沟通、交叉复核；父代理像指导老师一样控制方向和节奏。

## 组织结构

```text
竞赛队长 / Orchestrator
  负责阶段、分工、共享黑板、checkpoint、冲突裁决和最终交付。

建模手 / Modeling Lead
  题目解析专员
  数据理解与审计专员
  抽象建模专员
  模型选择专员
  验证方案专员

编程手 / Coding Lead
  代码实现专员
  代码审核专员
  结果整理专员

论文手 / Paper Lead
  摘要标题专员
  问题重述专员
  问题分析专员
  假设符号专员
  模型建立求解专员
  敏感性分析专员
  模型评价专员
  引用格式终审专员
  论文章节写作专员（兜底）
  论文一致性审核专员（跨章节）
```

当前运行层面由父代理统一调度所有 Lead 和专项代理。Lead 代理负责提出需求、交接和复核；专项代理负责执行具体任务。这样能避免嵌套委派失控，也符合现有多代理中“子代理禁止再次 task”的运行约束。

## 核心流程

```text
开题会
  -> 题目解析
  -> 数据审计
  -> 初始模型路线
  -> C1 checkpoint

建模-编程对接
  -> 模型规格
  -> 编程任务单
  -> C2 checkpoint

编程实现与审核
  -> 代码实现
  -> 代码审核
  -> 结果整理
  -> 建模复核
  -> C3 checkpoint

论文同步写作
  -> 编程手解释结果和图表
  -> 论文手分章节写作
  -> 建模手复核公式和假设
  -> 编程手复核数值和图表
  -> C4 checkpoint

终审导出
  -> 论文一致性审核
  -> 三位 Lead 给出 review_card
  -> C5 checkpoint
  -> 官方 Word 模板转 LaTeX
  -> 拼接正文模板
  -> 编译 PDF 和 Word
```

## 交叉协作

不是单向流水线：

- 建模手在代码结果出来后复核题意、约束、假设和评价指标。
- 编程手在论文写作时解释代码逻辑、图表含义和关键数值。
- 论文手在开题阶段就参与结构规划，并持续反馈材料缺口。
- 父代理发现分歧时要求相关角色给证据，而不是直接拍板。

## 共享黑板卡片

所有重要信息通过卡片沉淀，供后续代理读取和复核：

- `problem_card`：题目背景、问题列表、目标、约束、交付要求。
- `data_card`：文件、字段、单位、缺失、异常、可用性、风险。
- `model_card`：变量、参数、假设、目标函数、约束、候选模型、选择理由。
- `coding_task_card`：任务、输入、输出、算法、评价指标、文件路径。
- `result_card`：核心结果、指标、表格、图表、失败尝试、复现步骤。
- `figure_card`：图片路径、图示对象、关键数值、论文 caption。
- `paper_section_card`：章节目标、依赖材料、公式、图表、LaTeX 正文、材料缺口。
- `review_card`：审核对象、审核人、问题、严重程度、证据、是否通过。

## 文件落点

- 父代理：[agents/orchestrator.md](../agents/orchestrator.md)
- 三位 Lead：[agents/modeling-lead.md](../agents/modeling-lead.md)、[agents/coding-lead.md](../agents/coding-lead.md)、[agents/paper-lead.md](../agents/paper-lead.md)
- 专项代理：
  - 建模：`agents/problem-parser.md`、`agents/data-auditor.md`、`agents/abstract-modeler.md`、`agents/model-selector.md`、`agents/validation-designer.md`
  - 编程：`agents/code-implementer.md`、`agents/code-reviewer.md`、`agents/result-structurer.md`
  - 论文：`agents/paper-abstract-title-writer.md`、`agents/paper-problem-restatement-writer.md`、`agents/paper-problem-analysis-writer.md`、`agents/paper-assumptions-symbols-writer.md`、`agents/paper-model-solution-writer.md`、`agents/paper-sensitivity-analysis-writer.md`、`agents/paper-model-evaluation-writer.md`、`agents/paper-citation-final-reviewer.md`、`agents/paper-section-writer.md`、`agents/paper-consistency-reviewer.md`
- 协作协议 Skill：[skills/team-collaboration-workflow/SKILL.md](../skills/team-collaboration-workflow/SKILL.md)
- 自建工具箱：[toolbox/](../toolbox/)
- 论文输出产线：[toolbox/report/](../toolbox/report/)

## 后续扩展

1. 项目黑板已切换为 `problem`、`data`、`model`、`code`、`result`、`figure`、`paper`、`review` category，后续可以把 Markdown body schema 再升级为机器可校验 JSON schema。
2. 将常用方法沉淀为 `knowledge_base/modeling/` 和 `toolbox/methods/` 的方法卡，再逐步配套 CLI/MCP 工具。
3. 把 Jupyter 持久化执行器抽成独立 CLI，再包装为 MCP，让编程手稳定生成 notebook、图表和 `result.json`。
4. 给论文导出增加官方 Word 模板转 LaTeX、LaTeX 编译 PDF、同源导出 Word、图片路径、公式、引用和卡片一致性检查。
