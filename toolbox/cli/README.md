# mathmodel-toolkit CLI

`mathmodel-toolkit` 是自建数学工具箱的方法计算入口。当前已实现第一批评价、优化、预测和仿真基础方法：

- `topsis`：多指标评价与方案排序。
- `entropy-weight`：熵权法客观赋权与综合得分。
- `ahp`：层次分析法权重与一致性检验。
- `linear-programming`：小规模线性规划顶点枚举求解。
- `regression`：普通最小二乘线性回归。
- `gm11`：GM(1,1) 灰色预测。
- `exponential-smoothing`：一次指数平滑预测。
- `monte-carlo`：按 JSON 分布配置进行蒙特卡洛仿真。

## 统一输出

提供 `--output <dir>` 时，每个命令都会生成：

- `result.json`：机器可读结果。
- `result_card.md`：可写入项目黑板的结果卡。
- `figure_card.md`：可写入项目黑板的图表/表格卡。
- `rank_table.csv`、`weights_table.csv`、预测表、候选顶点表或样本表：论文可引用的表格数据。

## 示例

```bash
toolbox/cli/mathmodel-toolkit topsis \
  --input toolbox/examples/evaluation/topsis_sample.csv \
  --config toolbox/examples/evaluation/topsis_config.json \
  --output /tmp/mathmodel-toolkit-topsis
```

```bash
toolbox/cli/mathmodel-toolkit entropy-weight \
  --input toolbox/examples/evaluation/topsis_sample.csv \
  --config toolbox/examples/evaluation/topsis_config.json \
  --output /tmp/mathmodel-toolkit-entropy
```

```bash
toolbox/cli/mathmodel-toolkit ahp \
  --input toolbox/examples/evaluation/ahp_matrix.csv \
  --labels cost,quality,time \
  --output /tmp/mathmodel-toolkit-ahp
```

```bash
toolbox/cli/mathmodel-toolkit linear-programming \
  --config toolbox/examples/optimization/linear_programming_config.json \
  --output /tmp/mathmodel-toolkit-lp
```

```bash
toolbox/cli/mathmodel-toolkit regression \
  --input toolbox/examples/prediction/regression_sample.csv \
  --config toolbox/examples/prediction/regression_config.json \
  --output /tmp/mathmodel-toolkit-regression
```

```bash
toolbox/cli/mathmodel-toolkit gm11 \
  --input toolbox/examples/prediction/gm11_sample.csv \
  --value-column value \
  --forecast-steps 3 \
  --output /tmp/mathmodel-toolkit-gm11
```

```bash
toolbox/cli/mathmodel-toolkit exponential-smoothing \
  --input toolbox/examples/prediction/exponential_smoothing_sample.csv \
  --value-column demand \
  --forecast-steps 3 \
  --output /tmp/mathmodel-toolkit-exp
```

```bash
toolbox/cli/mathmodel-toolkit monte-carlo \
  --config toolbox/examples/simulation/monte_carlo_config.json \
  --iterations 5000 \
  --output /tmp/mathmodel-toolkit-mc
```

## 输入约定

- TOPSIS 与熵权法输入为带表头 CSV。
- 第一列若包含非数值，会自动作为方案 ID。
- `config.json` 可包含 `columns`、`directions`、`weights`。
- AHP 输入为纯数值方阵 CSV，`--labels` 可提供指标名。
- 线性规划输入为 JSON，需包含 `variables`、`objective`、`constraints`、`sense`。
- 回归输入为带表头 CSV，`config.json` 可包含 `target`、`features`、`id_column`。
- GM(1,1) 输入必须是正数序列；指数平滑输入为单变量数值序列。
- 蒙特卡洛输入为 JSON，需包含变量分布 `variables` 和目标表达式 `expression`。

## 后续扩展

后续方法继续复用同一输出协议，优先沉淀整数规划、多目标优化、ARIMA、PCA、DEA、排队论等高频方法。

# mathmodel-pipeline CLI

`mathmodel-pipeline` 是编程手和审核 agent 使用的执行/验证入口。

## run

运行 Python 脚本，生成持久化运行目录：

```bash
toolbox/cli/mathmodel-pipeline run \
  --workspace /tmp/mathmodel-pipeline-demo \
  --script toolbox/examples/pipeline/simple_model.py \
  --output /tmp/mathmodel-pipeline-demo/run-1
```

产物：

- `script.py`
- `notebook.ipynb`
- `execution.json`
- `result.json`
- `code_card.md`
- `result_card.md`
- `repro_command.sh`

## review

检查数据泄露、指标、边界条件、随机种子和图表一致性：

```bash
toolbox/cli/mathmodel-pipeline review \
  --run-dir /tmp/mathmodel-pipeline-demo/run-1 \
  --output /tmp/mathmodel-pipeline-demo/review \
  --allow-fail
```

产物：`review_report.json`、`review_card.md`。

## structure

把运行目录或 `result.json` 整理成论文素材：

```bash
toolbox/cli/mathmodel-pipeline structure \
  --input /tmp/mathmodel-pipeline-demo/run-1 \
  --output /tmp/mathmodel-pipeline-demo/structured \
  --problem P1
```

产物：`structured_result.json`、`result_card.md`、`figure_card.md`、`paper_material.md`。

## validate

做结果级边界、基线和敏感性检查：

```bash
toolbox/cli/mathmodel-pipeline validate \
  --result-json /tmp/mathmodel-pipeline-demo/run-1/result.json \
  --baseline-json /tmp/mathmodel-pipeline-demo/run-1/result.json \
  --output /tmp/mathmodel-pipeline-demo/validation
```

产物：`validation_report.json`、`review_card.md`。
