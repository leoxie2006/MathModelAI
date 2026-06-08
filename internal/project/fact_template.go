package project

import (
	"fmt"
	"strings"
)

// 事实 category 常量（写入 upsert_project_fact 的 category 字段）。
const (
	FactCategoryProblem = "problem"
	FactCategoryData    = "data"
	FactCategoryModel   = "model"
	FactCategoryCode    = "code"
	FactCategoryResult  = "result"
	FactCategoryFigure  = "figure"
	FactCategoryPaper   = "paper"
	FactCategoryReview  = "review"
	FactCategoryNote    = "note"
)

var structuredModelingCategories = map[string]struct{}{
	FactCategoryProblem: {},
	FactCategoryData:    {},
	FactCategoryModel:   {},
	FactCategoryCode:    {},
	FactCategoryResult:  {},
	FactCategoryFigure:  {},
	FactCategoryPaper:   {},
	FactCategoryReview:  {},
}

var structuredModelingPrefixes = []string{
	FactCategoryProblem + "/",
	FactCategoryData + "/",
	FactCategoryModel + "/",
	FactCategoryCode + "/",
	FactCategoryResult + "/",
	FactCategoryFigure + "/",
	FactCategoryPaper + "/",
	FactCategoryReview + "/",
}

// RequiresStructuredFactBody 判断该事实是否应携带结构化建模卡片 body（非仅 summary）。
func RequiresStructuredFactBody(category, factKey string) bool {
	c := strings.ToLower(strings.TrimSpace(category))
	if _, ok := structuredModelingCategories[c]; ok {
		return true
	}
	key := strings.ToLower(strings.TrimSpace(factKey))
	for _, prefix := range structuredModelingPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// RequiresAttackChainBody 保留旧导出名以兼容调用点；当前语义已切换为建模卡片结构化 body。
func RequiresAttackChainBody(category, factKey string) bool {
	return RequiresStructuredFactBody(category, factKey)
}

// IsSparseFactBody 结构化建模事实 body 过短或缺少关键段落时返回 true（软校验，不阻断写入）。
func IsSparseFactBody(category, factKey, body string) bool {
	if !RequiresStructuredFactBody(category, factKey) {
		return false
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return true
	}
	lower := strings.ToLower(body)
	hasHeading := strings.Contains(body, "## ") || strings.Contains(body, "### ")
	hasEvidence := containsAny(body, []string{
		"证据", "来源", "依据", "输入", "输出", "文件", "路径", "字段", "单位",
		"缺失", "异常", "假设", "变量", "参数", "约束", "目标函数", "指标",
		"验证", "误差", "敏感性", "图表", "结论", "审计", "修改", "章节",
		"公式", "引用", "交接", "checkpoint",
	})
	hasArtifact := containsAny(lower, []string{
		"```", ".csv", ".xlsx", ".xls", ".json", ".py", ".ipynb", ".png",
		".jpg", ".jpeg", ".svg", ".tex", ".pdf", ".docx", "result.json",
		"notebook",
	})
	return !(hasHeading && (hasEvidence || hasArtifact))
}

func containsAny(s string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

// FactBodyTemplate 按 category 返回建议的 body Markdown 骨架（供 Agent 填入真实内容）。
func FactBodyTemplate(category, factKey string) string {
	switch inferFactCategory(category, factKey) {
	case FactCategoryProblem:
		return problemFactBodyTemplate
	case FactCategoryData:
		return dataFactBodyTemplate
	case FactCategoryModel:
		return modelFactBodyTemplate
	case FactCategoryCode:
		return codeFactBodyTemplate
	case FactCategoryResult:
		return resultFactBodyTemplate
	case FactCategoryFigure:
		return figureFactBodyTemplate
	case FactCategoryPaper:
		return paperFactBodyTemplate
	case FactCategoryReview:
		return reviewFactBodyTemplate
	default:
		return noteFactBodyTemplate
	}
}

func inferFactCategory(category, factKey string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	if _, ok := structuredModelingCategories[c]; ok || c == FactCategoryNote {
		return c
	}
	key := strings.ToLower(strings.TrimSpace(factKey))
	for _, prefix := range structuredModelingPrefixes {
		if strings.HasPrefix(key, prefix) {
			return strings.TrimSuffix(prefix, "/")
		}
	}
	return FactCategoryNote
}

const problemFactBodyTemplate = `## 题目与任务
- 背景: <赛题背景或业务场景>
- 问题列表: <P1 / P2 / P3 ...>
- 交付要求: <论文、图表、预测结果、附件等>

## 约束与评分点
- 显式约束: <时间、空间、单位、格式、边界条件>
- 隐式约束: <可解释性、稳定性、可复现性>
- 评分风险: <容易扣分或走偏之处>

## 已知数据与资料
- 附件: <文件路径、表名、字段概览>
- 外部资料: <允许使用的来源或引用>

## 待确认问题
- <需要用户或后续代理确认的问题>`

const dataFactBodyTemplate = `## 数据概览
- 文件/表: <路径、sheet、行列规模>
- 字段与单位: <字段解释、单位、取值范围>
- 样本粒度: <时间、空间、对象、观测频率>

## 数据质量审计
- 缺失: <字段、比例、处理建议>
- 异常: <异常规则、影响范围>
- 重复/泄露: <重复记录、未来信息、目标泄露风险>

## 可建模性结论
- 可直接使用: <字段或子集>
- 需要清洗/派生: <步骤与责任人>
- 不建议使用: <原因>

## 证据
- 文件路径/脚本/输出: <profile.json、图表路径、审计代码>`

const modelFactBodyTemplate = `## 模型规格
- 目标: <优化、预测、分类、评价、仿真等>
- 变量: <决策变量、状态变量、随机变量>
- 参数: <来源、估计方法、默认值>
- 假设: <编号列出，说明合理性与局限>

## 数学表达
- 目标函数/评价指标: <公式或文字>
- 约束条件: <公式或文字>
- 求解策略: <解析、数值优化、仿真、机器学习等>

## 备选模型与取舍
- 候选模型: <模型 A/B/C>
- 选择理由: <准确性、解释性、时间成本>
- 放弃理由: <数据不足、复杂度、不可验证>

## 验证方案
- 基线: <简单模型或规则>
- 敏感性/稳健性: <扰动参数、交叉验证、Bootstrap 等>
- 通过标准: <误差、排名稳定性、可行性>`

const codeFactBodyTemplate = `## 实现入口
- 脚本/Notebook: <路径>
- 命令: <运行命令>
- 输入: <数据文件、配置文件>
- 输出: <result.json、图表、日志>

## 实现说明
- 方法: <调用的方法卡或算法>
- 关键参数: <参数名、值、来源>
- 随机性控制: <seed、版本、环境>

## 运行状态
- 是否可复现: <是/否>
- 已知问题: <失败、性能、边界情况>
- 下一步: <需要建模手/论文手确认的点>`

const resultFactBodyTemplate = `## 结果摘要
- 对应问题: <P1 / P2 / P3>
- 主要结论: <数值、排名、预测、策略>
- 输出文件: <result.json、表格、日志路径>

## 指标与误差
- 评价指标: <MAE、RMSE、准确率、目标函数值等>
- 基线对照: <基线结果与差异>
- 不确定性: <置信区间、波动范围>

## 可解释性与限制
- 解释: <为什么得到该结果>
- 限制: <数据、假设、模型适用边界>
- 论文引用位置: <章节或图表编号>`

const figureFactBodyTemplate = `## 图表信息
- 图表编号/标题: <Figure/Table 编号>
- 文件路径: <png/pdf/svg/csv>
- 对应问题: <P1 / P2 / P3>

## 数据来源
- 输入数据: <数据文件或 result.json 字段>
- 生成脚本: <脚本/Notebook 路径>
- 关键参数: <绘图参数、筛选条件>

## 论文使用
- 结论句: <图表支持的结论>
- 需要检查: <单位、图例、坐标、中文字体、清晰度>`

const paperFactBodyTemplate = `## 论文章节
- 章节: <摘要、问题重述、模型建立、求解、验证等>
- LaTeX 文件: <paper.tex 或章节 tex 路径>
- 来源卡片: <problem/data/model/result/figure fact_key>

## 内容要点
- 必写结论: <由已验证结果支撑>
- 公式/图表: <编号、路径、引用 key>
- 引用: <bib key 或来源>

## 格式状态
- 官方 Word 模板对齐: <是/否/待处理>
- PDF 编译: <通过/失败/未运行>
- DOCX 导出: <通过/失败/未运行>`

const reviewFactBodyTemplate = `## 评审对象
- 范围: <题目、数据、模型、代码、结果、论文>
- 输入: <相关 fact_key、文件路径>
- 评审人/代理: <角色>

## 发现的问题
- 阻断问题: <必须返工的问题>
- 一般问题: <建议修订的问题>
- 已确认无问题: <检查过的项目>

## 处理决议
- 责任人: <建模/编程/论文>
- 修订动作: <具体修改>
- checkpoint: <允许进入下一阶段/需要返工>`

const noteFactBodyTemplate = `## 摘要
<该备注的核心认知>

## 细节
<补充说明、上下文、讨论记录>

## 来源与关联
- 来源: <对话、文件、工具输出>
- 相关 fact_key: <可选>`

// FactRecordingGuidanceBlock 写入系统提示：要求事实沉淀建模卡片上下文而非仅结论。
func FactRecordingGuidanceBlock() string {
	return `### 黑板卡片写入规范（数学建模 / 竞赛协作）

- **summary**：索引用一行，须含「什么 + 属于哪个问题/文件/模型 + 如何验证或使用」要点，禁止只写结论。
- **body**：结构化建模卡片，写入 ` + "`upsert_project_fact`" + ` 的 body 字段；索引不含 body，后续会话须靠 ` + "`get_project_fact`" + ` 取回。
- **category / fact_key 建议**：
  - ` + "`problem/`" + `：题意、任务拆解、约束、交付要求。
  - ` + "`data/`" + `：附件、字段、单位、缺失异常、清洗建议、数据泄露风险。
  - ` + "`model/`" + `：假设、变量、参数、目标函数、约束、候选模型和验证方案。
  - ` + "`code/`" + `：脚本、Notebook、命令、输入输出、随机种子、运行环境。
  - ` + "`result/`" + ` / ` + "`figure/`" + `：结果 JSON、指标、图表路径、基线对照、敏感性结论。
  - ` + "`paper/`" + `：LaTeX 章节、公式图表引用、官方模板对齐、PDF/DOCX 导出状态。
  - ` + "`review/`" + `：交叉评审、阻断问题、返工决议、checkpoint。
- 更新同一认知时保持相同 ` + "`fact_key`" + ` 覆盖写入，勿散落多个 key 导致上下文丢失。`
}

// SparseBodyWarning 结构化建模事实 body 不足时的工具返回提示（不阻断保存）。
func SparseBodyWarning(category, factKey string) string {
	if !IsSparseFactBody(category, factKey, "") {
		return ""
	}
	return fmt.Sprintf(
		"\n\n提示：category=%q / fact_key=%q 属于结构化建模卡片，但 body 为空或过简。请补充题意、数据、模型、代码、结果、论文或评审所需的证据与产物路径。\n建议 body 骨架：\n%s",
		category, factKey, FactBodyTemplate(category, factKey),
	)
}

// SparseBodyWarningIfNeeded 根据实际 body 判断是否追加警告。
func SparseBodyWarningIfNeeded(category, factKey, body string) string {
	if !IsSparseFactBody(category, factKey, body) {
		return ""
	}
	return SparseBodyWarning(category, factKey)
}
