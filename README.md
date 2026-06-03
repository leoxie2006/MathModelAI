# MathModelAI

MathModelAI is a math-modeling AI development skeleton extracted from the existing Go + Web + MCP + multi-agent architecture.

This branch keeps only the architecture plus a small set of example files. The original security-testing examples are preserved on the `example` branch.

## What Remains

- `cmd/`: service and test entry points.
- `internal/`: backend architecture, including config, HTTP APIs, agents, MCP, knowledge base, project facts, audit, and storage.
- `web/`: static frontend.
- `agents/`: multi-agent Markdown examples.
- `roles/`: role YAML examples.
- `skills/`: Agent Skills examples.
- `tools/`: MCP tool YAML examples.
- `knowledge_base/`: math-modeling knowledge examples.
- `mcp-servers/`: external MCP service example.
- `plugins/`: plugin extension example.
- `docs/`: architecture notes and getting started guide.

## Quick Start

Requirements:

- Go 1.21+
- Python 3.10+

Run:

```bash
./run.sh --http
```

Open:

```text
http://127.0.0.1:8080/
```

Configure `openai.base_url`, `openai.api_key`, and `openai.model` in `config.yaml` before first use.

## Suggested Next Steps

1. Refine the math-modeling agent roles in `agents/`.
2. Add reusable modeling workflows under `skills/`.
3. Add scientific-computing, optimization, plotting, and data-processing tools under `tools/`.
4. Grow the formula and case-study library under `knowledge_base/`.
5. Gradually replace the remaining security-domain UI text with math-modeling product language.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md).
