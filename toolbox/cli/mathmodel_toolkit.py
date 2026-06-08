#!/usr/bin/env python3
"""MathModelAI local modeling toolkit CLI.

The first stable commands focus on contest-friendly evaluation methods and
always emit result_card / figure_card artifacts for the project blackboard.
"""

from __future__ import annotations

import argparse
import ast
import csv
import itertools
import json
import math
import random
import statistics
from pathlib import Path
from typing import Any


RI_TABLE = {
    1: 0.0,
    2: 0.0,
    3: 0.58,
    4: 0.90,
    5: 1.12,
    6: 1.24,
    7: 1.32,
    8: 1.41,
    9: 1.45,
    10: 1.49,
}

SAFE_MATH_NAMES = {
    name: getattr(math, name)
    for name in dir(math)
    if not name.startswith("_")
}
SAFE_MATH_NAMES.update({"abs": abs, "min": min, "max": max, "pow": pow, "round": round})


class SafeExpression(ast.NodeVisitor):
    allowed_nodes = {
        ast.Expression,
        ast.BinOp,
        ast.UnaryOp,
        ast.Num,
        ast.Constant,
        ast.Name,
        ast.Load,
        ast.Add,
        ast.Sub,
        ast.Mult,
        ast.Div,
        ast.Pow,
        ast.Mod,
        ast.USub,
        ast.UAdd,
        ast.Call,
        ast.Compare,
        ast.BoolOp,
        ast.And,
        ast.Or,
        ast.Eq,
        ast.NotEq,
        ast.Lt,
        ast.LtE,
        ast.Gt,
        ast.GtE,
    }

    def __init__(self, names: set[str]) -> None:
        self.names = names

    def generic_visit(self, node: ast.AST) -> None:
        if type(node) not in self.allowed_nodes:
            raise ValueError(f"unsupported expression node: {type(node).__name__}")
        super().generic_visit(node)

    def visit_Name(self, node: ast.Name) -> None:
        if node.id not in self.names and node.id not in SAFE_MATH_NAMES:
            raise ValueError(f"unknown name in expression: {node.id}")

    def visit_Call(self, node: ast.Call) -> None:
        if not isinstance(node.func, ast.Name) or node.func.id not in SAFE_MATH_NAMES:
            raise ValueError("only math/simple functions are allowed in expressions")
        self.generic_visit(node)


def is_float(value: str) -> bool:
    try:
        float(value)
        return True
    except (TypeError, ValueError):
        return False


def read_json(path: str | None) -> dict[str, Any]:
    if not path:
        return {}
    return json.loads(Path(path).read_text(encoding="utf-8"))


def normalize_output_value(value: Any) -> Any:
    if isinstance(value, float):
        if math.isfinite(value):
            return round(value, 10)
        return str(value)
    if isinstance(value, list):
        return [normalize_output_value(v) for v in value]
    if isinstance(value, dict):
        return {k: normalize_output_value(v) for k, v in value.items()}
    return value


def read_csv_table(path: str) -> tuple[list[str], list[dict[str, str]]]:
    with Path(path).open("r", encoding="utf-8-sig", newline="") as f:
        reader = csv.DictReader(f)
        if not reader.fieldnames:
            raise ValueError(f"CSV has no header: {path}")
        return list(reader.fieldnames), list(reader)


def first_numeric_column(headers: list[str], rows: list[dict[str, str]]) -> str:
    for header in headers:
        if all(is_float(row.get(header, "")) for row in rows):
            return header
    raise ValueError("no numeric column found")


def column_values(path: str, column: str | None = None) -> tuple[str, list[float]]:
    headers, rows = read_csv_table(path)
    if not rows:
        raise ValueError("input CSV has no data rows")
    col = column or first_numeric_column(headers, rows)
    values = [float(row[col]) for row in rows if is_float(row.get(col, ""))]
    if len(values) < 3:
        raise ValueError("at least 3 numeric observations are required")
    return col, values


def select_numeric_matrix(
    path: str,
    config: dict[str, Any],
    id_column: str | None,
) -> tuple[list[str], list[str], list[list[float]]]:
    headers, rows = read_csv_table(path)
    if not rows:
        raise ValueError("input CSV has no data rows")

    if id_column is None:
        first = headers[0]
        if any(not is_float(r.get(first, "")) for r in rows):
            id_column = first

    configured_columns = config.get("columns")
    if configured_columns:
        columns = [str(c) for c in configured_columns]
    else:
        columns = [
            h for h in headers
            if h != id_column and all(is_float(r.get(h, "")) for r in rows)
        ]
    if not columns:
        raise ValueError("no numeric indicator columns found")

    ids = []
    matrix: list[list[float]] = []
    for idx, row in enumerate(rows, start=1):
        ids.append(row.get(id_column, str(idx)) if id_column else str(idx))
        matrix.append([float(row[c]) for c in columns])
    return ids, columns, matrix


def directions_for(columns: list[str], config: dict[str, Any]) -> list[str]:
    raw = config.get("directions", {})
    if isinstance(raw, list):
        directions = [str(v).lower() for v in raw]
    elif isinstance(raw, dict):
        directions = [str(raw.get(c, "positive")).lower() for c in columns]
    else:
        directions = ["positive"] * len(columns)
    out = []
    for d in directions:
        if d in {"positive", "benefit", "max", "+", "正向"}:
            out.append("positive")
        elif d in {"negative", "cost", "min", "-", "逆向", "负向"}:
            out.append("negative")
        else:
            raise ValueError(f"unknown indicator direction: {d}")
    return out


def weights_for(columns: list[str], config: dict[str, Any]) -> list[float]:
    raw = config.get("weights")
    if raw is None:
        return [1.0 / len(columns)] * len(columns)
    if isinstance(raw, list):
        weights = [float(v) for v in raw]
    elif isinstance(raw, dict):
        weights = [float(raw[c]) for c in columns]
    else:
        raise ValueError("weights must be a list or object")
    if len(weights) != len(columns):
        raise ValueError("weights length does not match indicator columns")
    total = sum(weights)
    if total <= 0:
        raise ValueError("weights sum must be positive")
    return [w / total for w in weights]


def transpose(matrix: list[list[float]]) -> list[list[float]]:
    return [list(col) for col in zip(*matrix)]


def solve_linear_system(matrix: list[list[float]], rhs: list[float]) -> list[float] | None:
    n = len(rhs)
    if not matrix or any(len(row) != n for row in matrix):
        raise ValueError("linear system must be square")
    aug = [list(row) + [rhs[i]] for i, row in enumerate(matrix)]
    for col in range(n):
        pivot = max(range(col, n), key=lambda r: abs(aug[r][col]))
        if math.isclose(aug[pivot][col], 0.0, abs_tol=1e-12):
            return None
        if pivot != col:
            aug[col], aug[pivot] = aug[pivot], aug[col]
        factor = aug[col][col]
        for j in range(col, n + 1):
            aug[col][j] /= factor
        for r in range(n):
            if r == col:
                continue
            f = aug[r][col]
            if math.isclose(f, 0.0, abs_tol=1e-12):
                continue
            for j in range(col, n + 1):
                aug[r][j] -= f * aug[col][j]
    return [aug[i][n] for i in range(n)]


def least_squares(x: list[list[float]], y: list[float]) -> list[float]:
    if not x or len(x) != len(y):
        raise ValueError("least squares input is empty or misaligned")
    cols = len(x[0])
    xtx = [[0.0 for _ in range(cols)] for _ in range(cols)]
    xty = [0.0 for _ in range(cols)]
    for row, target in zip(x, y):
        if len(row) != cols:
            raise ValueError("least squares matrix has inconsistent width")
        for i in range(cols):
            xty[i] += row[i] * target
            for j in range(cols):
                xtx[i][j] += row[i] * row[j]
    beta = solve_linear_system(xtx, xty)
    if beta is None:
        raise ValueError("normal equation is singular; remove collinear features or add more data")
    return beta


def minmax_or_zero(values: list[float], direction: str) -> list[float]:
    lo = min(values)
    hi = max(values)
    if math.isclose(hi, lo):
        return [0.0 for _ in values]
    if direction == "positive":
        return [(v - lo) / (hi - lo) for v in values]
    return [(hi - v) / (hi - lo) for v in values]


def vector_normalize(matrix: list[list[float]]) -> list[list[float]]:
    cols = transpose(matrix)
    norms = [math.sqrt(sum(v * v for v in col)) for col in cols]
    out = []
    for row in matrix:
        out.append([
            0.0 if math.isclose(norms[j], 0.0) else row[j] / norms[j]
            for j in range(len(row))
        ])
    return out


def weighted_rows(matrix: list[list[float]], weights: list[float]) -> list[list[float]]:
    return [[value * weights[j] for j, value in enumerate(row)] for row in matrix]


def write_csv(path: Path, headers: list[str], rows: list[list[Any]]) -> None:
    with path.open("w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(headers)
        writer.writerows(rows)


def emit_artifacts(
    output_dir: str | None,
    payload: dict[str, Any],
    table_name: str,
    table_headers: list[str],
    table_rows: list[list[Any]],
) -> dict[str, str]:
    if not output_dir:
        return {}

    out = Path(output_dir)
    out.mkdir(parents=True, exist_ok=True)
    result_path = out / "result.json"
    result_card_path = out / "result_card.md"
    figure_card_path = out / "figure_card.md"
    table_path = out / table_name

    write_csv(table_path, table_headers, table_rows)
    payload = dict(payload)
    payload["artifacts"] = {
        "result_json": str(result_path),
        "result_card": str(result_card_path),
        "figure_card": str(figure_card_path),
        "table": str(table_path),
    }
    result_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    result_card_path.write_text(render_result_card(payload), encoding="utf-8")
    figure_card_path.write_text(render_figure_card(payload, table_path), encoding="utf-8")
    return payload["artifacts"]


def render_result_card(payload: dict[str, Any]) -> str:
    method = payload["method"]
    summary = payload.get("summary", "")
    metrics = payload.get("metrics", {})
    artifacts = payload.get("artifacts", {})
    lines = [
        "## 结果摘要",
        f"- 方法: {method}",
        f"- 主要结论: {summary}",
        f"- 输出文件: {artifacts.get('result_json', '(stdout only)')}",
        "",
        "## 指标与误差",
    ]
    if metrics:
        for key, value in metrics.items():
            lines.append(f"- {key}: {value}")
    else:
        lines.append("- 暂无额外指标")
    lines.extend([
        "",
        "## 可解释性与限制",
        "- 解释: 结果由 CLI 按方法卡计算生成，需由建模手检查指标方向、权重来源和题意一致性。",
        "- 限制: CLI 只保证计算流程可复现，不替代建模假设审查。",
    ])
    return "\n".join(lines) + "\n"


def render_figure_card(payload: dict[str, Any], table_path: Path) -> str:
    return "\n".join([
        "## 图表信息",
        f"- 图表编号/标题: {payload['method']} 输出表",
        f"- 文件路径: {table_path}",
        "- 对应问题: <由父代理或论文手填写>",
        "",
        "## 数据来源",
        f"- 输入数据: {payload.get('input', '(unknown)')}",
        "- 生成脚本: toolbox/cli/mathmodel_toolkit.py",
        "- 关键参数: 见 result.json",
        "",
        "## 论文使用",
        f"- 结论句: {payload.get('summary', '<待论文手改写>')}",
        "- 需要检查: 指标方向、单位、权重来源、表格标题和排序稳定性",
        "",
    ])


def topsis(args: argparse.Namespace) -> dict[str, Any]:
    config = read_json(args.config)
    ids, columns, matrix = select_numeric_matrix(args.input, config, args.id_column)
    directions = directions_for(columns, config)
    weights = weights_for(columns, config)
    normalized = vector_normalize(matrix)
    weighted = weighted_rows(normalized, weights)
    cols = transpose(weighted)

    ideals = []
    anti_ideals = []
    for j, col in enumerate(cols):
        if directions[j] == "positive":
            ideals.append(max(col))
            anti_ideals.append(min(col))
        else:
            ideals.append(min(col))
            anti_ideals.append(max(col))

    rows = []
    scores = []
    for idx, row in enumerate(weighted):
        d_pos = math.sqrt(sum((row[j] - ideals[j]) ** 2 for j in range(len(columns))))
        d_neg = math.sqrt(sum((row[j] - anti_ideals[j]) ** 2 for j in range(len(columns))))
        score = 0.0 if math.isclose(d_pos + d_neg, 0.0) else d_neg / (d_pos + d_neg)
        scores.append((ids[idx], score, d_pos, d_neg))

    ranked = sorted(scores, key=lambda x: x[1], reverse=True)
    rank_by_id = {item[0]: rank for rank, item in enumerate(ranked, start=1)}
    for item_id, score, d_pos, d_neg in scores:
        rows.append([item_id, round(score, 10), rank_by_id[item_id], round(d_pos, 10), round(d_neg, 10)])

    payload = {
        "method": "topsis",
        "input": args.input,
        "columns": columns,
        "directions": directions,
        "weights": dict(zip(columns, weights)),
        "summary": f"最优方案为 {ranked[0][0]}，TOPSIS 得分 {ranked[0][1]:.6f}",
        "metrics": {"scheme_count": len(ids), "indicator_count": len(columns)},
        "results": [
            {"id": item_id, "score": score, "rank": rank_by_id[item_id], "d_positive": d_pos, "d_negative": d_neg}
            for item_id, score, d_pos, d_neg in scores
        ],
    }
    emit_artifacts(args.output, payload, "rank_table.csv", ["id", "score", "rank", "d_positive", "d_negative"], rows)
    return payload


def entropy_weight(args: argparse.Namespace) -> dict[str, Any]:
    config = read_json(args.config)
    ids, columns, matrix = select_numeric_matrix(args.input, config, args.id_column)
    directions = directions_for(columns, config)
    normalized_cols = []
    for col, direction in zip(transpose(matrix), directions):
        normalized_cols.append(minmax_or_zero(col, direction))
    normalized = transpose(normalized_cols)

    n = len(ids)
    k = 0.0 if n <= 1 else 1.0 / math.log(n)
    entropy = []
    for col in normalized_cols:
        total = sum(col)
        if math.isclose(total, 0.0):
            entropy.append(1.0)
            continue
        e = 0.0
        for value in col:
            p = value / total
            if p > 0:
                e += p * math.log(p)
        entropy.append(-k * e)
    diversity = [1 - e for e in entropy]
    diversity_sum = sum(diversity)
    weights = [1.0 / len(columns)] * len(columns) if math.isclose(diversity_sum, 0.0) else [d / diversity_sum for d in diversity]

    score_rows = []
    for item_id, row in zip(ids, normalized):
        score = sum(row[j] * weights[j] for j in range(len(columns)))
        score_rows.append((item_id, score))
    ranked = sorted(score_rows, key=lambda x: x[1], reverse=True)
    rank_by_id = {item[0]: rank for rank, item in enumerate(ranked, start=1)}
    rows = [[item_id, round(score, 10), rank_by_id[item_id]] for item_id, score in score_rows]

    payload = {
        "method": "entropy-weight",
        "input": args.input,
        "columns": columns,
        "directions": directions,
        "summary": f"最大综合得分方案为 {ranked[0][0]}，得分 {ranked[0][1]:.6f}",
        "metrics": {"scheme_count": len(ids), "indicator_count": len(columns)},
        "weights": dict(zip(columns, weights)),
        "entropy": dict(zip(columns, entropy)),
        "results": [
            {"id": item_id, "score": score, "rank": rank_by_id[item_id]}
            for item_id, score in score_rows
        ],
    }
    emit_artifacts(args.output, payload, "rank_table.csv", ["id", "score", "rank"], rows)
    return payload


def read_matrix_csv(path: str) -> list[list[float]]:
    rows: list[list[float]] = []
    with Path(path).open("r", encoding="utf-8-sig", newline="") as f:
        reader = csv.reader(f)
        for row in reader:
            if not row:
                continue
            if all(is_float(v) for v in row):
                rows.append([float(v) for v in row])
            elif rows:
                raise ValueError("AHP matrix contains nonnumeric row after numeric data")
    if not rows or any(len(r) != len(rows) for r in rows):
        raise ValueError("AHP input must be a square numeric matrix")
    return rows


def ahp(args: argparse.Namespace) -> dict[str, Any]:
    matrix = read_matrix_csv(args.input)
    n = len(matrix)
    geo = [math.prod(row) ** (1.0 / n) for row in matrix]
    total = sum(geo)
    weights = [v / total for v in geo]
    aw = [sum(matrix[i][j] * weights[j] for j in range(n)) for i in range(n)]
    lambda_max = sum(aw[i] / weights[i] for i in range(n)) / n
    ci = 0.0 if n <= 2 else (lambda_max - n) / (n - 1)
    ri = RI_TABLE.get(n)
    cr = 0.0 if not ri else ci / ri
    labels = args.labels.split(",") if args.labels else [f"c{i}" for i in range(1, n + 1)]
    if len(labels) != n:
        raise ValueError("labels count must match matrix size")

    rows = [[label, round(weight, 10)] for label, weight in zip(labels, weights)]
    payload = {
        "method": "ahp",
        "input": args.input,
        "labels": labels,
        "summary": f"最大权重指标为 {labels[weights.index(max(weights))]}，权重 {max(weights):.6f}，CR={cr:.6f}",
        "metrics": {"lambda_max": lambda_max, "ci": ci, "ri": ri, "cr": cr, "consistent": cr < args.cr_threshold},
        "weights": dict(zip(labels, weights)),
    }
    emit_artifacts(args.output, payload, "weights_table.csv", ["indicator", "weight"], rows)
    return payload


def coefficients_from_config(raw: Any, variables: list[str]) -> list[float]:
    if isinstance(raw, list):
        coeffs = [float(v) for v in raw]
    elif isinstance(raw, dict):
        coeffs = [float(raw.get(v, 0.0)) for v in variables]
    else:
        raise ValueError("coefficients/objective must be a list or object")
    if len(coeffs) != len(variables):
        raise ValueError("coefficient count does not match variables")
    return coeffs


def linear_programming(args: argparse.Namespace) -> dict[str, Any]:
    config = read_json(args.config)
    variables = [str(v) for v in config.get("variables", [])]
    if not variables:
        objective_raw = config.get("objective")
        if isinstance(objective_raw, dict):
            variables = list(objective_raw.keys())
        else:
            raise ValueError("linear-programming config requires variables")
    objective = coefficients_from_config(config.get("objective"), variables)
    sense = str(config.get("sense", "max")).lower()
    if sense not in {"max", "min"}:
        raise ValueError("sense must be max or min")

    constraints = []
    for idx, c in enumerate(config.get("constraints", []), start=1):
        coeffs = coefficients_from_config(c.get("coefficients", c.get("lhs")), variables)
        op = str(c.get("op", "<=")).strip()
        rhs = float(c.get("rhs"))
        name = str(c.get("name", f"c{idx}"))
        if op in {"<=", "le", "≤"}:
            constraints.append({"name": name, "coefficients": coeffs, "rhs": rhs})
        elif op in {">=", "ge", "≥"}:
            constraints.append({"name": name, "coefficients": [-v for v in coeffs], "rhs": -rhs})
        elif op in {"=", "==", "eq"}:
            constraints.append({"name": name, "coefficients": coeffs, "rhs": rhs})
            constraints.append({"name": f"{name}_reverse", "coefficients": [-v for v in coeffs], "rhs": -rhs})
        else:
            raise ValueError(f"unsupported constraint operator: {op}")

    if config.get("nonnegative", True):
        for i, v in enumerate(variables):
            coeffs = [0.0] * len(variables)
            coeffs[i] = -1.0
            constraints.append({"name": f"{v}_nonnegative", "coefficients": coeffs, "rhs": 0.0})

    n = len(variables)
    if len(constraints) < n:
        raise ValueError("not enough constraints to determine candidate vertices")
    if len(constraints) > args.max_constraints:
        raise ValueError(f"too many constraints for enumeration; limit is {args.max_constraints}")

    candidates: list[dict[str, Any]] = []
    for combo in itertools.combinations(range(len(constraints)), n):
        matrix = [constraints[i]["coefficients"] for i in combo]
        rhs = [constraints[i]["rhs"] for i in combo]
        sol = solve_linear_system(matrix, rhs)
        if sol is None:
            continue
        feasible = all(
            sum(c["coefficients"][i] * sol[i] for i in range(n)) <= c["rhs"] + args.tolerance
            for c in constraints
        )
        if not feasible:
            continue
        value = sum(objective[i] * sol[i] for i in range(n))
        candidates.append({
            "variables": dict(zip(variables, sol)),
            "objective_value": value,
            "active_constraints": [constraints[i]["name"] for i in combo],
        })

    if not candidates:
        raise ValueError("no feasible vertex found; check constraints or equality handling")
    reverse = sense == "max"
    ranked = sorted(candidates, key=lambda item: item["objective_value"], reverse=reverse)
    best = ranked[0]
    rows = [
        [
            rank,
            round(item["objective_value"], 10),
            json.dumps(normalize_output_value(item["variables"]), ensure_ascii=False),
            ";".join(item["active_constraints"]),
        ]
        for rank, item in enumerate(ranked[: args.keep_candidates], start=1)
    ]
    payload = {
        "method": "linear-programming",
        "input": args.config,
        "variables": variables,
        "sense": sense,
        "summary": f"最优目标值为 {best['objective_value']:.6f}，变量为 {normalize_output_value(best['variables'])}",
        "metrics": {
            "variable_count": len(variables),
            "constraint_count": len(config.get("constraints", [])),
            "candidate_count": len(candidates),
        },
        "objective": dict(zip(variables, objective)),
        "best": normalize_output_value(best),
        "candidates": normalize_output_value(ranked[: args.keep_candidates]),
    }
    emit_artifacts(args.output, payload, "lp_candidates.csv", ["rank", "objective_value", "variables", "active_constraints"], rows)
    return payload


def regression(args: argparse.Namespace) -> dict[str, Any]:
    config = read_json(args.config)
    headers, rows_raw = read_csv_table(args.input)
    target = args.target or config.get("target")
    if not target:
        target = headers[-1]
    features = args.features.split(",") if args.features else config.get("features")
    if not features:
        features = [h for h in headers if h != target and all(is_float(r.get(h, "")) for r in rows_raw)]
    features = [str(f) for f in features]
    if target not in headers:
        raise ValueError(f"target column not found: {target}")
    if not features:
        raise ValueError("no numeric feature columns found")

    x: list[list[float]] = []
    y: list[float] = []
    ids: list[str] = []
    id_column = args.id_column or config.get("id_column")
    for idx, row in enumerate(rows_raw, start=1):
        if not is_float(row.get(target, "")) or any(not is_float(row.get(f, "")) for f in features):
            continue
        ids.append(row.get(id_column, str(idx)) if id_column else str(idx))
        values = [float(row[f]) for f in features]
        x.append(([1.0] if args.intercept else []) + values)
        y.append(float(row[target]))
    if len(y) <= len(features):
        raise ValueError("not enough valid rows for regression")

    beta = least_squares(x, y)
    y_mean = sum(y) / len(y)
    predictions = [sum(beta[j] * row[j] for j in range(len(beta))) for row in x]
    residuals = [actual - pred for actual, pred in zip(y, predictions)]
    rmse = math.sqrt(sum(r * r for r in residuals) / len(residuals))
    mae = sum(abs(r) for r in residuals) / len(residuals)
    sst = sum((actual - y_mean) ** 2 for actual in y)
    sse = sum(r * r for r in residuals)
    r2 = 1.0 if math.isclose(sst, 0.0) else 1.0 - sse / sst
    coefficient_names = (["intercept"] if args.intercept else []) + features
    table_rows = [
        [ids[i], round(y[i], 10), round(predictions[i], 10), round(residuals[i], 10)]
        for i in range(len(y))
    ]
    payload = {
        "method": "regression",
        "input": args.input,
        "target": target,
        "features": features,
        "summary": f"线性回归完成，RMSE={rmse:.6f}，R2={r2:.6f}",
        "metrics": {"rows": len(y), "rmse": rmse, "mae": mae, "r2": r2},
        "coefficients": dict(zip(coefficient_names, beta)),
        "results": [
            {"id": ids[i], "actual": y[i], "prediction": predictions[i], "residual": residuals[i]}
            for i in range(len(y))
        ],
    }
    emit_artifacts(args.output, payload, "regression_predictions.csv", ["id", "actual", "prediction", "residual"], table_rows)
    return payload


def gm11(args: argparse.Namespace) -> dict[str, Any]:
    column, values = column_values(args.input, args.value_column)
    if any(v <= 0 for v in values):
        raise ValueError("GM(1,1) requires positive observations")
    horizon = args.forecast_steps
    x1 = list(itertools.accumulate(values))
    z1 = [0.5 * (x1[k] + x1[k - 1]) for k in range(1, len(x1))]
    design = [[-z, 1.0] for z in z1]
    params = least_squares(design, values[1:])
    a, b = params
    if math.isclose(a, 0.0, abs_tol=1e-12):
        raise ValueError("GM(1,1) development coefficient is too close to zero")
    x1_hat = [
        (values[0] - b / a) * math.exp(-a * k) + b / a
        for k in range(0, len(values) + horizon)
    ]
    fitted = [values[0]]
    for k in range(1, len(x1_hat)):
        fitted.append(x1_hat[k] - x1_hat[k - 1])
    fitted_known = fitted[: len(values)]
    residuals = [values[i] - fitted_known[i] for i in range(len(values))]
    rmse = math.sqrt(sum(r * r for r in residuals) / len(residuals))
    forecast = fitted[len(values):]
    rows = []
    for i, actual in enumerate(values, start=1):
        rows.append([i, "fitted", round(actual, 10), round(fitted_known[i - 1], 10), round(actual - fitted_known[i - 1], 10)])
    for j, value in enumerate(forecast, start=1):
        rows.append([len(values) + j, "forecast", "", round(value, 10), ""])
    payload = {
        "method": "gm11",
        "input": args.input,
        "value_column": column,
        "summary": f"GM(1,1) 完成，未来 {horizon} 期预测为 {[round(v, 6) for v in forecast]}",
        "metrics": {"rows": len(values), "forecast_steps": horizon, "rmse": rmse, "a": a, "b": b},
        "fitted": fitted_known,
        "forecast": forecast,
        "results": [{"period": i + 1, "actual": values[i], "fitted": fitted_known[i], "residual": residuals[i]} for i in range(len(values))],
    }
    emit_artifacts(args.output, payload, "gm11_forecast.csv", ["period", "kind", "actual", "value", "residual"], rows)
    return payload


def exponential_smoothing(args: argparse.Namespace) -> dict[str, Any]:
    column, values = column_values(args.input, args.value_column)
    horizon = args.forecast_steps

    def fit(alpha: float) -> tuple[list[float], float]:
        level = values[0]
        fitted = [level]
        errors = []
        for actual in values[1:]:
            fitted.append(level)
            errors.append(actual - level)
            level = alpha * actual + (1 - alpha) * level
        sse = sum(e * e for e in errors)
        return fitted, sse

    if args.alpha is None:
        candidates = [i / 100 for i in range(1, 100)]
        alpha = min(candidates, key=lambda a: fit(a)[1])
    else:
        alpha = args.alpha
    if not 0 < alpha < 1:
        raise ValueError("alpha must be between 0 and 1")
    fitted, sse = fit(alpha)
    residuals = [values[i] - fitted[i] for i in range(1, len(values))]
    rmse = math.sqrt(sse / max(1, len(residuals)))
    level = values[0]
    for actual in values[1:]:
        level = alpha * actual + (1 - alpha) * level
    forecast = [level for _ in range(horizon)]
    rows = []
    for i, actual in enumerate(values, start=1):
        residual = "" if i == 1 else round(actual - fitted[i - 1], 10)
        rows.append([i, "fitted", round(actual, 10), round(fitted[i - 1], 10), residual])
    for j, value in enumerate(forecast, start=1):
        rows.append([len(values) + j, "forecast", "", round(value, 10), ""])
    payload = {
        "method": "exponential-smoothing",
        "input": args.input,
        "value_column": column,
        "summary": f"一次指数平滑完成，alpha={alpha:.2f}，未来 {horizon} 期预测值 {level:.6f}",
        "metrics": {"rows": len(values), "forecast_steps": horizon, "alpha": alpha, "rmse": rmse},
        "fitted": fitted,
        "forecast": forecast,
    }
    emit_artifacts(args.output, payload, "exponential_smoothing_forecast.csv", ["period", "kind", "actual", "value", "residual"], rows)
    return payload


def eval_safe_expression(expression: str, values: dict[str, float]) -> Any:
    tree = ast.parse(expression, mode="eval")
    SafeExpression(set(values.keys())).visit(tree)
    compiled = compile(tree, "<mathmodel-expression>", "eval")
    return eval(compiled, {"__builtins__": {}}, {**SAFE_MATH_NAMES, **values})


def sample_variable(spec: dict[str, Any], rng: random.Random) -> float:
    dist = str(spec.get("dist", "uniform")).lower()
    if dist == "uniform":
        return rng.uniform(float(spec.get("low", 0.0)), float(spec.get("high", 1.0)))
    if dist == "normal":
        return rng.gauss(float(spec.get("mean", 0.0)), float(spec.get("sd", spec.get("std", 1.0))))
    if dist == "triangular":
        return rng.triangular(float(spec.get("low", 0.0)), float(spec.get("high", 1.0)), float(spec.get("mode", 0.5)))
    if dist == "randint":
        return float(rng.randint(int(spec.get("low", 0)), int(spec.get("high", 1))))
    if dist == "choice":
        choices = spec.get("values", [])
        if not choices:
            raise ValueError("choice distribution requires values")
        return float(rng.choice(choices))
    raise ValueError(f"unsupported distribution: {dist}")


def quantile(values: list[float], q: float) -> float:
    if not values:
        raise ValueError("cannot compute quantile for empty list")
    ordered = sorted(values)
    pos = (len(ordered) - 1) * q
    lo = math.floor(pos)
    hi = math.ceil(pos)
    if lo == hi:
        return ordered[int(pos)]
    return ordered[lo] * (hi - pos) + ordered[hi] * (pos - lo)


def monte_carlo(args: argparse.Namespace) -> dict[str, Any]:
    config = read_json(args.config)
    variables = config.get("variables", {})
    if not isinstance(variables, dict) or not variables:
        raise ValueError("monte-carlo config requires variables object")
    expression = args.expression or config.get("expression")
    if not expression:
        raise ValueError("monte-carlo requires expression")
    iterations = args.iterations or int(config.get("iterations", 10000))
    seed = args.seed if args.seed is not None else config.get("seed", 42)
    rng = random.Random(seed)
    constraints = [str(c) for c in config.get("constraints", [])]
    threshold = config.get("threshold")

    values_out: list[float] = []
    feasible = 0
    sample_rows = []
    for i in range(iterations):
        sample = {name: sample_variable(spec, rng) for name, spec in variables.items()}
        result = eval_safe_expression(str(expression), sample)
        result_value = float(result)
        values_out.append(result_value)
        if constraints:
            ok = all(bool(eval_safe_expression(c, sample)) for c in constraints)
        else:
            ok = True
        if ok:
            feasible += 1
        if i < args.keep_samples:
            sample_rows.append([
                i + 1,
                round(result_value, 10),
                json.dumps(normalize_output_value(sample), ensure_ascii=False),
                ok,
            ])

    mean = statistics.fmean(values_out)
    stdev = statistics.pstdev(values_out) if len(values_out) > 1 else 0.0
    metrics = {
        "iterations": iterations,
        "mean": mean,
        "stdev": stdev,
        "min": min(values_out),
        "max": max(values_out),
        "p05": quantile(values_out, 0.05),
        "p50": quantile(values_out, 0.50),
        "p95": quantile(values_out, 0.95),
        "feasible_rate": feasible / iterations,
    }
    if threshold is not None:
        threshold_f = float(threshold)
        metrics["probability_ge_threshold"] = sum(1 for v in values_out if v >= threshold_f) / iterations
    payload = {
        "method": "monte-carlo",
        "input": args.config,
        "expression": expression,
        "summary": f"蒙特卡洛仿真完成，均值 {mean:.6f}，P5/P50/P95={metrics['p05']:.6f}/{metrics['p50']:.6f}/{metrics['p95']:.6f}",
        "metrics": metrics,
        "variables": variables,
        "constraints": constraints,
        "seed": seed,
    }
    emit_artifacts(args.output, payload, "monte_carlo_samples.csv", ["sample", "value", "variables", "feasible"], sample_rows)
    return payload


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="mathmodel-toolkit")
    sub = parser.add_subparsers(dest="command", required=True)

    def add_table_args(p: argparse.ArgumentParser) -> None:
        p.add_argument("--input", required=True, help="CSV input with header")
        p.add_argument("--config", help="JSON config with columns, directions and optional weights")
        p.add_argument("--id-column", help="scheme id column; inferred from the first nonnumeric column when omitted")
        p.add_argument("--output", help="artifact output directory")

    p_topsis = sub.add_parser("topsis", help="Run TOPSIS ranking")
    add_table_args(p_topsis)
    p_topsis.set_defaults(func=topsis)

    p_entropy = sub.add_parser("entropy-weight", help="Run entropy-weight ranking")
    add_table_args(p_entropy)
    p_entropy.set_defaults(func=entropy_weight)

    p_ahp = sub.add_parser("ahp", help="Compute AHP weights")
    p_ahp.add_argument("--input", required=True, help="square numeric CSV matrix")
    p_ahp.add_argument("--labels", help="comma-separated indicator names")
    p_ahp.add_argument("--cr-threshold", type=float, default=0.1)
    p_ahp.add_argument("--output", help="artifact output directory")
    p_ahp.set_defaults(func=ahp)

    p_lp = sub.add_parser("linear-programming", help="Solve small linear programming problems by vertex enumeration")
    p_lp.add_argument("--config", required=True, help="JSON config with variables, objective and constraints")
    p_lp.add_argument("--output", help="artifact output directory")
    p_lp.add_argument("--tolerance", type=float, default=1e-8)
    p_lp.add_argument("--max-constraints", type=int, default=24)
    p_lp.add_argument("--keep-candidates", type=int, default=20)
    p_lp.set_defaults(func=linear_programming)

    p_reg = sub.add_parser("regression", help="Run ordinary least squares regression")
    p_reg.add_argument("--input", required=True, help="CSV input with header")
    p_reg.add_argument("--config", help="optional JSON config with target, features and id_column")
    p_reg.add_argument("--target", help="target column; defaults to config target or last column")
    p_reg.add_argument("--features", help="comma-separated feature columns; inferred from numeric columns when omitted")
    p_reg.add_argument("--id-column", help="optional row id column")
    p_reg.add_argument("--no-intercept", dest="intercept", action="store_false", help="fit without intercept")
    p_reg.add_argument("--output", help="artifact output directory")
    p_reg.set_defaults(func=regression, intercept=True)

    p_gm = sub.add_parser("gm11", help="Run GM(1,1) grey prediction")
    p_gm.add_argument("--input", required=True, help="CSV input with a positive numeric sequence")
    p_gm.add_argument("--value-column", help="numeric value column; inferred when omitted")
    p_gm.add_argument("--forecast-steps", type=int, default=3)
    p_gm.add_argument("--output", help="artifact output directory")
    p_gm.set_defaults(func=gm11)

    p_es = sub.add_parser("exponential-smoothing", help="Run single exponential smoothing")
    p_es.add_argument("--input", required=True, help="CSV input with a numeric sequence")
    p_es.add_argument("--value-column", help="numeric value column; inferred when omitted")
    p_es.add_argument("--alpha", type=float, help="smoothing factor; grid-searched when omitted")
    p_es.add_argument("--forecast-steps", type=int, default=3)
    p_es.add_argument("--output", help="artifact output directory")
    p_es.set_defaults(func=exponential_smoothing)

    p_mc = sub.add_parser("monte-carlo", help="Run Monte Carlo simulation from JSON config")
    p_mc.add_argument("--config", required=True, help="JSON config with variables and expression")
    p_mc.add_argument("--expression", help="override objective expression")
    p_mc.add_argument("--iterations", type=int, help="override iteration count")
    p_mc.add_argument("--seed", type=int, help="override random seed")
    p_mc.add_argument("--keep-samples", type=int, default=200)
    p_mc.add_argument("--output", help="artifact output directory")
    p_mc.set_defaults(func=monte_carlo)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    payload = args.func(args)
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
