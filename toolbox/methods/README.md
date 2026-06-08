# 方法模板库

这里预留给自建数学建模方法工具箱。每个方法建议同时沉淀两类资产：

1. 方法卡：说明适用条件、输入输出、假设、验证方式和常见错误。
2. 可执行模板：稳定后做成 CLI，再通过 MCP 暴露给 Agent 调用。

## 已建分类与方法卡

- `optimization/`：线性规划、整数规划、非线性规划、多目标优化、遗传算法、模拟退火。
- `prediction/`：回归、XGBoost、ARIMA、指数平滑、GM(1,1)。
- `evaluation/`：AHP、熵权法、TOPSIS、模糊综合评价、PCA、DEA。
- `simulation/`：蒙特卡洛、排队论、元胞自动机、系统动力学。

## 质量检查

新增或修改方法卡后运行：

```bash
python3 toolbox/scripts/check_method_cards.py
```

检查脚本会确认 TODO 中列出的 21 张方法卡全部存在，并包含 `applies_when`、`inputs`、`outputs`、`assumptions`、`validation`、`pitfalls`、`future_cli` 等关键字段。

## 方法卡模板

```yaml
id: topsis
category: evaluation
applies_when: 多指标评价、方案排序
inputs:
  - 指标矩阵
  - 指标方向
  - 权重
outputs:
  - 综合得分
  - 排名
  - 敏感性分析
assumptions:
  - 指标可同向化
  - 方案之间可比较
validation:
  - 权重扰动
  - 指标方向检查
  - 极端样本检查
pitfalls:
  - 指标方向写反
  - 权重无来源
  - 量纲未处理
future_cli: mathmodel-toolkit topsis --input data.xlsx --config config.yaml
```
