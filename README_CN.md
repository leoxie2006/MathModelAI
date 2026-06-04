# MathModelAI

MathModelAI 是一个基于现有 Go + Web + MCP + 多智能体架构整理出的数学建模 AI 开发骨架。

当前分支只保留架构和少量示例文件，便于后续围绕数学建模能力继续开发。原始完整示例已保存在 `example` 分支。

## 当前保留内容

- `cmd/`：服务入口与测试入口。
- `internal/`：后端核心架构，包括配置、HTTP API、Agent、MCP、知识库、项目事实黑板、审计和存储。
- `web/`：静态前端。
- `agents/`：多代理 Markdown 示例，当前保留主代理和建模分析子代理。
- `roles/`：角色 YAML 示例，当前保留默认角色和数学建模角色。
- `skills/`：Agent Skills 示例，当前保留问题抽象和模型验证两个技能包。
- `tools/`：MCP 工具模板，当前保留 Python 执行和 Shell 执行两个通用工具。
- `knowledge_base/`：数学建模知识库示例。
- `docs/`：架构说明和快速开始。

## 快速开始

环境要求：

- Go 1.21+
- Python 3.10+

启动：

```bash
./run.sh --http
```

然后访问：

```text
http://127.0.0.1:8080/
```

模型配置在 `config.yaml` 中，首次使用前请配置 `openai.base_url`、`openai.api_key` 和 `openai.model`。

## 后续开发建议

1. 先完善 `agents/` 中的数学建模多代理分工。
2. 在 `skills/` 中沉淀建模流程，例如优化、预测、评价、仿真和报告写作。
3. 在 `tools/` 中接入 Python 科学计算、求解器、绘图和数据处理工具。
4. 在 `knowledge_base/` 中增加常用模型、公式和案例。
5. 逐步把前端中的旧业务文案替换为数学建模业务文案。

更多说明见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) 和 [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md)。
