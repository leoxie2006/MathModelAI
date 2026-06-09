package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mathmodel-ai/internal/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const maxProjectUploadBytes = 512 << 20

var projectWorkspaceDirs = []string{"attachments", "data", "code", "outputs", "paper", "cards"}

func defaultProjectWorkspaceRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "workspaces"
	}
	return filepath.Join(cwd, "workspaces")
}

func (h *ProjectHandler) projectWorkspaceRoot() string {
	if h == nil || strings.TrimSpace(h.workspaceRoot) == "" {
		return defaultProjectWorkspaceRoot()
	}
	return h.workspaceRoot
}

func (h *ProjectHandler) projectWorkspaceDir(projectID string) string {
	return filepath.Join(h.projectWorkspaceRoot(), sanitizePathSegment(projectID))
}

func (h *ProjectHandler) ensureProjectWorkspace(projectID string) (string, error) {
	root := h.projectWorkspaceDir(projectID)
	for _, dir := range projectWorkspaceDirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			return "", err
		}
	}
	return root, nil
}

func (h *ProjectHandler) removeProjectWorkspace(projectID string) {
	root := h.projectWorkspaceDir(projectID)
	if strings.TrimSpace(projectID) == "" || root == "" {
		return
	}
	if err := os.RemoveAll(root); err != nil && h.logger != nil {
		h.logger.Warn("删除项目工作空间失败", zap.String("projectId", projectID), zap.String("dir", root), zap.Error(err))
	}
}

func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unnamed"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\x00", "")
	out := replacer.Replace(s)
	out = strings.Trim(out, ". ")
	if out == "" {
		out = "unnamed"
	}
	return out
}

func normalizeProjectFileType(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "attachment" || s == "problem" || s == "template" {
		return "attachments"
	}
	if s == "output" || s == "result" || s == "figure" || s == "figures" {
		return "outputs"
	}
	for _, allowed := range projectWorkspaceDirs {
		if s == allowed {
			return s
		}
	}
	return "attachments"
}

func inferProjectFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv", ".tsv", ".xlsx", ".xls", ".json", ".parquet", ".db", ".sqlite":
		return "data"
	case ".py", ".ipynb", ".r", ".m", ".jl", ".sql":
		return "code"
	case ".png", ".jpg", ".jpeg", ".svg", ".gif", ".log":
		return "outputs"
	case ".tex", ".bib", ".cls", ".sty", ".docx":
		return "paper"
	case ".md":
		return "cards"
	default:
		return "attachments"
	}
}

func uniqueStoredName(original string) string {
	base := filepath.Base(strings.TrimSpace(original))
	if base == "" || base == "." {
		base = "file"
	}
	base = sanitizePathSegment(base)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "file"
	}
	return fmt.Sprintf("%s_%s_%s%s", name, time.Now().Format("20060102-150405"), randomHex(4), ext)
}

func randomHex(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func relPathForWorkspace(root, abs string) (string, error) {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", err
	}
	absClean, err := filepath.Abs(filepath.Clean(abs))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, absClean)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path outside workspace")
	}
	return filepath.ToSlash(rel), nil
}

func safeWorkspacePath(root, rel string) (string, string, error) {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if rel == "" {
		return "", "", fmt.Errorf("relative path required")
	}
	rel = strings.TrimLeft(rel, "/")
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." || filepath.IsAbs(clean) {
		return "", "", fmt.Errorf("invalid relative path")
	}
	first := strings.Split(filepath.ToSlash(clean), "/")[0]
	if normalizeProjectFileType(first) != first {
		return "", "", fmt.Errorf("relative path must start with one of: %s", strings.Join(projectWorkspaceDirs, ", "))
	}
	abs := filepath.Join(root, clean)
	relClean, err := relPathForWorkspace(root, abs)
	if err != nil {
		return "", "", err
	}
	return abs, relClean, nil
}

func sha256File(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func mimeForPath(path string, fallback string) string {
	if s := strings.TrimSpace(fallback); s != "" {
		return s
	}
	if m := mime.TypeByExtension(filepath.Ext(path)); m != "" {
		return m
	}
	return "application/octet-stream"
}

func (h *ProjectHandler) buildProjectFileRecord(projectID, root, absPath, originalName, fileType, purpose, mimeType, source, note string) (*database.ProjectFile, error) {
	rel, err := relPathForWorkspace(root, absPath)
	if err != nil {
		return nil, err
	}
	hash, size, err := sha256File(absPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(fileType) == "" {
		parts := strings.Split(rel, "/")
		if len(parts) > 1 {
			fileType = normalizeProjectFileType(parts[0])
		} else {
			fileType = inferProjectFileType(absPath)
		}
	}
	stored := filepath.Base(absPath)
	return &database.ProjectFile{
		ProjectID:    projectID,
		OriginalName: originalName,
		StoredName:   stored,
		RelPath:      rel,
		AbsPath:      absPath,
		FileType:     normalizeProjectFileType(fileType),
		Purpose:      strings.TrimSpace(purpose),
		MimeType:     mimeForPath(absPath, mimeType),
		SizeBytes:    size,
		SHA256:       hash,
		Source:       strings.TrimSpace(source),
		Note:         strings.TrimSpace(note),
	}, nil
}

// ListProjectFiles GET /api/projects/:id/files
func (h *ProjectHandler) ListProjectFiles(c *gin.Context) {
	projectID := c.Param("id")
	if _, err := h.db.GetProject(projectID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	root, err := h.ensureProjectWorkspace(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	limit := 200
	if q := strings.TrimSpace(c.Query("limit")); q != "" {
		if parsed, perr := parsePositiveInt(q, 200); perr == nil {
			limit = parsed
		}
	}
	offset := 0
	if q := strings.TrimSpace(c.Query("offset")); q != "" {
		if parsed, perr := parsePositiveInt(q, 0); perr == nil {
			offset = parsed
		}
	}
	fileType := normalizeOptionalProjectFileType(c.Query("type"))
	files, err := h.db.ListProjectFiles(projectID, fileType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if files == nil {
		files = []*database.ProjectFile{}
	}
	c.JSON(http.StatusOK, gin.H{
		"files":          files,
		"workspace_root": root,
		"dirs":           projectWorkspaceDirs,
	})
}

func parsePositiveInt(s string, fallback int) (int, error) {
	var out int
	_, err := fmt.Sscanf(s, "%d", &out)
	if err != nil || out < 0 {
		return fallback, err
	}
	return out, nil
}

func normalizeOptionalProjectFileType(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return normalizeProjectFileType(raw)
}

// UploadProjectFile POST /api/projects/:id/files
func (h *ProjectHandler) UploadProjectFile(c *gin.Context) {
	projectID := c.Param("id")
	if _, err := h.db.GetProject(projectID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProjectUploadBytes)
	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少上传文件"})
		return
	}
	fileType := normalizeProjectFileType(firstNonEmpty(c.PostForm("file_type"), c.PostForm("type"), c.PostForm("purpose")))
	purpose := c.PostForm("purpose")
	note := c.PostForm("note")
	root, err := h.ensureProjectWorkspace(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	storedName := uniqueStoredName(header.Filename)
	dest := filepath.Join(root, fileType, storedName)
	if err := c.SaveUploadedFile(header, dest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.buildProjectFileRecord(projectID, root, dest, header.Filename, fileType, purpose, header.Header.Get("Content-Type"), "upload", note)
	if err != nil {
		_ = os.Remove(dest)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	saved, err := h.db.UpsertProjectFile(rec)
	if err != nil {
		_ = os.Remove(dest)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"file": saved, "workspace_root": root})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type writeProjectFileRequest struct {
	RelPath  string `json:"rel_path" binding:"required"`
	Content  string `json:"content"`
	FileType string `json:"file_type"`
	Purpose  string `json:"purpose"`
	Note     string `json:"note"`
}

// WriteProjectFile POST /api/projects/:id/files/text
func (h *ProjectHandler) WriteProjectFile(c *gin.Context) {
	projectID := c.Param("id")
	if _, err := h.db.GetProject(projectID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	var req writeProjectFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	root, err := h.ensureProjectWorkspace(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	abs, rel, err := safeWorkspacePath(root, req.RelPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fileType := normalizeProjectFileType(req.FileType)
	if strings.TrimSpace(req.FileType) == "" {
		fileType = normalizeProjectFileType(strings.Split(rel, "/")[0])
	}
	rec, err := h.buildProjectFileRecord(projectID, root, abs, filepath.Base(abs), fileType, req.Purpose, "", "generated", req.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	saved, err := h.db.UpsertProjectFile(rec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"file": saved, "workspace_root": root})
}

// DownloadProjectFile GET /api/projects/:id/files/:fileId/download
func (h *ProjectHandler) DownloadProjectFile(c *gin.Context) {
	file, err := h.db.GetProjectFile(c.Param("id"), c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目文件不存在"})
		return
	}
	if _, err := os.Stat(file.AbsPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.FileAttachment(file.AbsPath, file.OriginalName)
}

// DeleteProjectFile DELETE /api/projects/:id/files/:fileId
func (h *ProjectHandler) DeleteProjectFile(c *gin.Context) {
	projectID := c.Param("id")
	file, err := h.db.GetProjectFile(projectID, c.Param("fileId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目文件不存在"})
		return
	}
	if err := os.Remove(file.AbsPath); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.DeleteProjectFile(projectID, file.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ScanProjectWorkspace POST /api/projects/:id/files/scan
func (h *ProjectHandler) ScanProjectWorkspace(c *gin.Context) {
	projectID := c.Param("id")
	if _, err := h.db.GetProject(projectID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
		return
	}
	root, err := h.ensureProjectWorkspace(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var files []*database.ProjectFile
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := relPathForWorkspace(root, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(rel, "/")
		if len(parts) < 2 {
			return nil
		}
		fileType := normalizeProjectFileType(parts[0])
		rec, err := h.buildProjectFileRecord(projectID, root, path, filepath.Base(path), fileType, "", "", "scan", "")
		if err != nil {
			return nil
		}
		saved, err := h.db.UpsertProjectFile(rec)
		if err != nil {
			return nil
		}
		files = append(files, saved)
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})
	c.JSON(http.StatusOK, gin.H{"files": files, "count": len(files), "workspace_root": root})
}

// UploadChatAttachment POST /api/chat-uploads
func (h *AgentHandler) UploadChatAttachment(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxProjectUploadBytes)
	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少上传文件"})
		return
	}
	conversationID := strings.TrimSpace(c.PostForm("conversationId"))
	if conversationID == "" {
		conversationID = "_manual"
	}
	cwd, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	convDirName := strings.ReplaceAll(conversationID, string(filepath.Separator), "_")
	targetDir := filepath.Join(cwd, chatUploadsDirName, time.Now().Format("2006-01-02"), convDirName)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	storedName := uniqueStoredName(header.Filename)
	dest := filepath.Join(targetDir, storedName)
	if err := c.SaveUploadedFile(header, dest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	abs, _ := filepath.Abs(dest)
	_, size, _ := sha256File(abs)
	c.JSON(http.StatusOK, gin.H{
		"fileName":     header.Filename,
		"storedName":   storedName,
		"absolutePath": abs,
		"size":         size,
		"mimeType":     header.Header.Get("Content-Type"),
	})
}
