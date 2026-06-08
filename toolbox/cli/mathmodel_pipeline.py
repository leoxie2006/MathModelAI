#!/usr/bin/env python3
"""MathModelAI coding and validation pipeline CLI.

This CLI keeps modeling runs reproducible and converts raw code outputs into
blackboard-ready cards for the modeling, coding, and paper agents.
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import os
import re
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path
from typing import Any


def now_id() -> str:
    return datetime.now().strftime("%Y%m%d-%H%M%S")


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def write_card(path: Path, title: str, sections: list[tuple[str, list[str]]]) -> None:
    lines = [f"## {title}"]
    for heading, items in sections:
        lines.extend(["", f"## {heading}"])
        if items:
            lines.extend(items)
        else:
            lines.append("- 无")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def make_notebook(script: str, stdout: str, stderr: str, returncode: int) -> dict[str, Any]:
    output_text = stdout
    if stderr:
        output_text += ("\n" if output_text else "") + "[stderr]\n" + stderr
    return {
        "cells": [
            {
                "cell_type": "code",
                "execution_count": 1,
                "metadata": {},
                "outputs": [
                    {
                        "name": "stdout" if returncode == 0 else "stderr",
                        "output_type": "stream",
                        "text": output_text.splitlines(True),
                    }
                ],
                "source": script.splitlines(True),
            }
        ],
        "metadata": {
            "kernelspec": {"display_name": "Python 3", "language": "python", "name": "python3"},
            "language_info": {"name": "python", "pygments_lexer": "ipython3"},
        },
        "nbformat": 4,
        "nbformat_minor": 5,
    }


def list_artifacts(run_dir: Path) -> list[str]:
    ignored = {
        "script.py",
        "notebook.ipynb",
        "execution.json",
        "code_card.md",
        "result_card.md",
        "repro_command.sh",
    }
    out = []
    for path in sorted(run_dir.iterdir()):
        if path.is_file() and path.name not in ignored:
            out.append(str(path))
    return out


def append_run_manifest(workspace: Path, run: dict[str, Any]) -> None:
    state_dir = workspace / ".mathmodel_exec"
    state_dir.mkdir(parents=True, exist_ok=True)
    manifest = state_dir / "runs.json"
    if manifest.exists():
        data = json.loads(manifest.read_text(encoding="utf-8"))
    else:
        data = {"runs": []}
    data["runs"].append(run)
    write_json(manifest, data)


def cmd_run(args: argparse.Namespace) -> int:
    workspace = Path(args.workspace).resolve()
    workspace.mkdir(parents=True, exist_ok=True)
    run_id = args.run_id or now_id()
    run_dir = Path(args.output or (workspace / "runs" / run_id)).resolve()
    run_dir.mkdir(parents=True, exist_ok=True)

    if args.script:
        source_path = Path(args.script).resolve()
        script = read_text(source_path)
    elif args.code:
        source_path = None
        script = args.code
    else:
        raise SystemExit("--script or --code is required")

    script_path = run_dir / "script.py"
    script_path.write_text(script, encoding="utf-8")
    cwd = Path(args.cwd).resolve() if args.cwd else run_dir
    cwd.mkdir(parents=True, exist_ok=True)
    start = time.time()
    proc = subprocess.run(
        [sys.executable, str(script_path), *args.script_args],
        cwd=str(cwd),
        text=True,
        capture_output=True,
        timeout=args.timeout,
        check=False,
    )
    duration = time.time() - start

    notebook = make_notebook(script, proc.stdout, proc.stderr, proc.returncode)
    write_json(run_dir / "notebook.ipynb", notebook)

    repro = f"cd {cwd}\n{sys.executable} {script_path} {' '.join(args.script_args)}\n"
    (run_dir / "repro_command.sh").write_text(repro, encoding="utf-8")
    os.chmod(run_dir / "repro_command.sh", 0o755)

    execution = {
        "run_id": run_id,
        "status": "success" if proc.returncode == 0 else "failed",
        "returncode": proc.returncode,
        "workspace": str(workspace),
        "run_dir": str(run_dir),
        "cwd": str(cwd),
        "source_script": str(source_path) if source_path else "<inline>",
        "script": str(script_path),
        "notebook": str(run_dir / "notebook.ipynb"),
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "duration_seconds": duration,
        "artifacts": list_artifacts(run_dir),
        "repro_command": str(run_dir / "repro_command.sh"),
    }
    write_json(run_dir / "execution.json", execution)
    write_json(run_dir / "result.json", execution)
    write_card(
        run_dir / "code_card.md",
        "实现入口",
        [
            ("脚本与命令", [f"- 脚本: {script_path}", f"- Notebook: {run_dir / 'notebook.ipynb'}", f"- 复现命令: {run_dir / 'repro_command.sh'}"]),
            ("运行状态", [f"- 状态: {execution['status']}", f"- 返回码: {proc.returncode}", f"- 用时: {duration:.3f}s"]),
            ("输出", [f"- result.json: {run_dir / 'result.json'}", f"- artifacts: {len(execution['artifacts'])} 个"]),
        ],
    )
    write_card(
        run_dir / "result_card.md",
        "结果摘要",
        [
            ("执行结果", [f"- 状态: {execution['status']}", f"- stdout 摘要: {proc.stdout.strip()[:300] or '无'}"]),
            ("可复现性", [f"- Notebook: {run_dir / 'notebook.ipynb'}", f"- 复现命令: {run_dir / 'repro_command.sh'}"]),
            ("限制", ["- 该卡片只记录执行结果，仍需建模手检查题意、约束、单位和评价指标。"]),
        ],
    )
    append_run_manifest(workspace, execution)
    print(json.dumps(execution, ensure_ascii=False, indent=2))
    return proc.returncode


def detect_review_findings(script_text: str, result: dict[str, Any] | None, run_dir: Path | None) -> list[dict[str, str]]:
    findings: list[dict[str, str]] = []
    lower = script_text.lower()
    if re.search(r"fit_transform\(", lower) and re.search(r"train_test_split|test_size|x_test|y_test", lower):
        findings.append({
            "severity": "high",
            "category": "data_leakage",
            "message": "脚本同时出现 fit_transform 和 train/test 划分，请确认标准化/降维是否只在训练集 fit。",
        })
    if ("random" in lower or "numpy" in lower or "sklearn" in lower) and not re.search(r"seed|random_state", lower):
        findings.append({
            "severity": "medium",
            "category": "reproducibility",
            "message": "脚本使用随机相关库但未发现 seed/random_state。",
        })
    if not re.search(r"assert|constraint|bounds|clip|约束|边界", script_text):
        findings.append({
            "severity": "medium",
            "category": "boundary_conditions",
            "message": "未发现明显约束或边界条件检查。",
        })
    if result is not None:
        metrics = result.get("metrics") or result.get("results") or result.get("returncode")
        if metrics is None:
            findings.append({
                "severity": "medium",
                "category": "metrics",
                "message": "result.json 缺少 metrics/results/returncode 等可审计结果字段。",
            })
    else:
        findings.append({
            "severity": "high",
            "category": "metrics",
            "message": "未提供或未找到 result.json，无法检查指标和结果。",
        })
    if run_dir:
        figure_cards = list(run_dir.glob("*figure*card*.md"))
        figures = list(run_dir.glob("*.png")) + list(run_dir.glob("*.jpg")) + list(run_dir.glob("*.svg")) + list(run_dir.glob("*.pdf"))
        if figures and not figure_cards:
            findings.append({
                "severity": "low",
                "category": "figure_alignment",
                "message": "检测到图表文件但未发现 figure_card。",
            })
    return findings


def cmd_review(args: argparse.Namespace) -> int:
    run_dir = Path(args.run_dir).resolve() if args.run_dir else None
    output = Path(args.output or (run_dir / "review" if run_dir else "review")).resolve()
    output.mkdir(parents=True, exist_ok=True)
    script_path = Path(args.script).resolve() if args.script else (run_dir / "script.py" if run_dir else None)
    result_path = Path(args.result_json).resolve() if args.result_json else (run_dir / "result.json" if run_dir else None)
    script_text = read_text(script_path) if script_path and script_path.exists() else ""
    result = json.loads(result_path.read_text(encoding="utf-8")) if result_path and result_path.exists() else None
    findings = detect_review_findings(script_text, result, run_dir)
    status = "pass"
    if any(f["severity"] == "high" for f in findings):
        status = "fail"
    elif findings:
        status = "warn"
    report = {
        "status": status,
        "script": str(script_path) if script_path else "",
        "result_json": str(result_path) if result_path else "",
        "findings": findings,
        "checked_at": datetime.now().isoformat(timespec="seconds"),
    }
    write_json(output / "review_report.json", report)
    write_card(
        output / "review_card.md",
        "评审对象",
        [
            ("范围", [f"- 状态: {status}", f"- 脚本: {script_path}", f"- 结果: {result_path}"]),
            ("发现的问题", [f"- [{f['severity']}] {f['category']}: {f['message']}" for f in findings]),
            ("处理决议", ["- fail 表示阻断进入论文或最终导出；warn 需要 Lead 明确接受或修复。"]),
        ],
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if status != "fail" or args.allow_fail else 1


def cmd_structure(args: argparse.Namespace) -> int:
    source = Path(args.input).resolve()
    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    if source.is_dir():
        result_path = source / "result.json"
        raw_artifacts = sorted(str(p) for p in source.iterdir() if p.is_file())
    else:
        result_path = source
        raw_artifacts = [str(source)]
    data = json.loads(result_path.read_text(encoding="utf-8"))
    summary = data.get("summary") or data.get("status") or data.get("method") or "结果已整理"
    paper_material = {
        "summary": summary,
        "source": str(result_path),
        "artifacts": raw_artifacts,
        "suggested_cards": {
            "result_card": str(output / "result_card.md"),
            "figure_card": str(output / "figure_card.md"),
            "paper_material": str(output / "paper_material.md"),
        },
    }
    write_json(output / "structured_result.json", paper_material)
    write_card(
        output / "result_card.md",
        "结果摘要",
        [
            ("对应问题", [f"- {args.problem or '<待填写>'}"]),
            ("主要结论", [f"- {summary}"]),
            ("输出文件", [f"- {p}" for p in raw_artifacts]),
        ],
    )
    write_card(
        output / "figure_card.md",
        "图表信息",
        [
            ("图表与表格", [f"- {p}" for p in raw_artifacts if Path(p).suffix.lower() in {'.csv', '.png', '.jpg', '.svg', '.pdf'}]),
            ("论文使用", ["- 论文手需补充图表编号、caption、单位和结论句。"]),
        ],
    )
    (output / "paper_material.md").write_text(
        "\n".join([
            "# 论文素材",
            "",
            f"- 对应问题: {args.problem or '<待填写>'}",
            f"- 来源: {result_path}",
            f"- 核心结论: {summary}",
            "- 需要论文手核对：公式引用、图表编号、单位、结果解释。",
            "",
        ]),
        encoding="utf-8",
    )
    print(json.dumps(paper_material, ensure_ascii=False, indent=2))
    return 0


def numeric_values_from_result(data: Any) -> list[float]:
    values: list[float] = []
    if isinstance(data, dict):
        for value in data.values():
            values.extend(numeric_values_from_result(value))
    elif isinstance(data, list):
        for value in data:
            values.extend(numeric_values_from_result(value))
    elif isinstance(data, (int, float)) and not isinstance(data, bool):
        if math.isfinite(float(data)):
            values.append(float(data))
    return values


def cmd_validate(args: argparse.Namespace) -> int:
    result = json.loads(Path(args.result_json).read_text(encoding="utf-8"))
    baseline = json.loads(Path(args.baseline_json).read_text(encoding="utf-8")) if args.baseline_json else None
    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    values = numeric_values_from_result(result)
    issues: list[str] = []
    if not values:
        issues.append("结果中未发现可检查的数值。")
    if any(abs(v) > args.max_abs_value for v in values):
        issues.append(f"存在绝对值超过 {args.max_abs_value} 的数值，请检查量纲或异常。")
    if baseline is not None:
        current_values = numeric_values_from_result(result)
        baseline_values = numeric_values_from_result(baseline)
        if current_values and baseline_values:
            delta = abs(sum(current_values) / len(current_values) - sum(baseline_values) / len(baseline_values))
        else:
            delta = None
    else:
        delta = None
        issues.append("未提供 baseline_json，基线对照未执行。")
    status = "pass" if not issues else "warn"
    report = {
        "status": status,
        "numeric_value_count": len(values),
        "baseline_delta_mean": delta,
        "issues": issues,
        "sensitivity": {
            "perturbation": args.perturbation,
            "note": "当前为结果级检查；模型级扰动需由具体方法 CLI 输出多组 result.json。",
        },
    }
    write_json(output / "validation_report.json", report)
    write_card(
        output / "review_card.md",
        "评审对象",
        [
            ("范围", [f"- result_json: {args.result_json}", f"- baseline_json: {args.baseline_json or '未提供'}"]),
            ("发现的问题", [f"- {i}" for i in issues]),
            ("处理决议", [f"- 状态: {status}", "- 进入终审前需由建模手确认基线、边界和敏感性是否充分。"]),
        ],
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="mathmodel-pipeline")
    sub = parser.add_subparsers(dest="command", required=True)

    p_run = sub.add_parser("run", help="Run Python code and emit notebook/code/result cards")
    p_run.add_argument("--workspace", required=True)
    p_run.add_argument("--output")
    p_run.add_argument("--run-id")
    p_run.add_argument("--script")
    p_run.add_argument("--code")
    p_run.add_argument("--cwd")
    p_run.add_argument("--timeout", type=int, default=300)
    p_run.add_argument("script_args", nargs=argparse.REMAINDER)
    p_run.set_defaults(func=cmd_run)

    p_review = sub.add_parser("review", help="Review code and result artifacts")
    p_review.add_argument("--run-dir")
    p_review.add_argument("--script")
    p_review.add_argument("--result-json")
    p_review.add_argument("--output")
    p_review.add_argument("--allow-fail", action="store_true")
    p_review.set_defaults(func=cmd_review)

    p_structure = sub.add_parser("structure", help="Convert raw result into cards and paper material")
    p_structure.add_argument("--input", required=True)
    p_structure.add_argument("--output", required=True)
    p_structure.add_argument("--problem")
    p_structure.set_defaults(func=cmd_structure)

    p_validate = sub.add_parser("validate", help="Run result-level baseline and boundary checks")
    p_validate.add_argument("--result-json", required=True)
    p_validate.add_argument("--baseline-json")
    p_validate.add_argument("--output", required=True)
    p_validate.add_argument("--max-abs-value", type=float, default=1e12)
    p_validate.add_argument("--perturbation", type=float, default=0.05)
    p_validate.set_defaults(func=cmd_validate)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
