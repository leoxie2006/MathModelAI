# 角色示例

`roles/` 下的 YAML 文件会被系统加载为可选角色。

- `默认.yaml`：不添加额外提示，适合通用对话。
- `数学建模.yaml`：数学建模方向示例，限制为 Python 执行、Shell 执行和知识库检索等基础工具。

新增角色时复制任一 YAML，调整 `name`、`description`、`user_prompt`、`tools` 与 `enabled` 即可。
