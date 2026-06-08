# 架构速览

当前分支保留原项目的主要架构骨架：

- `cmd/`：服务启动与测试入口。
- `internal/`：核心后端模块，包括配置、HTTP handler、Agent、MCP、知识库、项目事实黑板、审计与存储。
- `web/`：静态前端页面、样式和交互脚本。
- `agents/`：多代理 Markdown 定义。
- `roles/`：角色配置。
- `skills/`：Agent Skills 技能包。
- `toolbox/tools/`：MCP 工具 YAML 模板。
- `toolbox/methods/`：自建数学建模方法模板和未来 CLI 工具。
- `toolbox/report/`：官方 Word 模板转 LaTeX、正文拼接、PDF 编译和 Word 导出产线。
- `knowledge_base/`：RAG 知识库 Markdown 内容。

原始完整样例已移到 `example` 分支保留；当前分支只作为 MathModelAI 的轻量开发骨架。
