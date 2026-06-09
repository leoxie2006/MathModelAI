package project

import (
	"encoding/json"
	"fmt"
	"strings"

	"mathmodel-ai/internal/config"
	"mathmodel-ai/internal/database"
)

// projectScopePayload 解析 projects.scope_json（约定字段，可扩展）。
type projectScopePayload struct {
	Targets      []string `json:"targets"`
	Exclude      []string `json:"exclude"`
	DataFiles    []string `json:"data_files"`
	Deliverables []string `json:"deliverables"`
	Constraints  []string `json:"constraints"`
	Deadline     string   `json:"deadline"`
	Notes        string   `json:"notes"`
}

// BuildScopeBlock 将项目 scope_json 格式化为 Agent 可读的赛题资料与约束块。
func BuildScopeBlock(proj *database.Project) string {
	if proj == nil {
		return ""
	}
	raw := strings.TrimSpace(proj.ScopeJSON)
	if raw == "" {
		return ""
	}

	var payload projectScopePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return fmt.Sprintf("## 项目资料与约束（project: %s）\n（scope_json 非合法 JSON，请人工核对配置）\n```\n%s\n```\n"+
			"继续建模前应先澄清题目资料、数据文件、交付格式和限制条件。\n", proj.Name, truncateRunes(raw, 800))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 项目资料与约束（project: %s, id: %s）\n", proj.Name, proj.ID))
	b.WriteString("以下信息用于约束建模与交付。若题面、附件或用户说明冲突，先写入 problem/review 卡片并请求 checkpoint。\n")

	if len(payload.Targets) > 0 {
		b.WriteString("\n**研究对象 / 任务目标（targets）**：\n")
		for _, t := range payload.Targets {
			t = strings.TrimSpace(t)
			if t != "" {
				b.WriteString("- " + t + "\n")
			}
		}
	}
	if len(payload.Exclude) > 0 {
		b.WriteString("\n**排除范围 / 不采用资料（exclude）**：\n")
		for _, t := range payload.Exclude {
			t = strings.TrimSpace(t)
			if t != "" {
				b.WriteString("- " + t + "\n")
			}
		}
	}
	if len(payload.DataFiles) > 0 {
		b.WriteString("\n**数据文件（data_files）**：\n")
		for _, t := range payload.DataFiles {
			t = strings.TrimSpace(t)
			if t != "" {
				b.WriteString("- " + t + "\n")
			}
		}
	}
	if len(payload.Deliverables) > 0 {
		b.WriteString("\n**交付物（deliverables）**：\n")
		for _, t := range payload.Deliverables {
			t = strings.TrimSpace(t)
			if t != "" {
				b.WriteString("- " + t + "\n")
			}
		}
	}
	if len(payload.Constraints) > 0 {
		b.WriteString("\n**限制条件（constraints）**：\n")
		for _, t := range payload.Constraints {
			t = strings.TrimSpace(t)
			if t != "" {
				b.WriteString("- " + t + "\n")
			}
		}
	}
	if deadline := strings.TrimSpace(payload.Deadline); deadline != "" {
		b.WriteString("\n**截止时间（deadline）**：\n" + deadline + "\n")
	}
	if n := strings.TrimSpace(payload.Notes); n != "" {
		b.WriteString("\n**说明（notes）**：\n" + n + "\n")
	}
	if len(payload.Targets) == 0 && len(payload.Exclude) == 0 && len(payload.DataFiles) == 0 &&
		len(payload.Deliverables) == 0 && len(payload.Constraints) == 0 &&
		strings.TrimSpace(payload.Deadline) == "" && strings.TrimSpace(payload.Notes) == "" {
		b.WriteString("\n（scope_json 已配置但未识别 targets/exclude/data_files/deliverables/constraints/deadline/notes 字段，原始内容供参考）\n```json\n")
		b.WriteString(truncateRunes(raw, 1200))
		b.WriteString("\n```\n")
	}
	b.WriteString("\n建模、编程和论文写作均不得无依据引入题面外数据或改动交付要求；确需扩展资料时先写入 review/checkpoint 卡片并等待确认。\n")
	return b.String()
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// BuildProjectBlackboardBlock 组合项目资料约束 + 事实黑板索引。
func BuildProjectBlackboardBlock(db *database.DB, projectID string, cfg config.ProjectConfig) (string, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return "", nil
	}
	proj, err := db.GetProject(projectID)
	if err != nil {
		return "", err
	}
	parts := []string{}
	if scope := strings.TrimSpace(BuildScopeBlock(proj)); scope != "" {
		parts = append(parts, scope)
	}
	workspace, err := BuildWorkspaceIndexBlock(db, projectID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(workspace) != "" {
		parts = append(parts, workspace)
	}
	index, err := BuildFactIndexBlock(db, projectID, cfg)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(index) != "" {
		parts = append(parts, index)
	}
	return strings.Join(parts, "\n\n"), nil
}
