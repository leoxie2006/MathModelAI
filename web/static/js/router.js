// 页面路由管理 - 数学建模 AI 基础能力
let currentPage = 'dashboard';

// 合法页面 ID 集合：仅包含基础能力相关页面
const VALID_PAGES = [
    'dashboard', 'chat', 'hitl', 'info-collect', 'projects', 'chat-files', 'settings', 'tasks',
    'mcp-monitor', 'mcp-management',
    'knowledge-management', 'knowledge-retrieval-logs',
    'roles-management',
    'skills-monitor', 'skills-management',
    'agents-management',
];

/** chat 切换时保留 hash 上的查询串（?conversation= / ?conversation_id=） */
function buildHashForPage(pageId) {
    if (pageId !== 'chat') {
        return pageId;
    }
    const full = window.location.hash.slice(1);
    const parts = full.split('?');
    const curPage = parts[0];
    const q = parts.length > 1 ? parts.slice(1).join('?') : '';
    if (curPage === pageId && q) {
        return pageId + '?' + q;
    }
    return pageId;
}

let chatConversationFromHashSeq = 0;
function scheduleChatConversationFromHash(delayMs) {
    const hash = window.location.hash.slice(1);
    const hashParts = hash.split('?');
    if (hashParts[0] !== 'chat' || hashParts.length < 2) {
        return;
    }
    const params = new URLSearchParams(hashParts.slice(1).join('?'));
    const conversationId = params.get('conversation');
    const projectId = params.get('project');
    if (projectId && typeof setActiveProjectId === 'function') {
        setActiveProjectId(projectId);
        if (typeof refreshChatProjectSelector === 'function') {
            refreshChatProjectSelector();
        }
    }
    if (!conversationId) {
        return;
    }
    const token = ++chatConversationFromHashSeq;
    setTimeout(() => {
        if (token !== chatConversationFromHashSeq) {
            return;
        }
        if (typeof loadConversation === 'function') {
            loadConversation(conversationId);
        } else if (typeof window.loadConversation === 'function') {
            window.loadConversation(conversationId);
        } else {
            console.warn('loadConversation function not found');
        }
    }, delayMs);
}

// 初始化路由
function initRouter() {
    const hash = window.location.hash.slice(1);
    if (hash) {
        const hashParts = hash.split('?');
        const pageId = hashParts[0];
        if (pageId && VALID_PAGES.includes(pageId)) {
            switchPage(pageId);
            if (pageId === 'chat') {
                scheduleChatConversationFromHash(500);
            }
            return;
        }
    }
    switchPage('dashboard');
}

// 切换页面
function switchPage(pageId) {
    document.querySelectorAll('.page').forEach(page => {
        page.classList.remove('active');
    });
    const targetPage = document.getElementById(`page-${pageId}`);
    if (targetPage) {
        targetPage.classList.add('active');
        currentPage = pageId;
        const newHash = buildHashForPage(pageId);
        if (window.location.hash.slice(1) !== newHash) {
            window.location.hash = newHash;
        }
        updateNavState(pageId);
        initPage(pageId);
    }
}
window.switchPage = switchPage;

// 更新导航状态
function updateNavState(pageId) {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelectorAll('.nav-submenu-item').forEach(item => {
        item.classList.remove('active');
    });

    // 复合子菜单: mcp / knowledge / skills / agents / roles
    const parentMap = {
        'mcp-monitor': 'mcp',
        'mcp-management': 'mcp',
        'knowledge-management': 'knowledge',
        'knowledge-retrieval-logs': 'knowledge',
        'skills-monitor': 'skills',
        'skills-management': 'skills',
        'agents-management': 'agents',
        'roles-management': 'roles',
    };
    const parent = parentMap[pageId];
    if (parent) {
        const navItem = document.querySelector(`.nav-item[data-page="${parent}"]`);
        if (navItem) {
            navItem.classList.add('active');
            navItem.classList.add('expanded');
        }
        const submenuItem = document.querySelector(`.nav-submenu-item[data-page="${pageId}"]`);
        if (submenuItem) submenuItem.classList.add('active');
    } else {
        const navItem = document.querySelector(`.nav-item[data-page="${pageId}"]`);
        if (navItem) navItem.classList.add('active');
    }
}

function getNavSubmenuItems(navItem) {
    if (!navItem) return [];
    const submenu = navItem.querySelector('.nav-submenu');
    if (!submenu) return [];
    return Array.from(submenu.querySelectorAll('.nav-submenu-item'));
}

function navigateSingleSubmenuPage(navItem) {
    const items = getNavSubmenuItems(navItem);
    if (items.length !== 1) return false;
    const pageId = items[0].getAttribute('data-page');
    if (!pageId) return false;
    switchPage(pageId);
    return true;
}

function toggleSubmenu(menuId) {
    const sidebar = document.getElementById('main-sidebar');
    const navItem = document.querySelector(`.nav-item[data-page="${menuId}"]`);
    if (!navItem) return;
    const collapsed = sidebar && sidebar.classList.contains('collapsed');

    if (collapsed) {
        showSubmenuPopup(navItem, menuId);
        return;
    }

    if (navigateSingleSubmenuPage(navItem)) {
        return;
    }

    const willExpand = !navItem.classList.contains('expanded');
    navItem.classList.toggle('expanded');
    if (willExpand) {
        requestAnimationFrame(() => {
            navItem.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
            const items = getNavSubmenuItems(navItem);
            const last = items[items.length - 1];
            if (last) {
                last.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
            }
        });
    }
}
window.toggleSubmenu = toggleSubmenu;

function showSubmenuPopup(navItem, menuId) {
    const existingPopup = document.querySelector('.submenu-popup');
    if (existingPopup) {
        const sameMenu = existingPopup.dataset.menuId === menuId;
        existingPopup.remove();
        if (sameMenu) {
            return;
        }
    }
    if (navigateSingleSubmenuPage(navItem)) {
        return;
    }
    const navItemContent = navItem.querySelector('.nav-item-content');
    const submenu = navItem.querySelector('.nav-submenu');
    if (!submenu) return;

    const rect = navItemContent.getBoundingClientRect();
    const popup = document.createElement('div');
    popup.className = 'submenu-popup';
    popup.dataset.menuId = menuId;
    popup.style.position = 'fixed';
    popup.style.left = (rect.right + 8) + 'px';
    popup.style.top = rect.top + 'px';
    popup.style.zIndex = '1000';

    const submenuItems = submenu.querySelectorAll('.nav-submenu-item');
    submenuItems.forEach(item => {
        const popupItem = document.createElement('div');
        popupItem.className = 'submenu-popup-item';
        popupItem.textContent = item.textContent.trim();
        const pageId = item.getAttribute('data-page');
        if (pageId && document.querySelector(`.nav-submenu-item[data-page="${pageId}"].active`)) {
            popupItem.classList.add('active');
        }
        popupItem.onclick = function (e) {
            e.stopPropagation();
            e.preventDefault();
            const pid = item.getAttribute('data-page');
            if (pid) {
                switchPage(pid);
            }
            popup.remove();
            document.removeEventListener('click', closePopup);
        };
        popup.appendChild(popupItem);
    });

    document.body.appendChild(popup);
    const closePopup = function (e) {
        if (!popup.contains(e.target) && !navItem.contains(e.target)) {
            popup.remove();
            document.removeEventListener('click', closePopup);
        }
    };
    setTimeout(() => {
        document.addEventListener('click', closePopup);
    }, 0);
}

// 初始化页面
async function initPage(pageId) {
    if (window.i18nReady) await window.i18nReady;
    switch (pageId) {
        case 'dashboard':
            if (typeof refreshDashboard === 'function') {
                refreshDashboard();
            }
            break;
        case 'chat':
            initConversationSidebarState();
            if (typeof prefetchProjectsForChat === 'function') {
                prefetchProjectsForChat();
            }
            if (typeof refreshChatProjectSelector === 'function') {
                refreshChatProjectSelector();
            }
            break;
        case 'hitl':
            if (typeof refreshHitlPending === 'function') {
                refreshHitlPending();
            }
            break;
        case 'info-collect':
            if (typeof initInfoCollectPage === 'function') {
                initInfoCollectPage();
            }
            break;
        case 'tasks':
            if (typeof initTasksPage === 'function') {
                initTasksPage();
            }
            break;
        case 'mcp-monitor':
            if (typeof refreshMonitorPanel === 'function') {
                refreshMonitorPanel();
            }
            break;
        case 'mcp-management':
            const startLoadMcpTools = () => {
                if (typeof loadToolsList === 'function') {
                    if (typeof getToolsPageSize === 'function' && typeof toolsPagination !== 'undefined') {
                        toolsPagination.pageSize = getToolsPageSize();
                    }
                    setTimeout(() => {
                        loadToolsList(1, '').catch(err => {
                            console.error('加载工具列表失败:', err);
                        });
                    }, 100);
                }
            };
            if (typeof loadConfig === 'function') {
                loadConfig(false)
                    .catch(err => console.warn('加载配置失败:', err))
                    .finally(startLoadMcpTools);
            } else {
                startLoadMcpTools();
            }
            if (typeof loadExternalMCPs === 'function') {
                loadExternalMCPs().catch(err => console.warn('加载外部MCP列表失败:', err));
            }
            break;
        case 'projects':
            if (typeof initProjectsPage === 'function') {
                initProjectsPage();
            }
            break;
        case 'chat-files':
            if (typeof initChatFilesPage === 'function') {
                initChatFilesPage();
            }
            break;
        case 'settings':
            if (typeof loadConfig === 'function') {
                loadConfig(false);
            }
            break;
        case 'roles-management':
            const rolesSearchInput = document.getElementById('roles-search');
            if (rolesSearchInput) rolesSearchInput.value = '';
            const rolesSearchClear = document.getElementById('roles-search-clear');
            if (rolesSearchClear) rolesSearchClear.style.display = 'none';
            if (typeof loadRoles === 'function') {
                loadRoles().then(() => {
                    if (typeof renderRolesList === 'function') renderRolesList();
                });
            }
            break;
        case 'skills-monitor':
            if (typeof loadSkillsMonitor === 'function') {
                loadSkillsMonitor();
            }
            break;
        case 'skills-management':
            const skillsSearchInput = document.getElementById('skills-search');
            if (skillsSearchInput) skillsSearchInput.value = '';
            const skillsSearchClear = document.getElementById('skills-search-clear');
            if (skillsSearchClear) skillsSearchClear.style.display = 'none';
            if (typeof initSkillsPagination === 'function') initSkillsPagination();
            if (typeof loadSkills === 'function') loadSkills();
            break;
        case 'agents-management':
            if (typeof loadMarkdownAgents === 'function') {
                loadMarkdownAgents();
            }
            break;
    }

    if (pageId !== 'tasks' && typeof cleanupTasksPage === 'function') {
        cleanupTasksPage();
    }
}

document.addEventListener('DOMContentLoaded', function () {
    initRouter();
    initSidebarState();
    window.addEventListener('hashchange', function () {
        const hash = window.location.hash.slice(1);
        const hashParts = hash.split('?');
        const pageId = hashParts[0];
        if (pageId && VALID_PAGES.includes(pageId)) {
            switchPage(pageId);
            if (pageId === 'chat') {
                scheduleChatConversationFromHash(200);
            }
        }
    });
});

function toggleSidebar() {
    const sidebar = document.getElementById('main-sidebar');
    if (sidebar) {
        sidebar.classList.toggle('collapsed');
        const isCollapsed = sidebar.classList.contains('collapsed');
        localStorage.setItem('sidebarCollapsed', isCollapsed ? 'true' : 'false');
    }
}
window.toggleSidebar = toggleSidebar;

function initSidebarState() {
    const sidebar = document.getElementById('main-sidebar');
    if (sidebar) {
        const savedState = localStorage.getItem('sidebarCollapsed');
        if (savedState === 'true') {
            sidebar.classList.add('collapsed');
        }
    }
    initConversationSidebarState();
}

function toggleConversationSidebar() {
    const sidebar = document.getElementById('conversation-sidebar');
    if (sidebar) {
        sidebar.classList.toggle('collapsed');
        const isCollapsed = sidebar.classList.contains('collapsed');
        localStorage.setItem('conversationSidebarCollapsed', isCollapsed ? 'true' : 'false');
    }
}
window.toggleConversationSidebar = toggleConversationSidebar;

function initConversationSidebarState() {
    const sidebar = document.getElementById('conversation-sidebar');
    if (sidebar) {
        const savedState = localStorage.getItem('conversationSidebarCollapsed');
        if (savedState === 'true') {
            sidebar.classList.add('collapsed');
        } else {
            sidebar.classList.remove('collapsed');
        }
    }
}

window.currentPage = function () { return currentPage; };
