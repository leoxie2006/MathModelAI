# 常用数学建模方法卡索引

本索引用于让 Agent 在知识库中快速定位自建工具箱的方法卡。完整方法卡以 `toolbox/methods/` 为准，避免双份内容漂移。

## 优化建模

- 线性规划：`toolbox/methods/optimization/linear-programming.yaml`，适合线性资源分配、运输、排产。
- 整数规划：`toolbox/methods/optimization/integer-programming.yaml`，适合 0-1 选择、任务分配、设施启用。
- 非线性规划：`toolbox/methods/optimization/nonlinear-programming.yaml`，适合连续非线性目标或约束。
- 多目标优化：`toolbox/methods/optimization/multi-objective-optimization.yaml`，适合成本、收益、风险等冲突目标折中。
- 遗传算法：`toolbox/methods/optimization/genetic-algorithm.yaml`，适合复杂组合搜索和不可导目标。
- 模拟退火：`toolbox/methods/optimization/simulated-annealing.yaml`，适合需要跳出局部最优的单解搜索。

## 预测模型

- 回归模型：`toolbox/methods/prediction/regression.yaml`，适合连续值预测和变量解释。
- XGBoost：`toolbox/methods/prediction/xgboost.yaml`，适合表格数据非线性预测。
- ARIMA：`toolbox/methods/prediction/arima.yaml`，适合单变量平稳或差分平稳时间序列。
- 指数平滑：`toolbox/methods/prediction/exponential-smoothing.yaml`，适合短序列快速预测基线。
- GM(1,1)：`toolbox/methods/prediction/gm11.yaml`，适合小样本单调趋势短期预测。

## 评价模型

- AHP：`toolbox/methods/evaluation/ahp.yaml`，适合专家判断权重和层次结构评价。
- 熵权法：`toolbox/methods/evaluation/entropy-weight.yaml`，适合基于指标离散度的客观赋权。
- TOPSIS：`toolbox/methods/evaluation/topsis.yaml`，适合多指标方案排序。
- 模糊综合评价：`toolbox/methods/evaluation/fuzzy-comprehensive-evaluation.yaml`，适合边界模糊的等级评价。
- PCA：`toolbox/methods/evaluation/pca.yaml`，适合降维和综合指标构造。
- DEA：`toolbox/methods/evaluation/dea.yaml`，适合多投入多产出效率评价。

## 仿真模型

- 蒙特卡洛模拟：`toolbox/methods/simulation/monte-carlo.yaml`，适合不确定性、风险和概率估计。
- 排队论：`toolbox/methods/simulation/queuing-theory.yaml`，适合服务系统等待时间和容量分析。
- 元胞自动机：`toolbox/methods/simulation/cellular-automata.yaml`，适合空间网格局部规则演化。
- 系统动力学：`toolbox/methods/simulation/system-dynamics.yaml`，适合多反馈变量随时间演化。

## 使用约定

1. 先按问题类型检索本索引。
2. 再读取对应 `toolbox/methods/<category>/<id>.yaml` 的完整方法卡。
3. 调用未来 CLI/MCP 前必须检查方法卡中的 `assumptions`、`validation` 和 `pitfalls`。
4. 工具输出应转成 `result_card`、`figure_card` 和必要的 `review_card`。

## 写作与复盘素材

- 论文结构、摘要与图表 caption 模板：`knowledge_base/modeling/paper-writing-templates.md`。
- 常见扣分点与终审清单：`knowledge_base/modeling/scoring-pitfalls.md`。
- 案例复盘库模板与示例路线：`knowledge_base/modeling/case-review-library.md`。
