import csv
import json
import random
from pathlib import Path

random.seed(42)

rows = [
    {"scheme": "A", "score": 0.72},
    {"scheme": "B", "score": 0.81},
    {"scheme": "C", "score": 0.67},
]

with Path("rank_table.csv").open("w", encoding="utf-8", newline="") as f:
    writer = csv.DictWriter(f, fieldnames=["scheme", "score"])
    writer.writeheader()
    writer.writerows(rows)

best = max(rows, key=lambda r: r["score"])
Path("model_output.json").write_text(
    json.dumps(
        {
            "summary": f"best scheme is {best['scheme']}",
            "metrics": {"best_score": best["score"], "scheme_count": len(rows)},
            "results": rows,
        },
        ensure_ascii=False,
        indent=2,
    ),
    encoding="utf-8",
)

print(f"best={best['scheme']} score={best['score']}")
