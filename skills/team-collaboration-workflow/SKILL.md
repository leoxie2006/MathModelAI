---
name: team-collaboration-workflow
description: 数学建模三人小队协作协议，用于父代理组织建模手、编程手、论文手的非线性交叉协作、交接卡和 checkpoint。
metadata:
  version: "0.1.0"
  tags:
    - math-modeling
    - multi-agent
    - workflow
---

# 数学建模三人小队协作协议

用于模拟真实数学建模比赛中的三人分工：建模手、编程手、论文手并行推进、反复交流、交叉复核；父代理像指导老师一样统筹节奏和质量。

## 角色分工

- 父代理：管阶段、分工、共享黑板、checkpoint、返工和最终交付。
- 建模手：管题意、变量、假设、目标函数、约束、模型路线和数学正确性。
- 编程手：管数据处理、模型实现、实验复现、图表、结果和代码质量。
- 论文手：管论文结构、章节写作、图表解释、引用、格式和终稿一致性。
- 论文默认主格式为 LaTeX；官方 Word 模板只作为版式来源，必须先一比一转成 LaTeX 模板，再拼接正文并导出 PDF 与 Word。

## 非线性协作规则

1. 建模手不是只做前期。代码结果和论文模型表达都必须回到建模手复核。
2. 编程手不是只跑代码。编程手必须把代码逻辑、图表和关键数值解释给论文手。
3. 论文手不是最后才加入。开题时就要参与结构规划，并持续反馈哪些材料还不能写。
4. 所有争议回到证据：题面、数据、模型规格、代码输出、图表、验证结果。
5. 子代理禁止私自跳阶段；需要新任务时输出 handoff_request，由父代理调度。

## 交接卡

### problem_card

```text
标题：
背景：
问题列表：
目标：
约束：
数据/附件：
交付要求：
风险：
```

### data_card

```text
文件：
字段：
单位：
缺失/异常：
可用性：
清洗需求：
数据泄露风险：
建模限制：
```

### model_card

```text
问题：
变量：
参数：
目标函数/评价指标：
约束：
假设：
候选模型：
推荐模型：
验证方法：
```

### coding_task_card

```text
任务：
输入：
输出：
算法：
评价指标：
图表要求：
复现要求：
```

### result_card

```text
问题：
模型：
核心指标：
关键结果：
输出文件：
图表：
不确定性：
失败尝试：
```

### figure_card

```text
路径：
对应问题：
图示内容：
关键数值：
论文 caption：
支撑结论：
```

### paper_section_card

```text
章节：
依赖材料：
公式：
图表：
正文：
LaTeX 主文档位置：
需要复核：
材料缺口：
```

### review_card

```text
审核对象：
审核人：
问题：
严重程度：
证据：
建议：
是否通过：
```

## 标准会议

1. 开题会：建模手主导，编程手和论文手参与，确认 problem_card、data_card、模型路线草案。
2. 建模-编程对接会：建模手给 coding_task_card，编程手确认可实现性。
3. 编程-建模复核会：编程手交 result_card，建模手判断是否符合题意和约束。
4. 编程-论文同步会：编程手交 figure_card 和代码思路，论文手同步写作素材。
5. 论文-建模复核会：论文手交模型章节，建模手查公式、假设、结论。
6. 论文-编程复核会：论文手交图表解释和数值引用，编程手查是否与输出一致。
7. 终审会：三位 Lead 都给 review_card，父代理决定是否导出。

每次会议输出 `meeting_card`：

```yaml
meeting_card:
  id: opening/<slug>
  phase: C1
  participants:
    - modeling-lead
    - coding-lead
    - paper-lead
  inputs:
    - problem/requirements
  decisions:
    - <已达成决议>
  blockers:
    - <阻断问题；没有写无>
  required_cards:
    - model/route
  next_owner: modeling-lead
  checkpoint: C1
```

## handoff_request

Lead 代理和专项代理禁止继续调用 `task`，需要新任务时输出：

```yaml
handoff_request:
  requested_agent: code-reviewer
  reason: <为什么需要该专项代理>
  input_cards:
    - <fact_key>
  input_files:
    - <path>
  expected_output_cards:
    - <fact_key 或卡片类型>
  reviewer: <modeling-lead|coding-lead|paper-lead>
  deadline_or_budget: <时间/轮次限制；未知写 none>
```

父代理收到后先检查输入卡片是否存在，再一次只调度一个专项代理；专项代理返回后必须交给 `reviewer` 复核。

## Checkpoint

- C1：题目拆解和初始模型路线。
- C2：主要模型规格和编程任务单。
- C3：核心结果通过建模复核。
- C4：论文初稿通过交叉复核。
- C5：终稿导出前确认，包括 LaTeX 编译 PDF 与同源导出 Word。

## 返工规则

- 严重数学错误：回到 model_card。
- 代码实现不符合模型规格：回到 coding_task_card 或代码实现。
- 结果不稳定或不可解释：回到 validation_plan_card。
- 论文材料缺口：回到 result_card/figure_card。
- 超过时间预算：父代理选择简单可解释方案，记录舍弃理由。

默认最多返工 3 轮。第 3 轮仍未通过时，父代理必须降级方案或请求用户 checkpoint，不能无限循环。
