# 工具示例

`tools/` 下的 YAML 文件会被加载为 MCP 工具模板。当前仅保留两个通用示例：

- `execute-python-script.yaml`：在虚拟环境中运行 Python 片段，适合建模计算、数据处理和可视化。
- `exec.yaml`：执行系统命令，适合文件检查、依赖安装和工程辅助操作。

新增工具时保持 `name`、`command`、`enabled`、`description` 和 `parameters` 字段完整即可。
