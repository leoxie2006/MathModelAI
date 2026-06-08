package project

import (
	"fmt"
	"sort"
	"strings"

	"mathmodel-ai/internal/config"
	"mathmodel-ai/internal/database"
)

// AppendSystemPromptBlock 将附加块追加到 system prompt。
func AppendSystemPromptBlock(base, block string) string {
	base = strings.TrimSpace(base)
	block = strings.TrimSpace(block)
	if block == "" {
		return base
	}
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}

// BuildFactIndexBlock 为 Agent 系统提示生成项目黑板索引（仅 key + summary，不含 body）。
func BuildFactIndexBlock(db *database.DB, projectID string, cfg config.ProjectConfig) (string, error) {
	if db == nil || !cfg.Enabled {
		return "", nil
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", nil
	}

	proj, err := db.GetProject(projectID)
	if err != nil {
		return "", err
	}

	facts, err := db.ListProjectFactsForIndex(projectID, cfg.DefaultInjectDeprecated)
	if err != nil {
		return "", err
	}
	if len(facts) == 0 {
		return fmt.Sprintf("## 项目黑板索引（project: %s, id: %s）\n（暂无建模卡片）\n需要写入请使用 upsert_project_fact；需要详情请调用 get_project_fact(fact_key)。", proj.Name, proj.ID), nil
	}

	sort.SliceStable(facts, func(i, j int) bool {
		if facts[i].Pinned != facts[j].Pinned {
			return facts[i].Pinned
		}
		return facts[i].UpdatedAt.After(facts[j].UpdatedAt)
	})

	maxRunes := cfg.FactIndexMaxRunesEffective()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 项目黑板索引（project: %s, id: %s）\n", proj.Name, proj.ID))
	used := len([]rune(b.String()))
	omitted := 0

	for _, f := range facts {
		line := fmt.Sprintf("- [%s] %s — %s (%s)\n", f.FactKey, f.Category, strings.TrimSpace(f.Summary), f.Confidence)
		lineRunes := len([]rune(line))
		if used+lineRunes > maxRunes {
			omitted++
			continue
		}
		b.WriteString(line)
		used += lineRunes
	}

	if omitted > 0 {
		b.WriteString(fmt.Sprintf("\n（另有 %d 条未列入索引，请使用 list_project_facts 或 search_project_facts 查询。）\n", omitted))
	}
	b.WriteString("需要完整内容（题意、数据审计、模型规格、代码入口、结果、图表、论文或评审细节）时必须调用 get_project_fact(fact_key)，禁止凭摘要臆造细节。\n")
	b.WriteString("写入卡片时：summary 写「什么 + 属于哪个问题/文件/模型 + 如何验证或使用」；body 写结构化证据、产物路径和交叉复核结论（fact_key 建议 problem|data|model|code|result|figure|paper|review/ 前缀）。\n")
	return b.String(), nil
}
