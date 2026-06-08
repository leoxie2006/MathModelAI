package project

import (
	"strings"

	"mathmodel-ai/internal/mcp/builtin"
)

// 边建模边记录：统一节奏文案（agents/*.md 须与 FactRecordingIncrementalRhythmMarkdown 保持一致）。
const (
	factRhythmCore              = "勿等阶段结束或收尾再批量写入。每**确认**一条题意、数据、模型、代码、结果、图表、论文或评审结论后，**立即**调用 `upsert_project_fact`（同 fact_key 覆盖更新）。继续下一步工作前优先落库，避免上下文压缩后细节丢失。未绑项目时说明无法写黑板，仍在本轮保留结构化摘要。"
	factRhythmCoordinatorSuffix = "委派/子任务返回新认知、产物路径或阻断问题时，由协调者及时写入，勿假定子代理已记。"
	factRhythmSubAgentSuffix    = "若工具集中无上述工具，须在交付物末尾给出「待落库」结构化条目（fact_key 建议、summary、body 要点、证据/产物路径），供协调者**立即**写入。"
)

// FactRecordingIncrementalRhythmMarkdown 返回边建模边记录节奏（Markdown，供 agents/*.md 与文档对齐）。
func FactRecordingIncrementalRhythmMarkdown(coordinator, subAgent bool) string {
	var b strings.Builder
	b.WriteString("- **边建模边记录（强制节奏）**：")
	b.WriteString(factRhythmCore)
	if coordinator {
		b.WriteString(factRhythmCoordinatorSuffix)
	}
	if subAgent {
		b.WriteString(factRhythmSubAgentSuffix)
	}
	return b.String()
}

func factRecordingIncrementalRhythmBuiltin(coordinator, subAgent bool) string {
	var b strings.Builder
	b.WriteString("- **边建模边记录（强制节奏）**：勿等阶段结束或收尾再批量写入。每**确认**一条题意、数据、模型、代码、结果、图表、论文或评审结论后，**立即**调用 ")
	b.WriteString(builtin.ToolUpsertProjectFact)
	b.WriteString("（同 fact_key 覆盖更新）。继续下一步工作前优先落库，避免上下文压缩后细节丢失。未绑项目时说明无法写黑板，仍在本轮保留结构化摘要。")
	if coordinator {
		b.WriteString(factRhythmCoordinatorSuffix)
	}
	if subAgent {
		b.WriteString(factRhythmSubAgentSuffix)
	}
	return b.String()
}

// FactRecordingBlackboardSection 项目黑板的完整系统提示块（单/多 Agent 主代理共用）。
// coordinatorDelegate 为 true 时追加「协调者代子代理落库」说明（Deep / plan_execute / supervisor）。
func FactRecordingBlackboardSection(coordinatorDelegate bool) string {
	var b strings.Builder
	b.WriteString("## 项目黑板（数学建模卡片）\n\n")
	b.WriteString("当前对话若已绑定项目，系统会自动注入「项目黑板索引」（仅 fact_key + 摘要）。**摘要不足时必须调用 ")
	b.WriteString(builtin.ToolGetProjectFact)
	b.WriteString("(fact_key) 获取 body，禁止凭摘要臆造细节。**\n\n")
	b.WriteString(factRecordingIncrementalRhythmBuiltin(coordinatorDelegate, false))
	b.WriteString("\n\n")
	b.WriteString("- **题意/数据/模型/代码/结果/论文/评审卡片**：使用 ")
	b.WriteString(builtin.ToolUpsertProjectFact)
	b.WriteString("，fact_key 建议 `category/slug`（如 `problem/requirements`、`data/attachment-audit`、`model/topsis-plan`、`result/problem-1`），同 key 覆盖更新；body 记结构化卡片、证据来源与产物路径。\n")
	b.WriteString("- **关键卡片 body 必填**：problem/data/model/code/result/figure/paper/review 类 fact 必须写清输入、输出、依据、验证方式和关联文件，禁止仅写结论；summary 写「什么 + 属于哪个问题/文件/模型 + 如何验证或使用」一行要点。\n")
	b.WriteString("- **无效或被替代的卡片**：使用 ")
	b.WriteString(builtin.ToolDeprecateProjectFact)
	b.WriteString(" 标记 deprecated；修订后的同类卡片优先覆盖原 fact_key，必要时在 body 记录替代关系。\n")
	b.WriteString("- 事实多时用 ")
	b.WriteString(builtin.ToolListProjectFacts)
	b.WriteString(" / ")
	b.WriteString(builtin.ToolSearchProjectFacts)
	b.WriteString(" 检索。\n\n")
	b.WriteString(FactRecordingGuidanceBlock())
	return b.String()
}

// FactRecordingSubAgentSection 子代理边建模边记录（无工具时输出待落库条目）。
func FactRecordingSubAgentSection() string {
	return "## 边建模边记录\n\n" + factRecordingIncrementalRhythmBuiltin(false, true) + "\n"
}

// FactRecordingBlackboardSectionMarkdown 与 FactRecordingBlackboardSection 等价的 Markdown（工具名为字面量，供 agents/*.md）。
func FactRecordingBlackboardSectionMarkdown(coordinatorDelegate bool) string {
	var b strings.Builder
	b.WriteString("## 项目黑板（数学建模卡片）\n\n")
	b.WriteString("当前对话若已绑定项目，系统会自动注入「项目黑板索引」（仅 `fact_key` + 摘要）。**摘要不足时必须调用 `get_project_fact(fact_key)` 获取 body，禁止凭摘要臆造细节。**\n\n")
	b.WriteString(FactRecordingIncrementalRhythmMarkdown(coordinatorDelegate, false))
	b.WriteString("\n\n")
	b.WriteString("- **题意/数据/模型/代码/结果/论文/评审卡片**：使用 **`upsert_project_fact`**，`fact_key` 建议 `category/slug`（如 `problem/requirements`、`data/attachment-audit`、`model/topsis-plan`、`result/problem-1`），同 key 覆盖更新；body 记结构化卡片、证据来源与产物路径。\n")
	b.WriteString("- **关键卡片 body 必填**：`problem` / `data` / `model` / `code` / `result` / `figure` / `paper` / `review` 类 fact 必须写清输入、输出、依据、验证方式和关联文件，禁止仅写结论；summary 写「什么 + 属于哪个问题/文件/模型 + 如何验证或使用」一行要点。\n")
	b.WriteString("- **无效或被替代的卡片**：使用 **`deprecate_project_fact`** 标记 deprecated；修订后的同类卡片优先覆盖原 `fact_key`，必要时在 body 记录替代关系。\n")
	b.WriteString("- 事实多时用 **`list_project_facts`** / **`search_project_facts`** 检索。\n\n")
	b.WriteString(FactRecordingGuidanceBlock())
	return b.String()
}
