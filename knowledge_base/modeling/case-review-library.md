# 数学建模案例复盘库

本库用于沉淀赛后复盘和可复用经验。每个案例按同一结构记录，供父 agent 检索后作为方法选择和评审参考。

## 案例模板

```yaml
id: case-YYYY-topic
contest: 国赛/美赛/校赛/课程
problem: A/B/C/自定义
domain: 交通/物流/生态/医疗/金融/制造/社会治理
task_type: 优化/预测/评价/仿真/机理建模
data_files:
  - 附件1.xlsx
model_route:
  - 问题一: 数据审计 + 基线模型
  - 问题二: 主模型
  - 问题三: 敏感性和策略
toolbox:
  - mathmodel-toolkit topsis
  - mathmodel-pipeline validate
key_results:
  - 核心数值、排名、预测、策略
review_findings:
  - 数学风险
  - 代码复现风险
  - 论文表达风险
lessons:
  - 后续同类题可复用的流程或模板
```

## 示例：多指标方案评价

- 赛题类型：评价 + 排序
- 推荐路线：数据审计 -> 指标同向化 -> 熵权法/AHP 赋权 -> TOPSIS 排序 -> 权重扰动敏感性分析
- 常用工具：`mathmodel-entropy-weight`、`mathmodel-topsis`、`mathmodel-validate-results`
- 高风险点：指标方向写反、权重没有来源、排名缺少稳定性检查、摘要只写最优方案没有解释原因

## 示例：需求预测

- 赛题类型：预测 + 策略建议
- 推荐路线：缺失/异常检查 -> 基线延续模型 -> 回归/GM(1,1)/指数平滑 -> 误差对比 -> 未来场景预测
- 常用工具：`mathmodel-regression`、`mathmodel-gm11`、`mathmodel-exponential-smoothing`
- 高风险点：用未来数据做特征、只报训练误差、没有说明预测区间、没有解释关键特征

## 示例：资源分配优化

- 赛题类型：优化
- 推荐路线：变量定义 -> 目标函数 -> 约束清单 -> 线性规划/整数规划 -> 结果可行性检查 -> 参数敏感性
- 常用工具：`mathmodel-linear-programming`、`mathmodel-pipeline validate`
- 高风险点：约束漏写、单位不统一、最优解不可执行、没有和人工规则比较

## 示例：不确定性仿真

- 赛题类型：仿真 + 风险评估
- 推荐路线：随机变量建模 -> 分布假设 -> 蒙特卡洛仿真 -> 分位数/可行率 -> 敏感性分析
- 常用工具：`mathmodel-monte-carlo`
- 高风险点：分布假设无依据、仿真次数太少、只报均值不报分位数、没有检查约束可行率
