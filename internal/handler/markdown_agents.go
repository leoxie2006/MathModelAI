package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mathmodel-ai/internal/agents"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

var markdownAgentFilenameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*\.md$`)

type markdownAgentUpsertRequest struct {
	Filename      string   `json:"filename,omitempty"`
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	Instruction   string   `json:"instruction,omitempty"`
	BindRole      string   `json:"bind_role,omitempty"`
	MaxIterations int      `json:"max_iterations,omitempty"`
	Kind          string   `json:"kind,omitempty"`
}

func (h *ConfigHandler) resolveMarkdownAgentsDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dir := ""
	if h.config != nil {
		dir = strings.TrimSpace(h.config.AgentsDir)
	}
	if dir == "" {
		dir = "agents"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	base := "."
	if strings.TrimSpace(h.configPath) != "" {
		base = filepath.Dir(h.configPath)
	}
	return filepath.Clean(filepath.Join(base, dir))
}

func validateMarkdownAgentFilename(filename string) (string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", fmt.Errorf("文件名不能为空")
	}
	if filepath.Base(name) != name {
		return "", fmt.Errorf("文件名不能包含路径")
	}
	if !markdownAgentFilenameRE.MatchString(name) {
		return "", fmt.Errorf("文件名须为 .md，且仅含字母、数字、._-")
	}
	if strings.EqualFold(name, "README.md") {
		return "", fmt.Errorf("README.md 不能作为 Agent 文件")
	}
	return name, nil
}

func defaultMarkdownAgentFilename(req markdownAgentUpsertRequest) string {
	base := strings.TrimSpace(req.ID)
	if base == "" {
		base = strings.TrimSpace(req.Name)
	}
	return agents.SlugID(base) + ".md"
}

func marshalMarkdownAgent(req markdownAgentUpsertRequest) ([]byte, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("请填写显示名称")
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = agents.SlugID(name)
	}
	kind := strings.TrimSpace(req.Kind)
	if kind != "" && !strings.EqualFold(kind, "orchestrator") {
		return nil, fmt.Errorf("kind 仅支持 orchestrator 或留空")
	}
	fm := agents.FrontMatter{
		Name:          name,
		ID:            id,
		Description:   strings.TrimSpace(req.Description),
		Tools:         req.Tools,
		MaxIterations: req.MaxIterations,
		BindRole:      strings.TrimSpace(req.BindRole),
	}
	if strings.EqualFold(kind, "orchestrator") {
		fm.Kind = "orchestrator"
	}
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(req.Instruction)
	out := "---\n" + strings.TrimSpace(string(fmBytes)) + "\n---\n"
	if body != "" {
		out += "\n" + body + "\n"
	}
	return []byte(out), nil
}

func markdownAgentResponse(filename string, file agents.FileAgent) gin.H {
	cfg := file.Config
	return gin.H{
		"filename":        filename,
		"id":              cfg.ID,
		"name":            cfg.Name,
		"description":     cfg.Description,
		"tools":           cfg.RoleTools,
		"bind_role":       cfg.BindRole,
		"max_iterations":  cfg.MaxIterations,
		"kind":            cfg.Kind,
		"instruction":     cfg.Instruction,
		"is_orchestrator": file.IsOrchestrator,
	}
}

// ListMarkdownAgents lists agents/*.md files for the Agent management page.
func (h *ConfigHandler) ListMarkdownAgents(c *gin.Context) {
	dir := h.resolveMarkdownAgentsDir()
	files, err := agents.LoadMarkdownAgentFiles(dir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(files))
	for _, f := range files {
		items = append(items, markdownAgentResponse(f.Filename, f))
	}
	c.JSON(http.StatusOK, gin.H{"dir": dir, "agents": items})
}

// GetMarkdownAgent returns one agents/*.md file parsed into editable fields.
func (h *ConfigHandler) GetMarkdownAgent(c *gin.Context) {
	filename, err := validateMarkdownAgentFilename(c.Param("filename"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dir := h.resolveMarkdownAgentsDir()
	path := filepath.Join(dir, filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent 文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg, err := agents.ParseMarkdownSubAgent(filename, string(raw))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, markdownAgentResponse(filename, agents.FileAgent{
		Filename:       filename,
		Config:         cfg,
		IsOrchestrator: agents.IsOrchestratorLikeMarkdown(filename, cfg.Kind),
	}))
}

// CreateMarkdownAgent creates a new agents/*.md file.
func (h *ConfigHandler) CreateMarkdownAgent(c *gin.Context) {
	var req markdownAgentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = defaultMarkdownAgentFilename(req)
	}
	filename, err := validateMarkdownAgentFilename(filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := marshalMarkdownAgent(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := agents.ParseMarkdownSubAgent(filename, string(content)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dir := h.resolveMarkdownAgentsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Agent 文件已存在"})
		return
	} else if !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := agents.LoadMarkdownAgentsDir(dir); err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"filename": filename})
}

// UpdateMarkdownAgent overwrites one existing agents/*.md file.
func (h *ConfigHandler) UpdateMarkdownAgent(c *gin.Context) {
	filename, err := validateMarkdownAgentFilename(c.Param("filename"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req markdownAgentUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	content, err := marshalMarkdownAgent(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := agents.ParseMarkdownSubAgent(filename, string(content)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dir := h.resolveMarkdownAgentsDir()
	path := filepath.Join(dir, filename)
	old, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent 文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := agents.LoadMarkdownAgentsDir(dir); err != nil {
		_ = os.WriteFile(path, old, 0644)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"filename": filename})
}

// DeleteMarkdownAgent removes one agents/*.md file.
func (h *ConfigHandler) DeleteMarkdownAgent(c *gin.Context) {
	filename, err := validateMarkdownAgentFilename(c.Param("filename"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Join(h.resolveMarkdownAgentsDir(), filename)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent 文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"filename": filename})
}
