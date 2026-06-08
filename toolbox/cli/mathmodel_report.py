#!/usr/bin/env python3
"""MathModelAI LaTeX-first report pipeline."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import zipfile
from pathlib import Path
from typing import Any
from xml.etree import ElementTree as ET


NS = {"w": "http://schemas.openxmlformats.org/wordprocessingml/2006/main"}


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")


def xml_attr(el: ET.Element | None, name: str) -> str:
    if el is None:
        return ""
    return el.attrib.get(f"{{{NS['w']}}}{name}", "")


def twips_to_mm(value: str) -> float | None:
    if not value:
        return None
    try:
        return round(int(value) * 25.4 / 1440, 2)
    except ValueError:
        return None


def parse_docx_layout(docx: Path) -> dict[str, Any]:
    with zipfile.ZipFile(docx) as zf:
        names = zf.namelist()
        document_xml = zf.read("word/document.xml")
        styles_xml = zf.read("word/styles.xml") if "word/styles.xml" in names else b""
        headers = [zf.read(n).decode("utf-8", errors="ignore") for n in names if n.startswith("word/header") and n.endswith(".xml")]
        footers = [zf.read(n).decode("utf-8", errors="ignore") for n in names if n.startswith("word/footer") and n.endswith(".xml")]
    root = ET.fromstring(document_xml)
    sect = root.find(".//w:sectPr", NS)
    margins: dict[str, float | None] = {}
    page: dict[str, float | None] = {}
    if sect is not None:
        pg_mar = sect.find("w:pgMar", NS)
        pg_sz = sect.find("w:pgSz", NS)
        if pg_mar is not None:
            for key in ["top", "right", "bottom", "left", "header", "footer"]:
                margins[key] = twips_to_mm(xml_attr(pg_mar, key))
        if pg_sz is not None:
            page["width"] = twips_to_mm(xml_attr(pg_sz, "w"))
            page["height"] = twips_to_mm(xml_attr(pg_sz, "h"))

    paragraphs = []
    detected = {
        "abstract": False,
        "keywords": False,
        "captions": False,
        "references": False,
        "appendix": False,
    }
    for p in root.findall(".//w:p", NS):
        texts = [t.text or "" for t in p.findall(".//w:t", NS)]
        text = "".join(texts).strip()
        p_style = p.find("./w:pPr/w:pStyle", NS)
        if text:
            style = xml_attr(p_style, "val") or "Normal"
            paragraphs.append({"style": style, "text": text})
            lower = text.lower()
            detected["abstract"] = detected["abstract"] or "摘要" in text or "abstract" in lower
            detected["keywords"] = detected["keywords"] or "关键词" in text or "keywords" in lower
            detected["captions"] = detected["captions"] or bool(re.match(r"^(图|表)\s*\d+", text))
            detected["references"] = detected["references"] or "参考文献" in text or "references" in lower
            detected["appendix"] = detected["appendix"] or "附录" in text or "appendix" in lower

    styles = []
    if styles_xml:
        styles_root = ET.fromstring(styles_xml)
        for style in styles_root.findall(".//w:style", NS):
            style_id = xml_attr(style, "styleId")
            name = xml_attr(style.find("w:name", NS), "val")
            font = style.find(".//w:rFonts", NS)
            size = style.find(".//w:sz", NS)
            spacing = style.find(".//w:spacing", NS)
            if style_id or name:
                styles.append({
                    "id": style_id,
                    "name": name,
                    "font_ascii": xml_attr(font, "ascii"),
                    "font_east_asia": xml_attr(font, "eastAsia"),
                    "font_size_half_points": xml_attr(size, "val"),
                    "line_spacing": xml_attr(spacing, "line"),
                })

    return {
        "source_docx": str(docx),
        "page": page,
        "margins_mm": margins,
        "headers_count": len(headers),
        "footers_count": len(footers),
        "detected_sections": detected,
        "paragraph_samples": paragraphs[:30],
        "styles": styles[:80],
    }


def latex_length(mm: float | None, fallback: str) -> str:
    if mm is None:
        return fallback
    return f"{mm:.2f}mm"


def render_style_file(layout: dict[str, Any]) -> str:
    margins = layout.get("margins_mm", {})
    return "\n".join([
        r"\NeedsTeXFormat{LaTeX2e}",
        r"\ProvidesPackage{mathmodel-template}[MathModelAI converted template]",
        r"\RequirePackage{geometry}",
        r"\RequirePackage{graphicx}",
        r"\RequirePackage{booktabs}",
        r"\RequirePackage{amsmath,amssymb}",
        r"\RequirePackage{hyperref}",
        r"\RequirePackage{caption}",
        r"\geometry{",
        f"  top={latex_length(margins.get('top'), '25mm')},",
        f"  bottom={latex_length(margins.get('bottom'), '25mm')},",
        f"  left={latex_length(margins.get('left'), '25mm')},",
        f"  right={latex_length(margins.get('right'), '25mm')}",
        r"}",
        r"\setlength{\parindent}{2em}",
        r"\linespread{1.25}",
        "",
    ])


def render_main_tex() -> str:
    return "\n".join([
        r"\documentclass[UTF8,a4paper,12pt]{ctexart}",
        r"\usepackage{mathmodel-template}",
        r"\title{数学建模论文}",
        r"\author{MathModelAI}",
        r"\date{\today}",
        r"\begin{document}",
        r"\maketitle",
        r"\input{paper-body.tex}",
        r"\end{document}",
        "",
    ])


def cmd_create_sample_template(args: argparse.Namespace) -> int:
    out = Path(args.output).resolve()
    out.parent.mkdir(parents=True, exist_ok=True)
    document_xml = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>数学建模论文模板</w:t></w:r></w:p>
    <w:p><w:r><w:t>摘要：请在此填写摘要。</w:t></w:r></w:p>
    <w:sectPr>
      <w:pgSz w:w="11906" w:h="16838"/>
      <w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720"/>
    </w:sectPr>
  </w:body>
</w:document>"""
    styles_xml = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style>
</w:styles>"""
    with zipfile.ZipFile(out, "w") as zf:
        zf.writestr("[Content_Types].xml", """<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>""")
        zf.writestr("_rels/.rels", """<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>""")
        zf.writestr("word/_rels/document.xml.rels", """<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>""")
        zf.writestr("word/document.xml", document_xml)
        zf.writestr("word/styles.xml", styles_xml)
    print(json.dumps({"sample_docx": str(out)}, ensure_ascii=False, indent=2))
    return 0


def cmd_convert_template(args: argparse.Namespace) -> int:
    docx = Path(args.docx).resolve()
    out = Path(args.out).resolve()
    out.mkdir(parents=True, exist_ok=True)
    layout = parse_docx_layout(docx)
    write_json(out / "template_layout.json", layout)
    (out / "mathmodel-template.sty").write_text(render_style_file(layout), encoding="utf-8")
    (out / "main.tex").write_text(render_main_tex(), encoding="utf-8")
    report_lines = [
        "# 模板转换差异报告",
        "",
        f"- 来源 Word 模板: {docx}",
        f"- 页面尺寸: {layout.get('page')}",
        f"- 页边距: {layout.get('margins_mm')}",
        "- 已转换: 页面边距、基础字号、行距、标题/正文主结构。",
        "- 待人工核对: 页眉页脚、复杂表格样式、封面特殊排版、赛事专用字体。",
    ]
    (out / "template_fidelity_report.md").write_text("\n".join(report_lines) + "\n", encoding="utf-8")
    print(json.dumps({"template_dir": str(out), "layout": layout}, ensure_ascii=False, indent=2))
    return 0


def escape_latex(text: str) -> str:
    repl = {
        "\\": r"\textbackslash{}",
        "&": r"\&",
        "%": r"\%",
        "$": r"\$",
        "#": r"\#",
        "_": r"\_",
        "{": r"\{",
        "}": r"\}",
        "~": r"\textasciitilde{}",
        "^": r"\textasciicircum{}",
    }
    return "".join(repl.get(ch, ch) for ch in text)


def markdown_to_latex(md: str) -> str:
    lines = []
    in_itemize = False
    for raw in md.splitlines():
        line = raw.rstrip()
        if line.startswith("## "):
            if in_itemize:
                lines.append(r"\end{itemize}")
                in_itemize = False
            lines.append(r"\section{" + escape_latex(line[3:].strip()) + "}")
        elif line.startswith("### "):
            if in_itemize:
                lines.append(r"\end{itemize}")
                in_itemize = False
            lines.append(r"\subsection{" + escape_latex(line[4:].strip()) + "}")
        elif line.startswith("- "):
            if not in_itemize:
                lines.append(r"\begin{itemize}")
                in_itemize = True
            lines.append(r"\item " + escape_latex(line[2:].strip()))
        elif line.strip() == "":
            if in_itemize:
                lines.append(r"\end{itemize}")
                in_itemize = False
            lines.append("")
        else:
            if in_itemize:
                lines.append(r"\end{itemize}")
                in_itemize = False
            lines.append(escape_latex(line))
    if in_itemize:
        lines.append(r"\end{itemize}")
    return "\n".join(lines) + "\n"


def collect_sections(sections_dir: str | None) -> str:
    if not sections_dir:
        return ""
    root = Path(sections_dir)
    if not root.is_dir():
        return ""
    parts = []
    for path in sorted(root.glob("*.md")):
        parts.append(path.read_text(encoding="utf-8"))
    return "\n\n".join(parts)


def cmd_assemble(args: argparse.Namespace) -> int:
    template_dir = Path(args.template_dir).resolve()
    body = Path(args.body).read_text(encoding="utf-8")
    sections = collect_sections(args.sections_dir)
    out_dir = Path(args.out).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    shutil.copy2(template_dir / "mathmodel-template.sty", out_dir / "mathmodel-template.sty")
    for name in ["template_fidelity_report.md", "template_layout.json"]:
        src = template_dir / name
        if src.exists():
            shutil.copy2(src, out_dir / name)
    combined_md = body + ("\n\n" + sections if sections else "")
    body_tex = markdown_to_latex(combined_md)
    (out_dir / "paper-body.tex").write_text(body_tex, encoding="utf-8")
    main = (template_dir / "main.tex").read_text(encoding="utf-8")
    if args.title:
        main = re.sub(r"\\title\{.*?\}", r"\\title{" + escape_latex(args.title) + "}", main)
    (out_dir / "paper.tex").write_text(main, encoding="utf-8")
    manifest = {
        "paper_tex": str(out_dir / "paper.tex"),
        "paper_body_tex": str(out_dir / "paper-body.tex"),
        "source_body_template": str(Path(args.body).resolve()),
        "sections_dir": args.sections_dir or "",
    }
    write_json(out_dir / "assemble_report.json", manifest)
    print(json.dumps(manifest, ensure_ascii=False, indent=2))
    return 0


def preflight_tex(tex_dir: Path, tex_file: Path) -> dict[str, Any]:
    text = tex_file.read_text(encoding="utf-8")
    missing_images = []
    for match in re.findall(r"\\includegraphics(?:\[[^\]]*\])?\{([^}]+)\}", text):
        image = (tex_dir / match).resolve()
        if not image.exists():
            missing_images.append(match)
    material_gaps = re.findall(r"\[MATERIAL GAP:[^\]]+\]", text)
    citations = re.findall(r"\\cite\{([^}]+)\}", text)
    return {
        "missing_images": missing_images,
        "material_gaps": material_gaps,
        "citations": citations,
        "ok": not missing_images and not material_gaps,
    }


def cmd_preflight(args: argparse.Namespace) -> int:
    tex = Path(args.tex).resolve()
    report = preflight_tex(tex.parent, tex)
    output = Path(args.output).resolve() if args.output else tex.parent / "preflight_report.json"
    write_json(output, report)
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report["ok"] else 1


def run_cmd(cmd: list[str], cwd: Path) -> dict[str, Any]:
    start = time_monotonic()
    proc = subprocess.run(cmd, cwd=str(cwd), text=True, capture_output=True, check=False)
    return {
        "cmd": cmd,
        "returncode": proc.returncode,
        "stdout": proc.stdout[-4000:],
        "stderr": proc.stderr[-4000:],
        "duration_seconds": round(time_monotonic() - start, 3),
    }


def time_monotonic() -> float:
    import time
    return time.monotonic()


def cmd_compile(args: argparse.Namespace) -> int:
    tex = Path(args.tex).resolve()
    out_dir = Path(args.out).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    work_tex = out_dir / "paper.tex"
    if tex.parent != out_dir:
        for name in ["paper.tex", "paper-body.tex", "mathmodel-template.sty"]:
            src = tex.parent / name
            if src.exists():
                shutil.copy2(src, out_dir / name)
    else:
        work_tex = tex
    preflight = preflight_tex(out_dir, work_tex)
    commands = []
    pdf_path = out_dir / "paper.pdf"
    if shutil.which("latexmk"):
        commands.append(run_cmd(["latexmk", "-xelatex", "-interaction=nonstopmode", "-halt-on-error", "paper.tex"], out_dir))
    elif shutil.which("xelatex"):
        commands.append(run_cmd(["xelatex", "-interaction=nonstopmode", "-halt-on-error", "paper.tex"], out_dir))
        commands.append(run_cmd(["xelatex", "-interaction=nonstopmode", "-halt-on-error", "paper.tex"], out_dir))
    else:
        commands.append({"cmd": ["xelatex"], "returncode": 127, "stdout": "", "stderr": "xelatex/latexmk not found", "duration_seconds": 0})

    pdf_ok = pdf_path.exists() and all(c["returncode"] == 0 for c in commands)
    docx_path = out_dir / "paper.docx"
    pandoc_report: dict[str, Any]
    if shutil.which("pandoc"):
        pandoc_report = run_cmd(["pandoc", "paper.tex", "-o", str(docx_path)], out_dir)
    else:
        pandoc_report = {"cmd": ["pandoc"], "returncode": 127, "stdout": "", "stderr": "pandoc not found", "duration_seconds": 0}
    docx_ok = docx_path.exists() and pandoc_report["returncode"] == 0

    report = {
        "template_fidelity_report": str(tex.parent / "template_fidelity_report.md"),
        "preflight": preflight,
        "pdf": {"path": str(pdf_path), "ok": pdf_ok, "commands": commands},
        "docx": {"path": str(docx_path), "ok": docx_ok, "command": pandoc_report},
        "outputs": {
            "paper_tex": str(work_tex),
            "paper_pdf": str(pdf_path),
            "paper_docx": str(docx_path),
        },
    }
    write_json(out_dir / "compile_report.json", report)
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if pdf_ok and docx_ok and preflight["ok"] else 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="mathmodel-report")
    sub = parser.add_subparsers(dest="command", required=True)

    p_sample = sub.add_parser("create-sample-template")
    p_sample.add_argument("--output", required=True)
    p_sample.set_defaults(func=cmd_create_sample_template)

    p_convert = sub.add_parser("convert-template")
    p_convert.add_argument("docx")
    p_convert.add_argument("--out", required=True)
    p_convert.set_defaults(func=cmd_convert_template)

    p_assemble = sub.add_parser("assemble")
    p_assemble.add_argument("--template-dir", required=True)
    p_assemble.add_argument("--body", required=True)
    p_assemble.add_argument("--sections-dir")
    p_assemble.add_argument("--out", required=True)
    p_assemble.add_argument("--title")
    p_assemble.set_defaults(func=cmd_assemble)

    p_preflight = sub.add_parser("preflight")
    p_preflight.add_argument("--tex", required=True)
    p_preflight.add_argument("--output")
    p_preflight.set_defaults(func=cmd_preflight)

    p_compile = sub.add_parser("compile")
    p_compile.add_argument("--tex", required=True)
    p_compile.add_argument("--out", required=True)
    p_compile.set_defaults(func=cmd_compile)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
