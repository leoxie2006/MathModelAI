let isAppInitialized = false;

async function apiFetch(url, options = {}) {
    return fetch(url, options);
}

/**
 * multipart POST with XMLHttpRequest so upload progress is available.
 * 返回与 fetch 类似的对象：ok、status、json()、text()
 */
async function apiUploadWithProgress(url, formData, options = {}) {
    const onProgress = typeof options.onProgress === 'function' ? options.onProgress : null;
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', url);
        xhr.upload.onprogress = (e) => {
            if (!onProgress || !e.lengthComputable) return;
            const percent = e.total > 0 ? Math.round((e.loaded / e.total) * 100) : 0;
            onProgress({ loaded: e.loaded, total: e.total, percent });
        };
        xhr.onerror = () => reject(new Error('Network error'));
        xhr.onload = () => {
            const responseText = xhr.responseText || '';
            resolve({
                ok: xhr.status >= 200 && xhr.status < 300,
                status: xhr.status,
                text: async () => responseText,
                json: async () => (responseText ? JSON.parse(responseText) : {}),
            });
        };
        xhr.send(formData);
    });
}

async function refreshAppData(showTaskErrors = false) {
    if (typeof initChatAgentModeFromConfig === 'function') {
        try {
            await initChatAgentModeFromConfig();
        } catch (error) {
            console.warn('刷新对话模式配置失败:', error);
        }
    }
    await Promise.allSettled([
        typeof loadConversations === 'function' ? loadConversations() : Promise.resolve(),
        typeof loadActiveTasks === 'function' ? loadActiveTasks(showTaskErrors) : Promise.resolve(),
    ]);
}

async function bootstrapApp() {
    if (!isAppInitialized) {
        try {
            if (window.i18nReady && typeof window.i18nReady.then === 'function') {
                await window.i18nReady;
            }
        } catch (e) {
            console.warn('等待 i18n 就绪失败，继续初始化聊天', e);
        }
        if (typeof initializeChatUI === 'function') {
            initializeChatUI();
        }
        isAppInitialized = true;
    }
    await refreshAppData();
}

window.apiFetch = apiFetch;
window.apiUploadWithProgress = apiUploadWithProgress;

document.addEventListener('DOMContentLoaded', bootstrapApp);
