// 设置相关功能 - 数学建模 AI 基础能力配置
// 保留: basic(AI 模型/Agent/MCP/多代理)、knowledge(知识库)
let currentConfig = null;
let allTools = [];
let alwaysVisibleToolNames = new Set();
let alwaysVisibleBuiltinToolNames = new Set();
// 全局工具状态映射
// key: 工具标识符（toolKey），value: { enabled: boolean, is_external: boolean, external_mcp: string }
let toolStateMap = new Map();

// 工具唯一标识符
function getToolKey(tool) {
    if (tool.is_external && tool.external_mcp) {
        return `${tool.external_mcp}::${tool.name}`;
    }
    return tool.name;
}

const getToolsPageSize = () => {
    const saved = localStorage.getItem('toolsPageSize');
    return saved ? parseInt(saved, 10) : 20;
};

let toolsPagination = {
    page: 1,
    pageSize: getToolsPageSize(),
    total: 0,
    totalPages: 0
};
let toolsCurrentSearch = '';
let toolsCurrentStatusFilter = '';
let externalMCPEditingName = null;
let settingsLoadSeq = 0;
let settingsFormDirty = false;

/** 数学建模工作台固定使用 Deep 多代理编排。 */
function syncMultiAgentModeSelectOptions(multiEnabled) {
    const sel = document.getElementById('multi-agent-default-mode');
    if (!sel) return;
    sel.value = 'deep';
}

function getTrimmedValue(id, fallback = '') {
    const el = document.getElementById(id);
    if (!el || typeof el.value !== 'string') return fallback;
    return el.value.trim();
}

function getIntValue(id, fallback) {
    const raw = getTrimmedValue(id, '');
    const n = parseInt(raw, 10);
    return isNaN(n) ? fallback : n;
}

function getFloatValue(id, fallback) {
    const raw = getTrimmedValue(id, '');
    const n = parseFloat(raw);
    return isNaN(n) ? fallback : n;
}

function getCheckedValue(id, fallback = false) {
    const el = document.getElementById(id);
    return el ? !!el.checked : fallback;
}

function setValue(id, value) {
    const el = document.getElementById(id);
    if (el) el.value = value ?? '';
}

function setChecked(id, value) {
    const el = document.getElementById(id);
    if (el) el.checked = !!value;
}

function csvToList(value) {
    return String(value || '')
        .split(',')
        .map(v => v.trim())
        .filter(Boolean);
}

function listToCsv(value) {
    return Array.isArray(value) ? value.join(', ') : '';
}

function markSettingsDirty(event) {
    if (!event || !event.target || typeof event.target.closest !== 'function') return;
    if (event.target.closest('#page-settings')) {
        settingsFormDirty = true;
    }
}

if (typeof document !== 'undefined' && typeof document.addEventListener === 'function') {
    document.addEventListener('input', markSettingsDirty, true);
    document.addEventListener('change', markSettingsDirty, true);
}

// 加载配置并填充表单
async function loadConfig(options = {}) {
    const requestSeq = ++settingsLoadSeq;
    const forceFill = !!(options && options.forceFill);
    try {
        const response = await apiFetch('/api/config');
        if (!response.ok) {
            throw new Error('加载配置失败: ' + response.status);
        }
        const config = await response.json();
        if (requestSeq !== settingsLoadSeq) {
            return currentConfig;
        }
        currentConfig = config;
        if (forceFill || !settingsFormDirty) {
            fillConfigForm(currentConfig);
            settingsFormDirty = false;
        }
        return currentConfig;
    } catch (error) {
        console.error('加载配置失败:', error);
        showNotification('加载配置失败: ' + error.message, 'error');
        return null;
    }
}

// 填充配置表单
function fillConfigForm(config) {
    // OpenAI 配置
    if (config.openai) {
        const baseUrlEl = document.getElementById('openai-base-url');
        if (baseUrlEl) baseUrlEl.value = config.openai.base_url || '';
        const apiKeyEl = document.getElementById('openai-api-key');
        if (apiKeyEl) apiKeyEl.value = config.openai.api_key || '';
        const modelEl = document.getElementById('openai-model');
        if (modelEl) modelEl.value = config.openai.model || '';
        const providerEl = document.getElementById('openai-provider');
        if (providerEl) providerEl.value = config.openai.provider || 'openai';
        const maxTokensEl = document.getElementById('openai-max-total-tokens');
        if (maxTokensEl) maxTokensEl.value = config.openai.max_total_tokens || 120000;

        const reasoning = config.openai.reasoning || {};
        const reasoningModeEl = document.getElementById('openai-reasoning-mode');
        if (reasoningModeEl) reasoningModeEl.value = reasoning.mode || 'auto';
        const reasoningEffortEl = document.getElementById('openai-reasoning-effort');
        if (reasoningEffortEl) reasoningEffortEl.value = reasoning.effort || '';
        const reasoningProfileEl = document.getElementById('openai-reasoning-profile');
        if (reasoningProfileEl) reasoningProfileEl.value = reasoning.profile || 'auto';
        const allowClientEl = document.getElementById('openai-reasoning-allow-client');
        if (allowClientEl) allowClientEl.checked = reasoning.allow_client_reasoning !== false;
    }

    // Agent 配置
    if (config.agent) {
        const maxIterEl = document.getElementById('agent-max-iterations');
        if (maxIterEl) maxIterEl.value = config.agent.max_iterations || 12000;
        const timeoutEl = document.getElementById('agent-tool-timeout');
        if (timeoutEl) timeoutEl.value = config.agent.tool_timeout_minutes || 60;
    }

    // MCP 配置
    if (config.mcp) {
        setChecked('mcp-enabled', config.mcp.enabled || false);
        setValue('mcp-host', config.mcp.host || '0.0.0.0');
        setValue('mcp-port', config.mcp.port || 8081);
    }

    // 知识库配置
    if (config.knowledge) {
        const embedding = config.knowledge.embedding || {};
        const retrieval = config.knowledge.retrieval || {};
        const postRetrieve = retrieval.post_retrieve || {};
        const indexing = config.knowledge.indexing || {};

        setChecked('knowledge-enabled', config.knowledge.enabled || false);
        setValue('knowledge-base-path', config.knowledge.base_path || 'knowledge_base');
        setValue('knowledge-embedding-provider', embedding.provider || 'openai');
        setValue('knowledge-embedding-base-url', embedding.base_url || '');
        setValue('knowledge-embedding-api-key', embedding.api_key || '');
        setValue('knowledge-embedding-model', embedding.model || 'text-embedding-v4');
        setValue('knowledge-retrieval-top-k', retrieval.top_k || 5);
        setValue('knowledge-retrieval-similarity-threshold', retrieval.similarity_threshold ?? 0.4);
        setValue('knowledge-retrieval-sub-index-filter', retrieval.sub_index_filter || '');
        setValue('knowledge-post-retrieve-prefetch-top-k', postRetrieve.prefetch_top_k || 0);
        setValue('knowledge-post-retrieve-max-chars', postRetrieve.max_context_chars || 0);
        setValue('knowledge-post-retrieve-max-tokens', postRetrieve.max_context_tokens || 0);
        setValue('knowledge-indexing-chunk-strategy', indexing.chunk_strategy || 'markdown_then_recursive');
        setValue('knowledge-indexing-request-timeout', indexing.request_timeout_seconds || 120);
        setValue('knowledge-indexing-batch-size', indexing.batch_size || 64);
        setChecked('knowledge-indexing-prefer-source-file', indexing.prefer_source_file || false);
        setValue('knowledge-indexing-sub-indexes', listToCsv(indexing.sub_indexes));
        setValue('knowledge-indexing-chunk-size', indexing.chunk_size || 512);
        setValue('knowledge-indexing-chunk-overlap', indexing.chunk_overlap || 50);
        setValue('knowledge-indexing-max-chunks-per-item', indexing.max_chunks_per_item || 0);
        setValue('knowledge-indexing-max-rpm', indexing.max_rpm || 0);
        setValue('knowledge-indexing-rate-limit-delay-ms', indexing.rate_limit_delay_ms || 300);
        setValue('knowledge-indexing-max-retries', indexing.max_retries || 3);
        setValue('knowledge-indexing-retry-delay-ms', indexing.retry_delay_ms || 1000);
    }

    // 多代理
    if (config.multi_agent) {
        const maEnEl = document.getElementById('multi-agent-enabled');
        if (maEnEl) {
            maEnEl.checked = true;
            maEnEl.disabled = true;
            syncMultiAgentModeSelectOptions(true);
        }
        const maDefaultMode = document.getElementById('multi-agent-default-mode');
        if (maDefaultMode) {
            maDefaultMode.value = 'deep';
            maDefaultMode.disabled = true;
        }
    }
}

// 应用配置
async function applySettings() {
    if (!currentConfig) {
        await loadConfig();
        if (!currentConfig) {
            showNotification('配置尚未加载完成，请稍后再试', 'error');
            return;
        }
    }

    const update = {};

    // OpenAI
    const currentOpenAI = currentConfig.openai || {};
    const currentReasoning = currentOpenAI.reasoning || {};
    const openaiBaseUrl = getTrimmedValue('openai-base-url', '');
    const openaiApiKey = getTrimmedValue('openai-api-key', '');
    const openaiModel = getTrimmedValue('openai-model', '');
    const openaiProvider = getTrimmedValue('openai-provider', currentOpenAI.provider || 'openai') || currentOpenAI.provider || 'openai';
    const maxTokens = getIntValue('openai-max-total-tokens', currentOpenAI.max_total_tokens || 120000);
    const reasoningMode = getTrimmedValue('openai-reasoning-mode', currentReasoning.mode || 'auto') || currentReasoning.mode || 'auto';
    const reasoningEffort = getTrimmedValue('openai-reasoning-effort', '');
    const reasoningProfile = getTrimmedValue('openai-reasoning-profile', currentReasoning.profile || 'auto') || currentReasoning.profile || 'auto';
    const allowClientReasoning = getCheckedValue('openai-reasoning-allow-client', currentReasoning.allow_client_reasoning !== false);
    update.openai = {
        ...currentOpenAI,
        base_url: openaiBaseUrl,
        api_key: openaiApiKey,
        model: openaiModel,
        provider: openaiProvider,
        max_total_tokens: maxTokens,
        reasoning: {
            ...currentReasoning,
            mode: reasoningMode,
            effort: reasoningEffort,
            profile: reasoningProfile,
            allow_client_reasoning: allowClientReasoning,
        },
    };

    // Agent
    const maxIter = getIntValue('agent-max-iterations', NaN);
    const timeout = getIntValue('agent-tool-timeout', NaN);
    if (!isNaN(maxIter) || !isNaN(timeout)) {
        update.agent = {
            max_iterations: isNaN(maxIter) ? undefined : maxIter,
            tool_timeout_minutes: isNaN(timeout) ? undefined : timeout,
        };
    }

    // MCP
    const mcpEnabled = getCheckedValue('mcp-enabled', !!currentConfig.mcp?.enabled);
    const mcpHost = getTrimmedValue('mcp-host', '');
    const mcpPort = getIntValue('mcp-port', NaN);
    const currentMCP = currentConfig.mcp || {};
    update.mcp = {
        ...currentMCP,
        enabled: typeof mcpEnabled === 'boolean' ? mcpEnabled : !!currentMCP.enabled,
        host: mcpHost || currentMCP.host || '0.0.0.0',
        port: isNaN(mcpPort) ? (currentMCP.port || 8081) : mcpPort,
    };

    // 知识库
    const currentKnowledge = currentConfig.knowledge || {};
    const currentEmbedding = currentKnowledge.embedding || {};
    const currentRetrieval = currentKnowledge.retrieval || {};
    const currentPostRetrieve = currentRetrieval.post_retrieve || {};
    const currentIndexing = currentKnowledge.indexing || {};
    update.knowledge = {
        ...currentKnowledge,
        enabled: getCheckedValue('knowledge-enabled', !!currentKnowledge.enabled),
        base_path: getTrimmedValue('knowledge-base-path', currentKnowledge.base_path || 'knowledge_base') || 'knowledge_base',
        embedding: {
            ...currentEmbedding,
            provider: getTrimmedValue('knowledge-embedding-provider', currentEmbedding.provider || 'openai') || 'openai',
            base_url: getTrimmedValue('knowledge-embedding-base-url', ''),
            api_key: getTrimmedValue('knowledge-embedding-api-key', ''),
            model: getTrimmedValue('knowledge-embedding-model', currentEmbedding.model || 'text-embedding-v4') || 'text-embedding-v4',
        },
        retrieval: {
            ...currentRetrieval,
            top_k: getIntValue('knowledge-retrieval-top-k', currentRetrieval.top_k || 5),
            similarity_threshold: getFloatValue('knowledge-retrieval-similarity-threshold', currentRetrieval.similarity_threshold ?? 0.4),
            sub_index_filter: getTrimmedValue('knowledge-retrieval-sub-index-filter', ''),
            post_retrieve: {
                ...currentPostRetrieve,
                prefetch_top_k: getIntValue('knowledge-post-retrieve-prefetch-top-k', currentPostRetrieve.prefetch_top_k || 0),
                max_context_chars: getIntValue('knowledge-post-retrieve-max-chars', currentPostRetrieve.max_context_chars || 0),
                max_context_tokens: getIntValue('knowledge-post-retrieve-max-tokens', currentPostRetrieve.max_context_tokens || 0),
            },
        },
        indexing: {
            ...currentIndexing,
            chunk_strategy: getTrimmedValue('knowledge-indexing-chunk-strategy', currentIndexing.chunk_strategy || 'markdown_then_recursive') || 'markdown_then_recursive',
            request_timeout_seconds: getIntValue('knowledge-indexing-request-timeout', currentIndexing.request_timeout_seconds || 120),
            batch_size: getIntValue('knowledge-indexing-batch-size', currentIndexing.batch_size || 64),
            prefer_source_file: getCheckedValue('knowledge-indexing-prefer-source-file', !!currentIndexing.prefer_source_file),
            sub_indexes: csvToList(getTrimmedValue('knowledge-indexing-sub-indexes', listToCsv(currentIndexing.sub_indexes))),
            chunk_size: getIntValue('knowledge-indexing-chunk-size', currentIndexing.chunk_size || 512),
            chunk_overlap: getIntValue('knowledge-indexing-chunk-overlap', currentIndexing.chunk_overlap || 50),
            max_chunks_per_item: getIntValue('knowledge-indexing-max-chunks-per-item', currentIndexing.max_chunks_per_item || 0),
            max_rpm: getIntValue('knowledge-indexing-max-rpm', currentIndexing.max_rpm || 0),
            rate_limit_delay_ms: getIntValue('knowledge-indexing-rate-limit-delay-ms', currentIndexing.rate_limit_delay_ms || 300),
            max_retries: getIntValue('knowledge-indexing-max-retries', currentIndexing.max_retries || 3),
            retry_delay_ms: getIntValue('knowledge-indexing-retry-delay-ms', currentIndexing.retry_delay_ms || 1000),
        },
    };

    // 多代理
    const currentMultiAgent = currentConfig.multi_agent || {};
    update.multi_agent = {
        enabled: true,
        robot_default_agent_mode: 'deep',
        batch_use_multi_agent: !!currentMultiAgent.batch_use_multi_agent,
        plan_execute_loop_max_iterations: currentMultiAgent.plan_execute_loop_max_iterations,
        tool_search_always_visible_tools: currentMultiAgent.tool_search_always_visible_tools,
    };

    try {
        const saveResponse = await apiFetch('/api/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(update),
        });
        if (!saveResponse.ok) {
            const err = await saveResponse.json().catch(() => ({}));
            throw new Error(err.error || '保存配置失败: ' + saveResponse.status);
        }

        const response = await apiFetch('/api/config/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || '应用配置失败: ' + response.status);
        }
        showNotification('配置已保存并应用', 'success');
        settingsFormDirty = false;
        await loadConfig({ forceFill: true });
    } catch (error) {
        console.error('应用配置失败:', error);
        showNotification(error.message, 'error');
    }
}

// 测试 OpenAI 连接
async function testOpenAI() {
    const openaiBaseUrl = document.getElementById('openai-base-url')?.value?.trim();
    const openaiApiKey = document.getElementById('openai-api-key')?.value?.trim();
    const openaiModel = document.getElementById('openai-model')?.value?.trim();
    const openaiProvider = document.getElementById('openai-provider')?.value || 'openai';
    if (!openaiBaseUrl || !openaiApiKey || !openaiModel) {
        showNotification('请填写 OpenAI base_url、api_key 和 model', 'error');
        return;
    }
    try {
        const response = await apiFetch('/api/config/test-openai', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                provider: openaiProvider,
                base_url: openaiBaseUrl,
                api_key: openaiApiKey,
                model: openaiModel,
            }),
        });
        const data = await response.json();
        if (response.ok && data.success) {
            showNotification('OpenAI 连接成功', 'success');
        } else {
            showNotification('OpenAI 连接失败: ' + (data.error || response.status), 'error');
        }
    } catch (error) {
        showNotification('OpenAI 连接失败: ' + error.message, 'error');
    }
}

async function loadToolsList(page = 1, search = toolsCurrentSearch) {
    const container = document.getElementById('tools-list');
    if (container) {
        container.innerHTML = '<div class="loading-spinner">加载中...</div>';
    }
    toolsPagination.page = page;
    toolsCurrentSearch = search || '';
    const params = new URLSearchParams({
        page: String(toolsPagination.page),
        page_size: String(toolsPagination.pageSize),
    });
    if (toolsCurrentSearch) {
        params.set('search', toolsCurrentSearch);
    }
    if (toolsCurrentStatusFilter) {
        params.set('enabled', toolsCurrentStatusFilter);
    }
    const response = await apiFetch('/api/config/tools?' + params.toString());
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        throw new Error(data.error || ('加载工具列表失败: ' + response.status));
    }
    allTools = Array.isArray(data.tools) ? data.tools : [];
    toolsPagination.total = data.total || allTools.length;
    toolsPagination.totalPages = data.total_pages || Math.max(1, Math.ceil(toolsPagination.total / toolsPagination.pageSize));
    toolsPagination.page = data.page || page;
    toolsPagination.pageSize = data.page_size || toolsPagination.pageSize;
    allTools.forEach(tool => {
        const key = getToolKey(tool);
        if (!toolStateMap.has(key)) {
            toolStateMap.set(key, {
                enabled: !!tool.enabled,
                is_external: !!tool.is_external,
                external_mcp: tool.external_mcp || '',
            });
        }
    });
    renderToolsList();
    updateToolsStats(data.total_enabled);
}

function updateToolsStats(totalEnabled) {
    const el = document.getElementById('tools-stats');
    if (!el) return;
    const enabled = typeof totalEnabled === 'number'
        ? totalEnabled
        : allTools.filter(t => {
            const key = getToolKey(t);
            return toolStateMap.has(key) ? toolStateMap.get(key).enabled : !!t.enabled;
        }).length;
    el.textContent = `已启用 ${enabled} / 共 ${toolsPagination.total || allTools.length}`;
}

// 工具列表(从 /api/config/tools 读取 security.tools_dir 加载后的工具)
function renderToolsList() {
    const container = document.getElementById('tools-list');
    if (!container) return;
    if (!allTools || allTools.length === 0) {
        container.innerHTML = '<div class="empty-state">暂无工具</div>';
        renderToolsPagination(1);
        updateToolsStats(0);
        return;
    }

    container.innerHTML = allTools.map(t => {
        const key = getToolKey(t);
        const checked = toolStateMap.has(key) ? toolStateMap.get(key).enabled : t.enabled;
        const extTag = t.is_external ? `<span class="external-tool-badge">外部:${escapeHtml(t.external_mcp || '')}</span>` : '';
        return `<div class="tool-item">
            <label class="checkbox-label">
                <input type="checkbox" data-tool-key="${key}" ${checked ? 'checked' : ''} class="modern-checkbox tool-toggle" />
                <span class="checkbox-custom"></span>
                <span class="tool-name">${escapeHtml(t.name)}</span>
                ${extTag}
            </label>
            <div class="tool-description tool-item-desc">${escapeHtml(t.description || '')}</div>
        </div>`;
    }).join('');

    container.querySelectorAll('.tool-toggle').forEach(input => {
        input.addEventListener('change', () => {
            const key = input.getAttribute('data-tool-key');
            const tool = allTools.find(t => getToolKey(t) === key);
            toolStateMap.set(key, {
                enabled: input.checked,
                is_external: !!(tool && tool.is_external),
                external_mcp: tool && tool.external_mcp ? tool.external_mcp : '',
            });
            updateToolsStats();
        });
    });

    renderToolsPagination(toolsPagination.totalPages || 1);
    updateToolsStats();
}

function renderToolsPagination(totalPages) {
    const container = document.getElementById('tools-pagination');
    if (!container) return;
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }
    const cur = toolsPagination.page;
    container.innerHTML = `<button ${cur === 1 ? 'disabled' : ''} onclick="toolsGoPage(${cur - 1})">上一页</button>
        <span>第 ${cur} / ${totalPages} 页</span>
        <button ${cur === totalPages ? 'disabled' : ''} onclick="toolsGoPage(${cur + 1})">下一页</button>`;
}

function toolsGoPage(p) {
    const next = Math.max(1, Math.min(p, toolsPagination.totalPages || 1));
    loadToolsList(next, toolsCurrentSearch).catch(err => {
        console.error('加载工具列表失败:', err);
        showNotification('加载工具列表失败: ' + err.message, 'error');
    });
}

function filterToolsByStatus(status) {
    toolsCurrentStatusFilter = status || '';
    document.querySelectorAll('.tools-status-filter .btn-filter').forEach(btn => {
        btn.classList.toggle('active', (btn.dataset.filter || '') === toolsCurrentStatusFilter);
    });
    loadToolsList(1, toolsCurrentSearch).catch(err => {
        console.error('筛选工具失败:', err);
        showNotification('筛选工具失败: ' + err.message, 'error');
    });
}

function searchTools() {
    const input = document.getElementById('tools-search');
    const keyword = input ? input.value.trim() : '';
    loadToolsList(1, keyword).catch(err => {
        console.error('搜索工具失败:', err);
        showNotification('搜索工具失败: ' + err.message, 'error');
    });
}

function clearSearch() {
    const input = document.getElementById('tools-search');
    if (input) input.value = '';
    loadToolsList(1, '').catch(err => {
        console.error('清空搜索失败:', err);
        showNotification('清空搜索失败: ' + err.message, 'error');
    });
}

function handleSearchKeyPress(event) {
    if (event && event.key === 'Enter') {
        searchTools();
    }
}

function setVisibleToolChecks(checked) {
    document.querySelectorAll('#tools-list .tool-toggle').forEach(input => {
        input.checked = checked;
        input.dispatchEvent(new Event('change'));
    });
}

function selectAllTools() {
    setVisibleToolChecks(true);
}

function deselectAllTools() {
    setVisibleToolChecks(false);
}

async function saveToolsConfig() {
    try {
        const tools = Array.from(toolStateMap.entries()).map(([key, state]) => {
            const out = {
                name: key,
                enabled: !!state.enabled,
            };
            if (state.is_external && state.external_mcp) {
                out.is_external = true;
                out.external_mcp = state.external_mcp;
                out.name = key.includes('::') ? key.split('::').slice(1).join('::') : key;
            }
            return out;
        });
        const response = await apiFetch('/api/config', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tools }),
        });
        const data = await response.json().catch(() => ({}));
        if (!response.ok) {
            throw new Error(data.error || ('保存工具配置失败: ' + response.status));
        }
        showNotification('工具配置已保存', 'success');
        toolStateMap.clear();
        await loadToolsList(toolsPagination.page, toolsCurrentSearch);
    } catch (err) {
        console.error('保存工具配置失败:', err);
        showNotification('保存工具配置失败: ' + err.message, 'error');
    }
}

async function refreshMCPPage() {
    await Promise.all([
        loadToolsList(toolsPagination.page || 1, toolsCurrentSearch),
        loadExternalMCPs(),
    ]);
}

async function loadExternalMCPs() {
    const listEl = document.getElementById('external-mcp-list');
    const statsEl = document.getElementById('external-mcp-stats');
    if (listEl) listEl.innerHTML = '<div class="loading-spinner">加载中...</div>';
    const response = await apiFetch('/api/external-mcp');
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        const msg = data.error || ('加载外部 MCP 失败: ' + response.status);
        if (listEl) listEl.innerHTML = '<div class="error">' + escapeHtml(msg) + '</div>';
        throw new Error(msg);
    }
    const servers = data.servers || {};
    const entries = Object.entries(servers).sort((a, b) => a[0].localeCompare(b[0]));
    const stats = data.stats || {};
    if (statsEl) {
        const total = entries.length;
        const connected = entries.filter(([, v]) => v.status === 'connected').length;
        statsEl.innerHTML = `<span><strong>${total}</strong> 个外部 MCP</span><span><strong>${connected}</strong> 已连接</span>`;
    }
    if (!listEl) return;
    if (entries.length === 0) {
        listEl.innerHTML = '<div class="empty">暂无外部 MCP 配置</div>';
        return;
    }
    listEl.innerHTML = '<div class="external-mcp-items">' + entries.map(([name, item]) => {
        const cfg = item.config || {};
        const status = item.status || (cfg.external_mcp_enable ? 'disconnected' : 'disabled');
        const transport = cfg.type || cfg.transport || (cfg.command ? 'stdio' : (cfg.url ? 'http' : 'unknown'));
        const target = cfg.command || cfg.url || '';
        const desc = cfg.description || '';
        const enabled = status !== 'disabled';
        return `<div class="external-mcp-item" data-mcp-name="${escapeHtml(name)}">
            <div class="external-mcp-item-header">
                <div class="external-mcp-item-info">
                    <h4>${escapeHtml(name)} <span class="tool-count-badge">${Number(item.tool_count || 0)} tools</span></h4>
                    <span class="external-mcp-status status-${escapeHtml(status)}">${escapeHtml(status)}</span>
                </div>
                <div class="external-mcp-item-actions">
                    <button class="btn-secondary btn-small" onclick="editExternalMCP('${encodeURIComponent(name)}')">编辑</button>
                    <button class="btn-secondary btn-small" onclick="${enabled ? 'stopExternalMCP' : 'startExternalMCP'}('${encodeURIComponent(name)}')">${enabled ? '停止' : '启动'}</button>
                    <button class="btn-danger btn-small" onclick="deleteExternalMCP('${encodeURIComponent(name)}')">删除</button>
                </div>
            </div>
            <div class="external-mcp-item-details">
                <div><strong>传输</strong><span>${escapeHtml(transport)}</span></div>
                <div><strong>目标</strong><span>${escapeHtml(target || '-')}</span></div>
                <div><strong>描述</strong><span>${escapeHtml(desc || '-')}</span></div>
                ${item.error ? `<div><strong>错误</strong><span>${escapeHtml(item.error)}</span></div>` : ''}
            </div>
        </div>`;
    }).join('') + '</div>';
}

function showAddExternalMCPModal() {
    externalMCPEditingName = null;
    const modal = document.getElementById('external-mcp-modal');
    const title = document.getElementById('external-mcp-modal-title');
    const input = document.getElementById('external-mcp-json');
    const err = document.getElementById('external-mcp-json-error');
    if (title) title.textContent = '添加外部MCP';
    if (input) input.value = '';
    if (err) err.style.display = 'none';
    if (modal) modal.style.display = 'flex';
}

function closeExternalMCPModal() {
    const modal = document.getElementById('external-mcp-modal');
    if (modal) modal.style.display = 'none';
    externalMCPEditingName = null;
}

function setExternalMCPError(message) {
    const err = document.getElementById('external-mcp-json-error');
    if (!err) return;
    err.textContent = message || '';
    err.style.display = message ? 'block' : 'none';
}

function parseExternalMCPJSON() {
    const input = document.getElementById('external-mcp-json');
    const text = input ? input.value.trim() : '';
    if (!text) {
        throw new Error('请填写配置 JSON');
    }
    const obj = JSON.parse(text);
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) {
        throw new Error('配置必须是 JSON 对象');
    }
    return obj;
}

function formatExternalMCPJSON() {
    try {
        const obj = parseExternalMCPJSON();
        const input = document.getElementById('external-mcp-json');
        if (input) input.value = JSON.stringify(obj, null, 2);
        setExternalMCPError('');
    } catch (err) {
        setExternalMCPError(err.message);
    }
}

function loadExternalMCPExample() {
    const input = document.getElementById('external-mcp-json');
    if (!input) return;
    input.value = JSON.stringify({
        "my-server": {
            command: "python3",
            args: ["${HOME}/mcp/server.py"],
            env: { API_KEY: "${API_KEY}" },
            timeout: 300,
            description: "示例 MCP 服务"
        }
    }, null, 2);
    setExternalMCPError('');
}

async function editExternalMCP(encodedName) {
    const name = decodeURIComponent(encodedName);
    const response = await apiFetch('/api/external-mcp/' + encodeURIComponent(name));
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        showNotification('加载外部 MCP 失败: ' + (data.error || response.status), 'error');
        return;
    }
    externalMCPEditingName = name;
    const modal = document.getElementById('external-mcp-modal');
    const title = document.getElementById('external-mcp-modal-title');
    const input = document.getElementById('external-mcp-json');
    if (title) title.textContent = '编辑外部MCP';
    if (input) input.value = JSON.stringify({ [name]: data.config || {} }, null, 2);
    setExternalMCPError('');
    if (modal) modal.style.display = 'flex';
}

async function saveExternalMCP() {
    try {
        const obj = parseExternalMCPJSON();
        const entries = Object.entries(obj);
        if (entries.length !== 1 && externalMCPEditingName) {
            throw new Error('编辑时只能提交一个 MCP 配置');
        }
        for (const [name, cfg] of entries) {
            const cleanName = String(name || '').trim();
            if (!cleanName) throw new Error('配置名称不能为空');
            const response = await apiFetch('/api/external-mcp/' + encodeURIComponent(cleanName), {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ config: cfg }),
            });
            const data = await response.json().catch(() => ({}));
            if (!response.ok) {
                throw new Error(data.error || ('保存失败: ' + response.status));
            }
        }
        closeExternalMCPModal();
        showNotification('外部 MCP 已保存', 'success');
        await loadExternalMCPs();
    } catch (err) {
        setExternalMCPError(err.message);
        showNotification('保存外部 MCP 失败: ' + err.message, 'error');
    }
}

async function deleteExternalMCP(encodedName) {
    const name = decodeURIComponent(encodedName);
    if (!confirm(`确定删除外部 MCP「${name}」吗？`)) return;
    const response = await apiFetch('/api/external-mcp/' + encodeURIComponent(name), { method: 'DELETE' });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        showNotification('删除失败: ' + (data.error || response.status), 'error');
        return;
    }
    showNotification('外部 MCP 已删除', 'success');
    await loadExternalMCPs();
}

async function startExternalMCP(encodedName) {
    const name = decodeURIComponent(encodedName);
    const response = await apiFetch('/api/external-mcp/' + encodeURIComponent(name) + '/start', { method: 'POST' });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        showNotification('启动失败: ' + (data.error || response.status), 'error');
        return;
    }
    showNotification(data.message || '启动请求已提交', 'success');
    await loadExternalMCPs();
}

async function stopExternalMCP(encodedName) {
    const name = decodeURIComponent(encodedName);
    const response = await apiFetch('/api/external-mcp/' + encodeURIComponent(name) + '/stop', { method: 'POST' });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
        showNotification('停止失败: ' + (data.error || response.status), 'error');
        return;
    }
    showNotification(data.message || '外部 MCP 已停止', 'success');
    await loadExternalMCPs();
}

function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({
        '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
}

// 设置 section 切换
function switchSettingsSection(section) {
    document.querySelectorAll('.settings-nav-item').forEach(el => el.classList.remove('active'));
    const navItem = document.querySelector(`.settings-nav-item[data-section="${section}"]`);
    if (navItem) navItem.classList.add('active');
    document.querySelectorAll('.settings-section-content').forEach(el => el.classList.remove('active'));
    const sec = document.getElementById('settings-section-' + section);
    if (sec) sec.classList.add('active');
}

// 暴露给 HTML onclick
window.switchSettingsSection = switchSettingsSection;
window.applySettings = applySettings;
window.testOpenAI = testOpenAI;
window.testOpenAIConnection = testOpenAI;
window.loadToolsList = loadToolsList;
window.saveToolsConfig = saveToolsConfig;
window.selectAllTools = selectAllTools;
window.deselectAllTools = deselectAllTools;
window.filterToolsByStatus = filterToolsByStatus;
window.searchTools = searchTools;
window.clearSearch = clearSearch;
window.handleSearchKeyPress = handleSearchKeyPress;
window.toolsGoPage = toolsGoPage;
window.refreshMCPPage = refreshMCPPage;
window.loadExternalMCPs = loadExternalMCPs;
window.showAddExternalMCPModal = showAddExternalMCPModal;
window.closeExternalMCPModal = closeExternalMCPModal;
window.formatExternalMCPJSON = formatExternalMCPJSON;
window.loadExternalMCPExample = loadExternalMCPExample;
window.saveExternalMCP = saveExternalMCP;
window.editExternalMCP = editExternalMCP;
window.deleteExternalMCP = deleteExternalMCP;
window.startExternalMCP = startExternalMCP;
window.stopExternalMCP = stopExternalMCP;
window.loadConfig = loadConfig;
window.syncMultiAgentModeSelectOptions = syncMultiAgentModeSelectOptions;
