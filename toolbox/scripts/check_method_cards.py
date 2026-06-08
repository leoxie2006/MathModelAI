#!/usr/bin/env python3
"""Validate the local math modeling method-card catalog."""

from pathlib import Path
import sys


ROOT = Path(__file__).resolve().parents[1]
METHOD_ROOT = ROOT / "methods"

EXPECTED = {
    "optimization": [
        "linear-programming",
        "integer-programming",
        "nonlinear-programming",
        "multi-objective-optimization",
        "genetic-algorithm",
        "simulated-annealing",
    ],
    "prediction": [
        "regression",
        "xgboost",
        "arima",
        "exponential-smoothing",
        "gm11",
    ],
    "evaluation": [
        "ahp",
        "entropy-weight",
        "topsis",
        "fuzzy-comprehensive-evaluation",
        "pca",
        "dea",
    ],
    "simulation": [
        "monte-carlo",
        "queuing-theory",
        "cellular-automata",
        "system-dynamics",
    ],
}

REQUIRED_FIELDS = [
    "id:",
    "name:",
    "category:",
    "applies_when:",
    "inputs:",
    "outputs:",
    "assumptions:",
    "validation:",
    "pitfalls:",
    "future_cli:",
]


def main() -> int:
    errors: list[str] = []
    count = 0

    for category, methods in EXPECTED.items():
        category_dir = METHOD_ROOT / category
        if not category_dir.is_dir():
            errors.append(f"missing category directory: {category_dir}")
            continue
        for method_id in methods:
            count += 1
            path = category_dir / f"{method_id}.yaml"
            if not path.is_file():
                errors.append(f"missing method card: {path}")
                continue
            text = path.read_text(encoding="utf-8")
            for field in REQUIRED_FIELDS:
                if field not in text:
                    errors.append(f"{path}: missing field {field}")
            if f"id: {method_id}" not in text:
                errors.append(f"{path}: id does not match filename")
            if f"category: {category}" not in text:
                errors.append(f"{path}: category does not match directory")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1

    print(f"method cards ok: {count}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
