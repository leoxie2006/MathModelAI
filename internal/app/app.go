package app

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mathmodel-ai/internal/agent"
	"mathmodel-ai/internal/audit"
	"mathmodel-ai/internal/config"
	"mathmodel-ai/internal/database"
	"mathmodel-ai/internal/einoobserve"
	"mathmodel-ai/internal/handler"
	"mathmodel-ai/internal/knowledge"
	"mathmodel-ai/internal/logger"
	"mathmodel-ai/internal/mcp"
	"mathmodel-ai/internal/security"
	"mathmodel-ai/internal/skillpackage"
	"mathmodel-ai/internal/storage"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// App 应用
type App struct {
	config             *config.Config
	logger             *logger.Logger
	router             *gin.Engine
	mcpServer          *mcp.Server
	externalMCPMgr     *mcp.ExternalMCPManager
	agent              *agent.Agent
	executor           *security.Executor
	db                 *database.DB
	knowledgeDB        *database.DB              // 知识库数据库连接（如果使用独立数据库）
	knowledgeManager   *knowledge.Manager        // 知识库管理器（用于动态初始化）
	knowledgeRetriever *knowledge.Retriever      // 知识库检索器（用于动态初始化）
	knowledgeIndexer   *knowledge.Indexer        // 知识库索引器（用于动态初始化）
	knowledgeHandler   *handler.KnowledgeHandler // 知识库处理器（用于动态初始化）
	agentHandler       *handler.AgentHandler     // Agent处理器（用于更新知识库管理器）
	auditSvc           *audit.Service
}

// New 创建新应用
func New(cfg *config.Config, log *logger.Logger, configPath string) (*App, error) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// CORS中间件
	router.Use(corsMiddleware())

	// 初始化数据库
	dbPath := cfg.Database.Path
	if dbPath == "" {
		dbPath = "data/conversations.db"
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := database.NewDB(dbPath, log.Logger)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	auditSvc := audit.NewService(db, cfg, log.Logger)
	audit.RegisterConversationCreateHook(auditSvc)
	auditSvc.PurgeExpired()
	audit.StartRetentionLoop(auditSvc, log.Logger)

	// 创建MCP服务器（带数据库持久化）
	mcpServer := mcp.NewServerWithStorage(log.Logger, db)
	mcpServer.ConfigureHTTPToolCallTimeoutFromAgentMinutes(cfg.Agent.ToolTimeoutMinutes)

	// 创建安全工具执行器
	executor := security.NewExecutor(&cfg.Security, mcpServer, log.Logger)

	// 注册工具
	executor.RegisterTools(mcpServer)

	registerProjectFactTools(mcpServer, db, cfg, log.Logger)

	// 创建外部MCP管理器（使用与内部MCP服务器相同的存储）
	externalMCPMgr := mcp.NewExternalMCPManagerWithStorage(log.Logger, db)
	if cfg.ExternalMCP.Servers != nil {
		externalMCPMgr.LoadConfigs(&cfg.ExternalMCP)
		// 启动所有启用的外部MCP客户端
		externalMCPMgr.StartAllEnabled()
	}

	// 初始化结果存储
	resultStorageDir := "tmp"
	if cfg.Agent.ResultStorageDir != "" {
		resultStorageDir = cfg.Agent.ResultStorageDir
	}

	// 确保存储目录存在
	if err := os.MkdirAll(resultStorageDir, 0755); err != nil {
		return nil, fmt.Errorf("创建结果存储目录失败: %w", err)
	}

	// 创建结果存储实例
	resultStorage, err := storage.NewFileResultStorage(resultStorageDir, log.Logger)
	if err != nil {
		return nil, fmt.Errorf("初始化结果存储失败: %w", err)
	}

	// 创建Agent
	maxIterations := cfg.Agent.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 30 // 默认值
	}
	agent := agent.NewAgent(&cfg.OpenAI, &cfg.Agent, mcpServer, externalMCPMgr, log.Logger, maxIterations)
	agent.UpdateToolDescriptionMode(cfg.Security.ToolDescriptionMode)

	// 设置结果存储到Agent
	agent.SetResultStorage(resultStorage)

	// 设置结果存储到Executor（用于查询工具）
	executor.SetResultStorage(resultStorage)

	// 初始化知识库模块（如果启用）
	var knowledgeManager *knowledge.Manager
	var knowledgeRetriever *knowledge.Retriever
	var knowledgeIndexer *knowledge.Indexer
	var knowledgeHandler *handler.KnowledgeHandler

	var knowledgeDBConn *database.DB
	log.Logger.Info("检查知识库配置", zap.Bool("enabled", cfg.Knowledge.Enabled))
	if cfg.Knowledge.Enabled {
		// 确定知识库数据库路径
		knowledgeDBPath := cfg.Database.KnowledgeDBPath
		var knowledgeDB *sql.DB

		if knowledgeDBPath != "" {
			// 使用独立的知识库数据库
			// 确保目录存在
			if err := os.MkdirAll(filepath.Dir(knowledgeDBPath), 0755); err != nil {
				return nil, fmt.Errorf("创建知识库数据库目录失败: %w", err)
			}

			var err error
			knowledgeDBConn, err = database.NewKnowledgeDB(knowledgeDBPath, log.Logger)
			if err != nil {
				return nil, fmt.Errorf("初始化知识库数据库失败: %w", err)
			}
			knowledgeDB = knowledgeDBConn.DB
			log.Logger.Info("使用独立的知识库数据库", zap.String("path", knowledgeDBPath))
		} else {
			// 向后兼容：使用会话数据库
			knowledgeDB = db.DB
			log.Logger.Info("使用会话数据库存储知识库数据（建议配置knowledge_db_path以分离数据）")
		}

		// 创建知识库管理器
		knowledgeManager = knowledge.NewManager(knowledgeDB, cfg.Knowledge.BasePath, log.Logger)

		// 创建嵌入器
		// 使用OpenAI配置的API Key（如果知识库配置中没有指定）
		if cfg.Knowledge.Embedding.APIKey == "" {
			cfg.Knowledge.Embedding.APIKey = cfg.OpenAI.APIKey
		}
		if cfg.Knowledge.Embedding.BaseURL == "" {
			cfg.Knowledge.Embedding.BaseURL = cfg.OpenAI.BaseURL
		}

		embedder, err := knowledge.NewEmbedder(context.Background(), &cfg.Knowledge, &cfg.OpenAI, log.Logger)
		if err != nil {
			return nil, fmt.Errorf("初始化知识库嵌入器失败: %w", err)
		}

		// 创建检索器
		retrievalConfig := &knowledge.RetrievalConfig{
			TopK:                cfg.Knowledge.Retrieval.TopK,
			SimilarityThreshold: cfg.Knowledge.Retrieval.SimilarityThreshold,
			SubIndexFilter:      cfg.Knowledge.Retrieval.SubIndexFilter,
			PostRetrieve:        cfg.Knowledge.Retrieval.PostRetrieve,
		}
		knowledgeRetriever = knowledge.NewRetriever(knowledgeDB, embedder, retrievalConfig, log.Logger)

		// 创建索引器（Eino Compose 链）
		knowledgeIndexer, err = knowledge.NewIndexer(context.Background(), knowledgeDB, embedder, log.Logger, &cfg.Knowledge)
		if err != nil {
			return nil, fmt.Errorf("初始化知识库索引器失败: %w", err)
		}

		// 注册知识检索工具到MCP服务器
		knowledge.RegisterKnowledgeTool(mcpServer, knowledgeRetriever, knowledgeManager, log.Logger)

		// 创建知识库API处理器
		knowledgeHandler = handler.NewKnowledgeHandler(knowledgeManager, knowledgeRetriever, knowledgeIndexer, db, log.Logger)
		knowledgeHandler.SetAudit(auditSvc)
		log.Logger.Info("知识库模块初始化完成", zap.Bool("handler_created", knowledgeHandler != nil))

		// 扫描知识库并建立索引（异步）
		go func() {
			itemsToIndex, err := knowledgeManager.ScanKnowledgeBase()
			if err != nil {
				log.Logger.Warn("扫描知识库失败", zap.Error(err))
				return
			}

			// 检查是否已有索引
			hasIndex, err := knowledgeIndexer.HasIndex()
			if err != nil {
				log.Logger.Warn("检查索引状态失败", zap.Error(err))
				return
			}

			if hasIndex {
				// 如果已有索引，只索引新添加或更新的项
				if len(itemsToIndex) > 0 {
					log.Logger.Info("检测到已有知识库索引，开始增量索引", zap.Int("count", len(itemsToIndex)))
					ctx := context.Background()
					consecutiveFailures := 0
					var firstFailureItemID string
					var firstFailureError error
					failedCount := 0

					for _, itemID := range itemsToIndex {
						if err := knowledgeIndexer.IndexItem(ctx, itemID); err != nil {
							failedCount++
							consecutiveFailures++

							if consecutiveFailures == 1 {
								firstFailureItemID = itemID
								firstFailureError = err
								log.Logger.Warn("索引知识项失败", zap.String("itemId", itemID), zap.Error(err))
							}

							// 如果连续失败2次，立即停止增量索引
							if consecutiveFailures >= 2 {
								log.Logger.Error("连续索引失败次数过多，立即停止增量索引",
									zap.Int("consecutiveFailures", consecutiveFailures),
									zap.Int("totalItems", len(itemsToIndex)),
									zap.String("firstFailureItemId", firstFailureItemID),
									zap.Error(firstFailureError),
								)
								break
							}
							continue
						}

						// 成功时重置连续失败计数
						if consecutiveFailures > 0 {
							consecutiveFailures = 0
							firstFailureItemID = ""
							firstFailureError = nil
						}
					}
					log.Logger.Info("增量索引完成", zap.Int("totalItems", len(itemsToIndex)), zap.Int("failedCount", failedCount))
				} else {
					log.Logger.Info("检测到已有知识库索引，没有需要索引的新项或更新项")
				}
				return
			}

			// 只有在没有索引时才自动重建
			log.Logger.Info("未检测到知识库索引，开始自动构建索引")
			ctx := context.Background()
			if err := knowledgeIndexer.RebuildIndex(ctx); err != nil {
				log.Logger.Warn("重建知识库索引失败", zap.Error(err))
			}
		}()
	}

	// 配置文件路径必须由入口传入（与 flag -config 一致）。勿再用 os.Args[1]，否则 ./mathmodel-ai --https 会把 --https 当成路径。
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		configPath = "config.yaml"
	}

	skillsDir := skillpackage.SkillsRootFromConfig(cfg.SkillsDir, configPath)
	log.Logger.Info("Skills 目录（Eino ADK skill 中间件 + Web 管理 API）", zap.String("skillsDir", skillsDir))
	configDir := filepath.Dir(configPath)
	agent.SetPromptBaseDir(configDir)

	agentsDir := cfg.AgentsDir
	if agentsDir == "" {
		agentsDir = "agents"
	}
	if !filepath.IsAbs(agentsDir) {
		agentsDir = filepath.Join(configDir, agentsDir)
	}
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		log.Logger.Warn("创建 agents 目录失败", zap.String("path", agentsDir), zap.Error(err))
	}
	log.Logger.Info("多代理 Markdown 子 Agent 目录", zap.String("agentsDir", agentsDir))

	// 创建处理器
	agentHandler := handler.NewAgentHandler(agent, db, cfg, log.Logger)
	agentHandler.SetAudit(auditSvc)
	agentHandler.SetAgentsMarkdownDir(agentsDir)
	// 如果知识库已启用，设置知识库管理器到AgentHandler以便记录检索日志
	if knowledgeManager != nil {
		agentHandler.SetKnowledgeManager(knowledgeManager)
	}
	groupHandler := handler.NewGroupHandler(db, log.Logger)
	projectHandler := handler.NewProjectHandler(db, log.Logger)
	configHandler := handler.NewConfigHandler(configPath, cfg, mcpServer, executor, agent, externalMCPMgr, log.Logger)
	configHandler.SetAudit(auditSvc)
	externalMCPHandler := handler.NewExternalMCPHandler(externalMCPMgr, cfg, configPath, log.Logger)
	externalMCPHandler.SetAudit(auditSvc)
	roleHandler := handler.NewRoleHandler(cfg, configPath, log.Logger)
	roleHandler.SetAudit(auditSvc)
	skillsHandler := handler.NewSkillsHandler(cfg, configPath, log.Logger)
	skillsHandler.SetAudit(auditSvc)
	if db != nil {
		skillsHandler.SetDB(db) // 设置数据库连接以便获取调用统计
	}

	conversationHandler := handler.NewConversationHandler(db, log.Logger)
	conversationHandler.SetAudit(auditSvc)
	// 创建 App 实例（部分字段稍后填充）
	app := &App{
		config:             cfg,
		logger:             log,
		router:             router,
		mcpServer:          mcpServer,
		externalMCPMgr:     externalMCPMgr,
		agent:              agent,
		executor:           executor,
		db:                 db,
		knowledgeDB:        knowledgeDBConn,
		knowledgeManager:   knowledgeManager,
		knowledgeRetriever: knowledgeRetriever,
		knowledgeIndexer:   knowledgeIndexer,
		knowledgeHandler:   knowledgeHandler,
		agentHandler:       agentHandler,
		auditSvc:           auditSvc,
	}

	// Skills 由 Eino ADK skill 中间件提供（多代理）；此处不注册 MCP 形态的技能工具
	configHandler.SetSkillsToolRegistrar(func() error { return nil })

	// 设置知识库初始化器（用于动态初始化，需要在 App 创建后设置）
	configHandler.SetKnowledgeInitializer(func() (*handler.KnowledgeHandler, error) {
		knowledgeHandler, err := initializeKnowledge(cfg, db, knowledgeDBConn, mcpServer, agentHandler, app, log.Logger)
		if err != nil {
			return nil, err
		}

		// 动态初始化后，设置知识库工具注册器和检索器更新器
		// 这样后续 ApplyConfig 时就能重新注册工具了
		if app.knowledgeRetriever != nil && app.knowledgeManager != nil {
			// 创建闭包，捕获knowledgeRetriever和knowledgeManager的引用
			registrar := func() error {
				knowledge.RegisterKnowledgeTool(mcpServer, app.knowledgeRetriever, app.knowledgeManager, log.Logger)
				return nil
			}
			configHandler.SetKnowledgeToolRegistrar(registrar)
			// 设置检索器更新器，以便在ApplyConfig时更新检索器配置
			configHandler.SetRetrieverUpdater(app.knowledgeRetriever)
			log.Logger.Info("动态初始化后已设置知识库工具注册器和检索器更新器")
		}

		return knowledgeHandler, nil
	})

	// 如果知识库已启用，设置知识库工具注册器和检索器更新器
	if cfg.Knowledge.Enabled && knowledgeRetriever != nil && knowledgeManager != nil {
		// 创建闭包，捕获knowledgeRetriever和knowledgeManager的引用
		registrar := func() error {
			knowledge.RegisterKnowledgeTool(mcpServer, knowledgeRetriever, knowledgeManager, log.Logger)
			return nil
		}
		configHandler.SetKnowledgeToolRegistrar(registrar)
		// 设置检索器更新器，以便在ApplyConfig时更新检索器配置
		configHandler.SetRetrieverUpdater(knowledgeRetriever)
	}

	// 设置路由（使用 App 实例以便动态获取 handler）
	setupRoutes(
		router,
		agentHandler,
		conversationHandler,
		groupHandler,
		configHandler,
		externalMCPHandler,
		app, // 传递 App 实例以便动态获取 knowledgeHandler
		projectHandler,
		roleHandler,
		skillsHandler,
		mcpServer,
	)

	return app, nil

}

// mcpHandlerWithAuth 在鉴权通过后转发到 MCP 处理；若配置了 auth_header 则校验请求头，否则直接放行
func (a *App) mcpHandlerWithAuth(w http.ResponseWriter, r *http.Request) {
	cfg := a.config.MCP
	if cfg.AuthHeader != "" {
		actual := []byte(r.Header.Get(cfg.AuthHeader))
		expected := []byte(cfg.AuthHeaderValue)
		if subtle.ConstantTimeCompare(actual, expected) != 1 {
			a.logger.Logger.Debug("MCP 鉴权失败：header 缺失或值不匹配", zap.String("header", cfg.AuthHeader))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
	}
	a.mcpServer.HandleHTTP(w, r)
}

// Run 启动应用（向后兼容，不支持优雅关闭）
func (a *App) Run() error {
	return a.RunWithContext(context.Background())
}

// RunWithContext 启动应用，支持通过 context 取消来优雅关闭
func (a *App) RunWithContext(ctx context.Context) error {
	// 启动MCP服务器（如果启用）
	var mcpServer *http.Server
	if a.config.MCP.Enabled {
		mcpAddr := fmt.Sprintf("%s:%d", a.config.MCP.Host, a.config.MCP.Port)
		a.logger.Info("启动MCP服务器", zap.String("address", mcpAddr))

		mux := http.NewServeMux()
		mux.HandleFunc("/mcp", a.mcpHandlerWithAuth)

		mcpServer = &http.Server{Addr: mcpAddr, Handler: mux}
		go func() {
			if err := mcpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				a.logger.Error("MCP服务器启动失败", zap.Error(err))
			}
		}()
	}

	// 启动主服务器（可选 HTTPS + HTTP/2，见 config server.tls_*）
	addr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
	tlsMode, tlsConf, certFile, keyFile, tlsErr := prepareMainServerTLS(&a.config.Server)
	if tlsErr != nil {
		return tlsErr
	}

	srv := &http.Server{Addr: addr, Handler: a.router}
	var mainMux *mainServerMux
	httpRedirect := config.ServerHTTPRedirectEnabled(&a.config.Server)
	if tlsMode != mainTLSOff {
		srv.TLSConfig = tlsConf
		if err := http2.ConfigureServer(srv, &http2.Server{}); err != nil {
			return fmt.Errorf("主服务 HTTP/2 配置失败: %w", err)
		}
		switch tlsMode {
		case mainTLSFromFiles:
			a.logger.Info("启动 HTTPS 主服务（已启用 HTTP/2 协商）",
				zap.String("address", addr),
				zap.String("cert", certFile),
			)
		case mainTLSInMemorySelfSigned:
			a.logger.Info("启动 HTTPS 主服务（内存自签证书，仅测试；已启用 HTTP/2 协商）",
				zap.String("address", addr),
			)
		}
		if httpRedirect {
			a.logger.Info("已启用 HTTP→HTTPS 自动跳转（同端口嗅探分流）", zap.String("address", addr))
		}
	} else {
		a.logger.Info("启动 HTTP 主服务", zap.String("address", addr))
	}

	// 监听 context 取消，优雅关闭 HTTP 服务器
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if mainMux != nil {
			if err := mainMux.Shutdown(shutdownCtx); err != nil {
				a.logger.Error("HTTP/HTTPS 分流服务器关闭失败", zap.Error(err))
			}
		} else if err := srv.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("HTTP服务器关闭失败", zap.Error(err))
		}
		if mcpServer != nil {
			if err := mcpServer.Shutdown(shutdownCtx); err != nil {
				a.logger.Error("MCP服务器关闭失败", zap.Error(err))
			}
		}
	}()

	var err error
	switch {
	case tlsMode != mainTLSOff && httpRedirect:
		var tlsConfReady *tls.Config
		tlsConfReady, err = ensureMainTLSConfigCerts(tlsMode, tlsConf, certFile, keyFile)
		if err != nil {
			return fmt.Errorf("加载 TLS 证书: %w", err)
		}
		srv.TLSConfig = tlsConfReady
		var ln net.Listener
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		mainMux = newMainServerMux(ln, srv, portFromListenAddr(addr), a.logger.Logger)
		err = mainMux.Serve()
	case tlsMode == mainTLSOff:
		err = srv.ListenAndServe()
	case tlsMode == mainTLSFromFiles:
		err = srv.ListenAndServeTLS(certFile, keyFile)
	case tlsMode == mainTLSInMemorySelfSigned:
		var ln net.Listener
		ln, err = tls.Listen("tcp", addr, srv.TLSConfig)
		if err == nil {
			err = srv.Serve(ln)
		}
	default:
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 关闭应用
func (a *App) Shutdown() {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = einoobserve.ShutdownOtel(shutdownCtx)
	shutdownCancel()

	// 停止所有外部MCP客户端
	if a.externalMCPMgr != nil {
		a.externalMCPMgr.StopAll()
	}

	// 关闭知识库数据库连接（如果使用独立数据库）
	if a.knowledgeDB != nil {
		if err := a.knowledgeDB.Close(); err != nil {
			a.logger.Logger.Warn("关闭知识库数据库连接失败", zap.Error(err))
		}
	}

	// 关闭主数据库连接
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Logger.Warn("关闭主数据库连接失败", zap.Error(err))
		}
	}
}

// setupRoutes 设置路由
func setupRoutes(
	router *gin.Engine,
	agentHandler *handler.AgentHandler,
	conversationHandler *handler.ConversationHandler,
	groupHandler *handler.GroupHandler,
	configHandler *handler.ConfigHandler,
	externalMCPHandler *handler.ExternalMCPHandler,
	app *App, // 传递 App 实例以便动态获取 knowledgeHandler
	projectHandler *handler.ProjectHandler,
	roleHandler *handler.RoleHandler,
	skillsHandler *handler.SkillsHandler,
	mcpServer *mcp.Server,
) {
	// API路由
	api := router.Group("/api")

	protected := api.Group("")
	{
		// Eino ADK 单代理（ChatModelAgent + Runner；不依赖 multi_agent.enabled）
		protected.POST("/eino-agent", agentHandler.EinoSingleAgentLoop)
		protected.POST("/eino-agent/stream", agentHandler.EinoSingleAgentLoopStream)
		protected.POST("/agent-loop/cancel", agentHandler.CancelAgentLoop)
		protected.GET("/agent-loop/tasks", agentHandler.ListAgentTasks)
		protected.GET("/agent-loop/task-events", agentHandler.SubscribeAgentTaskEvents)
		protected.GET("/agent-loop/tasks/completed", agentHandler.ListCompletedTasks)
		protected.POST("/chat-uploads", agentHandler.UploadChatAttachment)

		// Eino DeepAgent 多代理（与单 Agent 并存，需 config.multi_agent.enabled）
		// 多代理路由常注册；是否可用由运行时 h.config.MultiAgent.Enabled 决定（应用配置后无需重启）
		protected.POST("/multi-agent", agentHandler.MultiAgentLoop)
		protected.POST("/multi-agent/stream", agentHandler.MultiAgentLoopStream)
		protected.GET("/multi-agent/markdown-agents", configHandler.ListMarkdownAgents)
		protected.POST("/multi-agent/markdown-agents", configHandler.CreateMarkdownAgent)
		protected.GET("/multi-agent/markdown-agents/:filename", configHandler.GetMarkdownAgent)
		protected.PUT("/multi-agent/markdown-agents/:filename", configHandler.UpdateMarkdownAgent)
		protected.DELETE("/multi-agent/markdown-agents/:filename", configHandler.DeleteMarkdownAgent)

		// 对话历史
		protected.POST("/conversations", conversationHandler.CreateConversation)
		protected.GET("/conversations", conversationHandler.ListConversations)
		protected.GET("/conversations/:id", conversationHandler.GetConversation)
		protected.GET("/messages/:id/process-details", conversationHandler.GetMessageProcessDetails)
		protected.PUT("/conversations/:id", conversationHandler.UpdateConversation)
		protected.PUT("/conversations/:id/project", conversationHandler.SetConversationProject)
		protected.DELETE("/conversations/:id", conversationHandler.DeleteConversation)
		protected.POST("/conversations/:id/delete-turn", conversationHandler.DeleteConversationTurn)
		protected.PUT("/conversations/:id/pinned", groupHandler.UpdateConversationPinned)

		// 对话分组
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups", groupHandler.ListGroups)
		protected.GET("/groups/:id", groupHandler.GetGroup)
		protected.PUT("/groups/:id", groupHandler.UpdateGroup)
		protected.DELETE("/groups/:id", groupHandler.DeleteGroup)
		protected.PUT("/groups/:id/pinned", groupHandler.UpdateGroupPinned)
		protected.GET("/groups/:id/conversations", groupHandler.GetGroupConversations)
		protected.GET("/groups/mappings", groupHandler.GetAllMappings)
		protected.POST("/groups/conversations", groupHandler.AddConversationToGroup)
		protected.DELETE("/groups/:id/conversations/:conversationId", groupHandler.RemoveConversationFromGroup)
		protected.PUT("/groups/:id/conversations/:conversationId/pinned", groupHandler.UpdateConversationPinnedInGroup)

		// 配置管理
		protected.GET("/config", configHandler.GetConfig)
		protected.GET("/config/tools", configHandler.GetTools)
		protected.GET("/config/tools/:name/schema", configHandler.GetToolSchema)
		protected.PUT("/config", configHandler.UpdateConfig)
		protected.POST("/config/apply", configHandler.ApplyConfig)
		protected.POST("/config/test-openai", configHandler.TestOpenAI)

		// 外部MCP管理
		protected.GET("/external-mcp", externalMCPHandler.GetExternalMCPs)
		protected.GET("/external-mcp/stats", externalMCPHandler.GetExternalMCPStats)
		protected.GET("/external-mcp/:name", externalMCPHandler.GetExternalMCP)
		protected.PUT("/external-mcp/:name", externalMCPHandler.AddOrUpdateExternalMCP)
		protected.DELETE("/external-mcp/:name", externalMCPHandler.DeleteExternalMCP)
		protected.POST("/external-mcp/:name/start", externalMCPHandler.StartExternalMCP)
		protected.POST("/external-mcp/:name/stop", externalMCPHandler.StopExternalMCP)

		// 知识库管理（始终注册路由，通过 App 实例动态获取 handler）
		knowledgeRoutes := protected.Group("/knowledge")
		{
			knowledgeRoutes.GET("/categories", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"categories": []string{},
						"enabled":    false,
						"message":    "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetCategories(c)
			})
			knowledgeRoutes.GET("/items", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"items":   []interface{}{},
						"enabled": false,
						"message": "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetItems(c)
			})
			knowledgeRoutes.GET("/items/:id", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"message": "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetItem(c)
			})
			knowledgeRoutes.POST("/items", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.CreateItem(c)
			})
			knowledgeRoutes.PUT("/items/:id", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.UpdateItem(c)
			})
			knowledgeRoutes.DELETE("/items/:id", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.DeleteItem(c)
			})
			knowledgeRoutes.GET("/index-status", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled":          false,
						"total_items":      0,
						"indexed_items":    0,
						"progress_percent": 0,
						"is_complete":      false,
						"message":          "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetIndexStatus(c)
			})
			knowledgeRoutes.POST("/index", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.RebuildIndex(c)
			})
			knowledgeRoutes.POST("/scan", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.ScanKnowledgeBase(c)
			})
			knowledgeRoutes.GET("/retrieval-logs", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"logs":    []interface{}{},
						"enabled": false,
						"message": "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetRetrievalLogs(c)
			})
			knowledgeRoutes.DELETE("/retrieval-logs/:id", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled": false,
						"error":   "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.DeleteRetrievalLog(c)
			})
			knowledgeRoutes.POST("/search", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"results": []interface{}{},
						"enabled": false,
						"message": "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.Search(c)
			})
			knowledgeRoutes.GET("/stats", func(c *gin.Context) {
				if app.knowledgeHandler == nil {
					c.JSON(http.StatusOK, gin.H{
						"enabled":          false,
						"total_categories": 0,
						"total_items":      0,
						"message":          "知识库功能未启用，请前往系统设置启用知识检索功能",
					})
					return
				}
				app.knowledgeHandler.GetStats(c)
			})
		}

		// 项目管理与事实黑板
		protected.GET("/projects", projectHandler.ListProjects)
		protected.POST("/projects", projectHandler.CreateProject)
		protected.GET("/projects/:id/stats", projectHandler.GetProjectStats)
		protected.GET("/projects/:id/conversations", projectHandler.ListProjectConversations)
		protected.GET("/projects/:id", projectHandler.GetProject)
		protected.PUT("/projects/:id", projectHandler.UpdateProject)
		protected.DELETE("/projects/:id", projectHandler.DeleteProject)
		protected.GET("/projects/:id/files", projectHandler.ListProjectFiles)
		protected.POST("/projects/:id/files", projectHandler.UploadProjectFile)
		protected.POST("/projects/:id/files/text", projectHandler.WriteProjectFile)
		protected.POST("/projects/:id/files/scan", projectHandler.ScanProjectWorkspace)
		protected.GET("/projects/:id/files/:fileId/download", projectHandler.DownloadProjectFile)
		protected.DELETE("/projects/:id/files/:fileId", projectHandler.DeleteProjectFile)
		protected.GET("/projects/:id/facts", projectHandler.ListFacts)
		protected.GET("/projects/:id/facts/:factId/previous-version", projectHandler.GetFactPreviousVersion)
		protected.GET("/projects/:id/facts/:factId/versions", projectHandler.ListFactVersions)
		protected.POST("/projects/:id/facts", projectHandler.CreateFact)
		protected.PUT("/projects/:id/facts/:factId", projectHandler.UpdateFact)
		protected.DELETE("/projects/:id/facts/:factId", projectHandler.DeleteFact)
		protected.POST("/projects/:id/facts/deprecate", projectHandler.DeprecateFact)
		protected.POST("/projects/:id/facts/restore", projectHandler.RestoreFact)

		// 角色管理
		protected.GET("/roles", roleHandler.GetRoles)
		protected.GET("/roles/:name", roleHandler.GetRole)
		protected.POST("/roles", roleHandler.CreateRole)
		protected.PUT("/roles/:name", roleHandler.UpdateRole)
		protected.DELETE("/roles/:name", roleHandler.DeleteRole)

		// Skills管理（具体路径需注册在 /skills/:name 之前）
		protected.GET("/skills", skillsHandler.GetSkills)
		protected.GET("/skills/stats", skillsHandler.GetSkillStats)
		protected.DELETE("/skills/stats", skillsHandler.ClearSkillStats)
		protected.GET("/skills/:name/files", skillsHandler.ListSkillPackageFiles)
		protected.GET("/skills/:name/file", skillsHandler.GetSkillPackageFile)
		protected.PUT("/skills/:name/file", skillsHandler.PutSkillPackageFile)
		protected.GET("/skills/:name/bound-roles", skillsHandler.GetSkillBoundRoles)
		protected.POST("/skills", skillsHandler.CreateSkill)
		protected.PUT("/skills/:name", skillsHandler.UpdateSkill)
		protected.DELETE("/skills/:name", skillsHandler.DeleteSkill)
		protected.DELETE("/skills/:name/stats", skillsHandler.ClearSkillStatsByName)
		protected.GET("/skills/:name", skillsHandler.GetSkill)

		// MCP端点
		protected.POST("/mcp", func(c *gin.Context) {
			mcpServer.HandleHTTP(c.Writer, c.Request)
		})

	}

	// 静态文件
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// 前端页面
	router.GET("/", func(c *gin.Context) {
		version := app.config.Version
		if version == "" {
			version = "v1.0.0"
		}
		c.HTML(http.StatusOK, "index.html", gin.H{"Version": version})
	})
}

// initializeKnowledge 初始化知识库组件（用于动态初始化）
func initializeKnowledge(
	cfg *config.Config,
	db *database.DB,
	knowledgeDBConn *database.DB,
	mcpServer *mcp.Server,
	agentHandler *handler.AgentHandler,
	app *App, // 传递 App 引用以便更新知识库组件
	logger *zap.Logger,
) (*handler.KnowledgeHandler, error) {
	// 确定知识库数据库路径
	knowledgeDBPath := cfg.Database.KnowledgeDBPath
	var knowledgeDB *sql.DB

	if knowledgeDBPath != "" {
		// 使用独立的知识库数据库
		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(knowledgeDBPath), 0755); err != nil {
			return nil, fmt.Errorf("创建知识库数据库目录失败: %w", err)
		}

		var err error
		knowledgeDBConn, err = database.NewKnowledgeDB(knowledgeDBPath, logger)
		if err != nil {
			return nil, fmt.Errorf("初始化知识库数据库失败: %w", err)
		}
		knowledgeDB = knowledgeDBConn.DB
		logger.Info("使用独立的知识库数据库", zap.String("path", knowledgeDBPath))
	} else {
		// 向后兼容：使用会话数据库
		knowledgeDB = db.DB
		logger.Info("使用会话数据库存储知识库数据（建议配置knowledge_db_path以分离数据）")
	}

	// 创建知识库管理器
	knowledgeManager := knowledge.NewManager(knowledgeDB, cfg.Knowledge.BasePath, logger)

	// 创建嵌入器
	// 使用OpenAI配置的API Key（如果知识库配置中没有指定）
	if cfg.Knowledge.Embedding.APIKey == "" {
		cfg.Knowledge.Embedding.APIKey = cfg.OpenAI.APIKey
	}
	if cfg.Knowledge.Embedding.BaseURL == "" {
		cfg.Knowledge.Embedding.BaseURL = cfg.OpenAI.BaseURL
	}

	embedder, err := knowledge.NewEmbedder(context.Background(), &cfg.Knowledge, &cfg.OpenAI, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化知识库嵌入器失败: %w", err)
	}

	// 创建检索器
	retrievalConfig := &knowledge.RetrievalConfig{
		TopK:                cfg.Knowledge.Retrieval.TopK,
		SimilarityThreshold: cfg.Knowledge.Retrieval.SimilarityThreshold,
		SubIndexFilter:      cfg.Knowledge.Retrieval.SubIndexFilter,
		PostRetrieve:        cfg.Knowledge.Retrieval.PostRetrieve,
	}
	knowledgeRetriever := knowledge.NewRetriever(knowledgeDB, embedder, retrievalConfig, logger)

	// 创建索引器（Eino Compose 链）
	knowledgeIndexer, err := knowledge.NewIndexer(context.Background(), knowledgeDB, embedder, logger, &cfg.Knowledge)
	if err != nil {
		return nil, fmt.Errorf("初始化知识库索引器失败: %w", err)
	}

	// 注册知识检索工具到MCP服务器
	knowledge.RegisterKnowledgeTool(mcpServer, knowledgeRetriever, knowledgeManager, logger)

	// 创建知识库API处理器
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeManager, knowledgeRetriever, knowledgeIndexer, db, logger)
	if app != nil && app.auditSvc != nil {
		knowledgeHandler.SetAudit(app.auditSvc)
	}
	logger.Info("知识库模块初始化完成", zap.Bool("handler_created", knowledgeHandler != nil))

	// 设置知识库管理器到AgentHandler以便记录检索日志
	agentHandler.SetKnowledgeManager(knowledgeManager)

	// 更新 App 中的知识库组件（如果 App 不为 nil，说明是动态初始化）
	if app != nil {
		app.knowledgeManager = knowledgeManager
		app.knowledgeRetriever = knowledgeRetriever
		app.knowledgeIndexer = knowledgeIndexer
		app.knowledgeHandler = knowledgeHandler
		// 如果使用独立数据库，更新 knowledgeDB
		if knowledgeDBPath != "" {
			app.knowledgeDB = knowledgeDBConn
		}
		logger.Info("App 中的知识库组件已更新")
	}

	// 扫描知识库并建立索引（异步）
	go func() {
		itemsToIndex, err := knowledgeManager.ScanKnowledgeBase()
		if err != nil {
			logger.Warn("扫描知识库失败", zap.Error(err))
			return
		}

		// 检查是否已有索引
		hasIndex, err := knowledgeIndexer.HasIndex()
		if err != nil {
			logger.Warn("检查索引状态失败", zap.Error(err))
			return
		}

		if hasIndex {
			// 如果已有索引，只索引新添加或更新的项
			if len(itemsToIndex) > 0 {
				logger.Info("检测到已有知识库索引，开始增量索引", zap.Int("count", len(itemsToIndex)))
				ctx := context.Background()
				consecutiveFailures := 0
				var firstFailureItemID string
				var firstFailureError error
				failedCount := 0

				for _, itemID := range itemsToIndex {
					if err := knowledgeIndexer.IndexItem(ctx, itemID); err != nil {
						failedCount++
						consecutiveFailures++

						if consecutiveFailures == 1 {
							firstFailureItemID = itemID
							firstFailureError = err
							logger.Warn("索引知识项失败", zap.String("itemId", itemID), zap.Error(err))
						}

						// 如果连续失败2次，立即停止增量索引
						if consecutiveFailures >= 2 {
							logger.Error("连续索引失败次数过多，立即停止增量索引",
								zap.Int("consecutiveFailures", consecutiveFailures),
								zap.Int("totalItems", len(itemsToIndex)),
								zap.String("firstFailureItemId", firstFailureItemID),
								zap.Error(firstFailureError),
							)
							break
						}
						continue
					}

					// 成功时重置连续失败计数
					if consecutiveFailures > 0 {
						consecutiveFailures = 0
						firstFailureItemID = ""
						firstFailureError = nil
					}
				}
				logger.Info("增量索引完成", zap.Int("totalItems", len(itemsToIndex)), zap.Int("failedCount", failedCount))
			} else {
				logger.Info("检测到已有知识库索引，没有需要索引的新项或更新项")
			}
			return
		}

		// 只有在没有索引时才自动重建
		logger.Info("未检测到知识库索引，开始自动构建索引")
		ctx := context.Background()
		if err := knowledgeIndexer.RebuildIndex(ctx); err != nil {
			logger.Warn("重建知识库索引失败", zap.Error(err))
		}
	}()

	return knowledgeHandler, nil
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
