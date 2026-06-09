#!/usr/bin/env python3
"""Project workspace helper for MathModelAI.

This CLI is intentionally filesystem-only. The web API owns database indexing;
after generated files are written, use the project page "扫描工作区" action or
POST /api/projects/{id}/files/scan to refresh the index.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


DIRS = ("attachments", "data", "code", "outputs", "paper", "cards")


def workspace_root(project_id: str, root: str | None = None) -> Path:
    base = Path(root or Path.cwd() / "workspaces").resolve()
    safe_id = project_id.strip().replace("/", "_").replace("\\", "_") or "unnamed"
    return base / safe_id


def ensure_workspace(project_id: str, root: str | None = None) -> Path:
    ws = workspace_root(project_id, root)
    for name in DIRS:
        (ws / name).mkdir(parents=True, exist_ok=True)
    return ws


def safe_target(ws: Path, rel_path: str) -> Path:
    rel = Path(rel_path.strip().lstrip("/"))
    if not rel.parts:
        raise SystemExit("relative path is required")
    if rel.parts[0] not in DIRS:
        raise SystemExit(f"relative path must start with one of: {', '.join(DIRS)}")
    target = (ws / rel).resolve()
    if ws.resolve() not in target.parents and target != ws.resolve():
        raise SystemExit("path outside workspace")
    return target


def list_files(ws: Path) -> list[dict[str, object]]:
    files: list[dict[str, object]] = []
    for path in sorted(ws.rglob("*")):
        if not path.is_file():
            continue
        rel = path.relative_to(ws).as_posix()
        if rel.startswith("."):
            continue
        files.append({
            "rel_path": rel,
            "abs_path": str(path),
            "size_bytes": path.stat().st_size,
        })
    return files


def cmd_info(args: argparse.Namespace) -> int:
    ws = ensure_workspace(args.project_id, args.root)
    print(json.dumps({
        "project_id": args.project_id,
        "workspace_root": str(ws),
        "dirs": {name: str(ws / name) for name in DIRS},
        "files": list_files(ws),
    }, ensure_ascii=False, indent=2))
    return 0


def cmd_write(args: argparse.Namespace) -> int:
    ws = ensure_workspace(args.project_id, args.root)
    target = safe_target(ws, args.rel_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(args.content, encoding="utf-8")
    print(json.dumps({
        "project_id": args.project_id,
        "workspace_root": str(ws),
        "rel_path": target.relative_to(ws).as_posix(),
        "abs_path": str(target),
        "size_bytes": target.stat().st_size,
    }, ensure_ascii=False, indent=2))
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description="MathModelAI project workspace helper")
    sub = parser.add_subparsers(dest="command", required=True)

    p_info = sub.add_parser("info")
    p_info.add_argument("--project-id", required=True)
    p_info.add_argument("--root")
    p_info.set_defaults(func=cmd_info)

    p_write = sub.add_parser("write")
    p_write.add_argument("--project-id", required=True)
    p_write.add_argument("--rel-path", required=True)
    p_write.add_argument("--content", required=True)
    p_write.add_argument("--root")
    p_write.set_defaults(func=cmd_write)

    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
