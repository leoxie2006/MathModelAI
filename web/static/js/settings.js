// 设置相关功能 - 数学建模 AI 基础能力配置
// 保留: basic(AI 模型/Agent/MCP/HITL)、infocollect(信息收集)、knowledge(知识库)
// 删除: c2 / robots(企微/钉钉/飞书)/ terminal / audit / security
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

/** 根据多代理开关,启用/禁用 Eino 编排选项 */
function syncRobotAgentModeSelectOptions(multiEnabled) {
    const sel = document.getElementById('multi-agent-robot-mode');
    if (!sel) return;
    ['deep', 'plan_execute', 'supervisor'].forEach(function (v) {
        const opt = sel.querySelector('option[value="' + v + '"]');
        if (opt) opt.disabled = !multiEnabled;
    });
    if (!multiEnabled && ['deep', 'plan_execute', 'supervisor'].indexOf(sel.value) >= 0) {
        sel.value = 'eino_single';
    }
}

// 加载配置并填充表单
async function loadConfig() {
    try {
        const response = await apiFetch('/api/config');
        if (!response.ok) {
            throw new Error('加载配置失败: ' + response.status);
        }
        currentConfig = await response.json();
        fillConfigForm(currentConfig);
    } catch (error) {
        console.error('加载配置失败:', error);
        showNotification('加载配置失败: ' + error.message, 'error');
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
        const mcpEnabledEl = document.getElementById('mcp-enabled');
        if (mcpEnabledEl) mcpEnabledEl.checked = config.mcp.enabled || false;
        const mcpHostEl = document.getElementById('mcp-host');
        if (mcpHostEl) mcpHostEl.value = config.mcp.host || '0.0.0.0';
        const mcpPortEl = document.getElementById('mcp-port');
        if (mcpPortEl) mcpPortEl.value = config.mcp.port || 8081;
    }

    // HITL 配置
    if (config.hitl) {
        const whitelistEl = document.getElementById('hitl-whitelist');
        if (whitelistEl) {
            whitelistEl.value = (config.hitl.tool_whitelist || []).join(', ');
        }
    }

    // 知识库配置
    if (config.knowledge) {
        const kEnabledEl = document.getElementById('knowledge-enabled');
        if (kEnabledEl) kEnabledEl.checked = config.knowledge.enabled || false;
        const kBaseEl = document.getElementById('knowledge-base-path');
        if (kBaseEl) kBaseEl.value = config.knowledge.base_path || 'knowledge_base';
    }

    // 多代理
    if (config.multi_agent) {
        const maEnEl = document.getElementById('multi-agent-enabled');
        if (maEnEl) {
            maEnEl.checked = config.multi_agent.enabled !== false;
            syncRobotAgentModeSelectOptions(maEnEl.checked);
        }
        const maRobotMode = document.getElementById('multi-agent-robot-mode');
        if (maRobotMode) {
            let mode = (config.multi_agent.robot_default_agent_mode || 'eino_single').trim().toLowerCase();
            if (!['eino_single', 'deep', 'plan_execute', 'supervisor'].includes(mode)) {
                mode = 'eino_single';
            }
            maRobotMode.value = mode;
        }
    }
}

// 应用配置
async function applySettings() {
    if (!currentConfig) {
        showNotification('请先加载配置', 'error');
        return;
    }

    const update = {};

    // OpenAI
    const openaiBaseUrl = document.getElementById('openai-base-url')?.value?.trim();
    const openaiApiKey = document.getElementById('openai-api-key')?.value?.trim();
    const openaiModel = document.getElementById('openai-model')?.value?.trim();
    const openaiProvider = document.getElementById('openai-provider')?.value;
    if (openaiBaseUrl || openaiApiKey || openaiModel) {
        update.openai = {
            ...(currentConfig.openai || {}),
            base_url: openaiBaseUrl,
            api_key: openaiApiKey,
            model: openaiModel,
            provider: openaiProvider,
        };
    }

    // Agent
    const maxIter = parseInt(document.getElementById('agent-max-iterations')?.value, 10);
    const timeout = parseInt(document.getElementById('agent-tool-timeout')?.value, 10);
    if (!isNaN(maxIter) || !isNaN(timeout)) {
        update.agent = {
            max_iterations: isNaN(maxIter) ? undefined : maxIter,
            tool_timeout_minutes: isNaN(timeout) ? undefined : timeout,
        };
    }

    // MCP
    const mcpEnabled = document.getElementById('mcp-enabled')?.checked;
    const mcpHost = document.getElementById('mcp-host')?.value?.trim();
    const mcpPort = parseInt(document.getElementById('mcp-port')?.value, 10);
    update.mcp = {
        enabled: !!mcpEnabled,
        host: mcpHost || '0.0.0.0',
        port: isNaN(mcpPort) ? 8081 : mcpPort,
    };

    // HITL
    const hitlWhitelist = document.getElementById('hitl-whitelist')?.value;
    if (hitlWhitelist !== undefined) {
        const list = (hitlWhitelist || '').split(',').map(s => s.trim()).filter(Boolean);
        update.hitl = { tool_whitelist: list };
    }

    // 知识库
    const kEnabled = document.getElementById('knowledge-enabled')?.checked;
    const kBase = document.getElementById('knowledge-base-path')?.value?.trim();
    update.knowledge = {
        enabled: !!kEnabled,
        base_path: kBase || 'knowledge_base',
    };

    // 多代理
    const maEnabled = document.getElementById('multi-agent-enabled')?.checked;
    const maRobotMode = document.getElementById('multi-agent-robot-mode')?.value;
    if (maEnabled !== undefined) {
        update.multi_agent = {
            enabled: !!maEnabled,
            robot_default_agent_mode: maRobotMode || 'eino_single',
        };
    }

    try {
        const response = await apiFetch('/api/config/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(update),
        });
        if (!response.ok) {
            const err = await response.json().catch(() => ({}));
            throw new Error(err.error || '应用配置失败: ' + response.status);
        }
        showNotification('配置已应用', 'success');
        await loadConfig();
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
    if (!openaiBaseUrl || !openaiApiKey || !openaiModel) {
        showNotification('请填写 OpenAI base_url、api_key 和 model', 'error');
        return;
    }
    try {
        const response = await apiFetch('/api/config/test-openai', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                openai: { base_url: openaiBaseUrl, api_key: openaiApiKey, model: openaiModel },
            }),
        });
        const data = await response.json();
        if (response.ok && data.ok) {
            showNotification('OpenAI 连接成功', 'success');
        } else {
            showNotification('OpenAI 连接失败: ' + (data.error || response.status), 'error');
        }
    } catch (error) {
        showNotification('OpenAI 连接失败: ' + error.message, 'error');
    }
}

// 工具列表(从 /api/config 中读取 Security.Tools)
function renderToolsList() {
    const container = document.getElementById('tools-list');
    if (!container) return;
    if (!allTools || allTools.length === 0) {
        container.innerHTML = '<div class="empty-state">暂无工具</div>';
        return;
    }

    const pageSize = toolsPagination.pageSize;
    const totalPages = Math.max(1, Math.ceil(allTools.length / pageSize));
    if (toolsPagination.page > totalPages) toolsPagination.page = totalPages;
    const start = (toolsPagination.page - 1) * pageSize;
    const pageItems = allTools.slice(start, start + pageSize);

    container.innerHTML = pageItems.map(t => {
        const key = getToolKey(t);
        const checked = toolStateMap.has(key) ? toolStateMap.get(key).enabled : t.enabled;
        const extTag = t.is_external ? `<span class="tag tag-external">外部:${t.external_mcp}</span>` : '';
        return `<div class="tool-item">
            <label class="checkbox-label">
                <input type="checkbox" data-tool-key="${key}" ${checked ? 'checked' : ''} class="modern-checkbox tool-toggle" />
                <span class="checkbox-custom"></span>
                <span class="tool-name">${escapeHtml(t.name)}</span>
                ${extTag}
            </label>
            <div class="tool-description">${escapeHtml(t.description || '')}</div>
        </div>`;
    }).join('');

    renderToolsPagination(totalPages);
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
    toolsPagination.page = p;
    renderToolsList();
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
window.toolsGoPage = toolsGoPage;
window.loadConfig = loadConfig;
window.syncRobotAgentModeSelectOptions = syncRobotAgentModeSelectOptions;
