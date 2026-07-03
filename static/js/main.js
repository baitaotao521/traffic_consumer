AOS.init({ once: true });

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-bs-toggle="tooltip"]').forEach((element) => {
        bootstrap.Tooltip.getOrCreateInstance(element);
    });

    const startBtn = document.getElementById('start-btn');
    const tempTestBtn = document.getElementById('temp-test-btn');
    const stopBtn = document.getElementById('stop-btn');
    const stopSchedulerBtn = document.getElementById('stop-scheduler-btn');
    const deleteConfigBtn = document.getElementById('delete-config-btn');
    const saveConfigBtn = document.getElementById('save-config-btn');
    const configSelect = document.getElementById('config-select');
    const runningStatus = document.getElementById('running-status');
    const jobDetailsEl = document.getElementById('job-details');
    const cronPreviewEl = document.getElementById('cron-preview');
    const logSwitch = document.getElementById('log-switch');
    const logContainer = document.getElementById('log-container');
    const clearLogBtn = document.getElementById('clear-log-btn');
    const editorConfigSelect = document.getElementById('config-editor-select');
    const configEditorEl = document.getElementById('config-editor');
    const resetConfigBtn = document.getElementById('reset-config-btn');
    const threadStatusList = document.getElementById('thread-status-list');
    const urlUsageList = document.getElementById('url-usage-list');
    const notificationArea = document.getElementById('notification-area');
    const activeThreadCountEl = document.getElementById('active-thread-count');
    const idleThreadCountEl = document.getElementById('idle-thread-count');
    const totalThreadCountEl = document.getElementById('total-thread-count');
    const erroredThreadCountEl = document.getElementById('errored-thread-count');
    const currentConfigEl = document.getElementById('current-config');
    const runtimePlanListEl = document.getElementById('runtime-plan-list');
    const planDetailModalEl = document.getElementById('plan-detail-modal');
    const planDetailNameEl = document.getElementById('plan-detail-name');
    const planDetailNextRunEl = document.getElementById('plan-detail-next-run');
    const planDetailTotalEl = document.getElementById('plan-detail-total');
    const planDetailHistoryBody = document.getElementById('plan-detail-history-body');
    const planDetailModal = planDetailModalEl ? bootstrap.Modal.getOrCreateInstance(planDetailModalEl) : null;

    const configInputs = {
        name: document.getElementById('config-name'),
        urls: document.getElementById('urls'),
        threads: document.getElementById('threads'),
        limit_speed: document.getElementById('limit-speed'),
        traffic_limit: document.getElementById('traffic-limit'),
        duration: document.getElementById('duration'),
        count: document.getElementById('count'),
        cron_expr: document.getElementById('cron-expr'),
        interval: document.getElementById('interval'),
        url_strategy: document.getElementById('url-strategy'),
        auto_remove_failed_url: document.getElementById('auto-remove-failed-url'),
        auto_start: document.getElementById('auto-start'),
        user_agent: document.getElementById('user-agent'),
        request_headers: document.getElementById('request-headers'),
        url_switch_interval: document.getElementById('url-switch-interval'),
        thread_start_delay: document.getElementById('thread-start-delay')
    };

    let selectedConfigName = null;
    let selectedConfigDetail = null;
    let editorActiveConfig = null;
    let logCursor = 0;
    let eventCursor = 0;

    const speedChart = new Chart(document.getElementById('speed-chart').getContext('2d'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: '速度 (MB/s)',
                data: [],
                borderColor: 'rgba(255, 105, 180, 0.8)',
                backgroundColor: 'rgba(255, 105, 180, 0.2)',
                fill: true,
                tension: 0.4
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                x: { display: false },
                y: { beginAtZero: true, ticks: { color: '#FF69B4' } }
            },
            plugins: { legend: { display: false } }
        }
    });

    const urlPieChart = new Chart(document.getElementById('url-pie-chart').getContext('2d'), {
        type: 'doughnut',
        data: {
            labels: ['无数据'],
            datasets: [{ data: [1], backgroundColor: ['rgba(255, 228, 240, 0.9)'], borderWidth: 0 }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '58%',
            plugins: {
                legend: { position: 'bottom', labels: { boxWidth: 14, boxHeight: 14, padding: 12, color: '#666' } },
                tooltip: { callbacks: { label: (context) => `${context.label || ''}: ${context.parsed} 次` } }
            }
        }
    });

    const threadUsageChart = new Chart(document.getElementById('thread-usage-chart').getContext('2d'), {
        type: 'doughnut',
        data: {
            labels: ['活跃', '空闲', '失效'],
            datasets: [{
                data: [0, 0, 0],
                backgroundColor: ['rgba(255, 105, 180, 0.85)', 'rgba(255, 182, 193, 0.85)', 'rgba(220, 53, 69, 0.75)'],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '65%',
            plugins: {
                legend: { position: 'bottom', labels: { boxWidth: 12, boxHeight: 12, padding: 10, color: '#666' } },
                tooltip: { callbacks: { label: (context) => `${context.label || ''}: ${context.parsed} 条线程` } }
            }
        }
    });

    async function apiFetch(url, options = {}) {
        const response = await fetch(url, {
            headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
            ...options
        });
        const text = await response.text();
        const data = text ? JSON.parse(text) : {};
        if (!response.ok) {
            throw new Error(data.message || data.error || '请求失败');
        }
        return data;
    }

    function encodedName(name) {
        return encodeURIComponent(name || '');
    }

    function applyRunningState(state, isRunning) {
        const normalizedState = typeof state === 'string' ? state : (isRunning ? 'running' : 'stopped');
        if (normalizedState === 'scheduled') {
            runningStatus.textContent = '等待调度';
            runningStatus.className = 'badge bg-info text-dark';
            startBtn.disabled = true;
            if (tempTestBtn) tempTestBtn.disabled = false;
            stopBtn.disabled = false;
            return;
        }
        if (normalizedState === 'running') {
            runningStatus.textContent = '运行中';
            runningStatus.className = 'badge bg-success';
            startBtn.disabled = true;
            if (tempTestBtn) tempTestBtn.disabled = true;
            stopBtn.disabled = false;
            return;
        }
        runningStatus.textContent = '已停止';
        runningStatus.className = 'badge bg-secondary';
        startBtn.disabled = false;
        if (tempTestBtn) tempTestBtn.disabled = false;
        stopBtn.disabled = true;
    }

    function addDataToChart(label, data) {
        speedChart.data.labels.push(label);
        speedChart.data.datasets[0].data.push(data);
        if (speedChart.data.labels.length > 30) {
            speedChart.data.labels.shift();
            speedChart.data.datasets[0].data.shift();
        }
        speedChart.update('none');
    }

    function updateThreadUsageChart(active = 0, idle = 0, errored = 0) {
        threadUsageChart.data.datasets[0].data = [active, idle, errored];
        threadUsageChart.update('none');
    }

    function updateUrlPieChart(stats = []) {
        if (!Array.isArray(stats) || stats.length === 0) {
            urlPieChart.data.labels = ['无数据'];
            urlPieChart.data.datasets[0].data = [1];
            urlPieChart.data.datasets[0].backgroundColor = ['rgba(255, 228, 240, 0.9)'];
            urlPieChart.update('none');
            return;
        }
        const palette = [
            'rgba(255, 105, 180, 0.85)',
            'rgba(255, 182, 193, 0.85)',
            'rgba(255, 160, 176, 0.85)',
            'rgba(255, 215, 230, 0.85)',
            'rgba(255, 240, 245, 0.9)',
            'rgba(255, 200, 210, 0.85)'
        ];
        urlPieChart.data.labels = stats.map((item, index) => item.url || `链接 ${index + 1}`);
        urlPieChart.data.datasets[0].data = stats.map((item) => Number(item.count) || 0);
        urlPieChart.data.datasets[0].backgroundColor = stats.map((_, index) => palette[index % palette.length]);
        urlPieChart.update('none');
    }

    function resetConfigForm(keepSelection = false) {
        Object.values(configInputs).forEach((element) => {
            if (!element) return;
            if (element.type === 'checkbox') {
                element.checked = false;
            } else {
                element.value = '';
            }
        });
        if (!keepSelection && editorConfigSelect) editorConfigSelect.value = '';
        if (cronPreviewEl) cronPreviewEl.innerHTML = '';
        editorActiveConfig = null;
    }

    function populateEditorForm(name, config = {}) {
        if (configInputs.name) configInputs.name.value = name || '';
        if (configInputs.urls) configInputs.urls.value = Array.isArray(config.urls) ? config.urls.join('\n') : '';
        if (configInputs.threads) configInputs.threads.value = config.threads ?? '';
        if (configInputs.limit_speed) configInputs.limit_speed.value = config.limit_speed ?? '';
        if (configInputs.traffic_limit) configInputs.traffic_limit.value = config.traffic_limit ?? '';
        if (configInputs.duration) configInputs.duration.value = config.duration ?? '';
        if (configInputs.count) configInputs.count.value = config.count ?? '';
        if (configInputs.cron_expr) configInputs.cron_expr.value = config.cron_expr ?? '';
        if (configInputs.interval) configInputs.interval.value = config.interval ?? '';
        if (configInputs.url_strategy) configInputs.url_strategy.value = config.url_strategy ?? '';
        if (configInputs.auto_remove_failed_url) configInputs.auto_remove_failed_url.checked = Boolean(config.auto_remove_failed_url);
        if (configInputs.auto_start) configInputs.auto_start.checked = Boolean(config.auto_start);
        if (configInputs.user_agent) configInputs.user_agent.value = config.user_agent ?? '';
        if (configInputs.request_headers) {
            const headers = config.request_headers;
            configInputs.request_headers.value = headers && typeof headers === 'object'
                ? Object.entries(headers).map(([key, value]) => `${key}: ${value}`).join('\n')
                : '';
        }
        if (configInputs.url_switch_interval) configInputs.url_switch_interval.value = config.url_switch_interval ?? '';
        if (configInputs.thread_start_delay) configInputs.thread_start_delay.value = config.thread_start_delay ?? '';
        editorActiveConfig = name || null;
        if (configInputs.cron_expr) configInputs.cron_expr.dispatchEvent(new Event('input'));
    }

    function normalizeConfigPayload(config = {}, name = null) {
        const payload = {
            urls: Array.isArray(config.urls) ? [...config.urls] : [],
            url_strategy: config.url_strategy || null,
            threads: parseInteger(config.threads),
            limit_speed: parseFloatOrNull(config.limit_speed),
            traffic_limit: parseInteger(config.traffic_limit),
            duration: parseInteger(config.duration),
            count: parseInteger(config.count),
            cron_expr: config.cron_expr || null,
            interval: parseInteger(config.interval),
            config_name: name || config.config_name || null,
            name: name || config.name || '',
            auto_remove_failed_url: Boolean(config.auto_remove_failed_url),
            auto_start: Boolean(config.auto_start),
            user_agent: config.user_agent ? String(config.user_agent).trim() : null,
            request_headers: normalizeHeaders(config.request_headers),
            url_switch_interval: parseFloatOrNull(config.url_switch_interval),
            thread_start_delay: parseFloatOrNull(config.thread_start_delay)
        };
        payload.urls = payload.urls.map((url) => String(url || '').trim()).filter(Boolean);
        return payload;
    }

    function parseInteger(value) {
        if (value === null || value === undefined || value === '') return null;
        const parsed = parseInt(value, 10);
        return Number.isFinite(parsed) ? parsed : null;
    }

    function parseFloatOrNull(value) {
        if (value === null || value === undefined || value === '') return null;
        const parsed = parseFloat(value);
        return Number.isFinite(parsed) ? parsed : null;
    }

    function normalizeHeaders(headers) {
        if (!headers) return null;
        if (typeof headers === 'string') {
            const result = {};
            headers.split(/\r?\n/).forEach((line) => {
                const index = line.indexOf(':');
                if (index <= 0) return;
                const name = line.slice(0, index).trim();
                const value = line.slice(index + 1).trim();
                if (name && value) result[name] = value;
            });
            return Object.keys(result).length ? result : null;
        }
        if (typeof headers === 'object') {
            return Object.fromEntries(Object.entries(headers).filter(([key, value]) => key && value));
        }
        return null;
    }

    function getConfigFromForm() {
        const raw = {};
        Object.entries(configInputs).forEach(([key, element]) => {
            if (!element) {
                raw[key] = null;
                return;
            }
            if (element.type === 'checkbox') {
                raw[key] = Boolean(element.checked);
            } else if (key === 'urls') {
                raw.urls = element.value.split(/\r?\n/).map((url) => url.trim()).filter(Boolean);
            } else {
                raw[key] = typeof element.value === 'string' ? element.value.trim() : element.value;
            }
        });
        const normalized = normalizeConfigPayload(raw, raw.name || null);
        normalized.name = raw.name || '';
        return normalized;
    }

    async function loadConfigs() {
        try {
            const data = await apiFetch('/api/configs');
            renderConfigLists(Array.isArray(data.configs) ? data.configs : []);
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    }

    function renderConfigLists(configs) {
        const previousSelection = selectedConfigName;
        if (configSelect) {
            configSelect.innerHTML = '';
            const placeholder = document.createElement('option');
            placeholder.value = '';
            placeholder.textContent = configs.length ? '请选择运行配置' : '暂无配置，请先创建';
            configSelect.appendChild(placeholder);
            configs.forEach((name) => configSelect.appendChild(new Option(name, name)));
            if (configs.length > 0) {
                const next = configs.includes(previousSelection) ? previousSelection : configs[0];
                configSelect.value = next;
                requestConfigDetails(next, 'runtime');
            } else {
                selectedConfigName = null;
                selectedConfigDetail = null;
            }
        }
        if (editorConfigSelect) {
            const previousEditor = editorActiveConfig || editorConfigSelect.value;
            editorConfigSelect.innerHTML = '';
            editorConfigSelect.appendChild(new Option('新建配置', ''));
            configs.forEach((name) => editorConfigSelect.appendChild(new Option(name, name)));
            if (previousEditor && configs.includes(previousEditor)) {
                editorConfigSelect.value = previousEditor;
            }
        }
    }

    async function requestConfigDetails(name, target) {
        if (!name) {
            if (target === 'editor') resetConfigForm(true);
            return;
        }
        try {
            const data = await apiFetch(`/api/configs/${encodedName(name)}`);
            if (target === 'editor') {
                populateEditorForm(data.name, data.config || {});
                if (editorConfigSelect) editorConfigSelect.value = data.name;
                return;
            }
            selectedConfigName = data.name;
            selectedConfigDetail = normalizeConfigPayload(data.config || {}, data.name);
            if (currentConfigEl) {
                currentConfigEl.textContent = data.name;
                currentConfigEl.title = data.name;
            }
            if (configSelect) configSelect.value = data.name;
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    }

    function renderThreadStatus(threadMap = {}, totalThreads = 0) {
        if (!threadStatusList) return;
        const entries = Object.entries(threadMap || {});
        const expectedThreads = Number(totalThreads) || entries.length;
        threadStatusList.innerHTML = '';
        if (!expectedThreads) {
            threadStatusList.innerHTML = '<tr class="placeholder-row"><td colspan="3" class="text-center text-muted py-3">暂无线程数据。</td></tr>';
            setThreadCounts(0, 0, 0, 0);
            return;
        }
        const mapped = new Map(entries.map(([key, value]) => [Number(key), value]));
        let activeCount = 0;
        let idleCount = 0;
        let errorCount = 0;
        for (let i = 1; i <= expectedThreads; i += 1) {
            let displayUrl = mapped.get(i) || '等待分配...';
            let status = 'idle';
            if (displayUrl.includes('已失效') || displayUrl.includes('无可用')) status = 'error';
            else if (displayUrl && displayUrl !== '等待分配...' && displayUrl !== '等待中...') status = 'active';

            if (status === 'active') activeCount += 1;
            else if (status === 'error') errorCount += 1;
            else idleCount += 1;

            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="text-muted">线程 ${i}</td>
                <td><span class="thread-url text-truncate ${status === 'error' ? 'text-danger' : ''}" title="${escapeAttr(displayUrl)}">${escapeHtml(displayUrl)}</span></td>
                <td class="text-end"><span class="badge thread-status-badge ${status === 'active' ? 'bg-success' : status === 'error' ? 'bg-danger' : 'bg-secondary'}">${status === 'active' ? '运行中' : status === 'error' ? (displayUrl.includes('无可用') ? '耗尽' : '失效') : '待命'}</span></td>
            `;
            threadStatusList.appendChild(row);
        }
        setThreadCounts(expectedThreads, activeCount, idleCount, errorCount);
    }

    function setThreadCounts(total, active, idle, errored) {
        if (totalThreadCountEl) totalThreadCountEl.textContent = String(total);
        if (activeThreadCountEl) activeThreadCountEl.textContent = String(active);
        if (idleThreadCountEl) idleThreadCountEl.textContent = String(idle);
        if (erroredThreadCountEl) erroredThreadCountEl.textContent = String(errored);
        updateThreadUsageChart(active, idle, errored);
    }

    function renderUrlUsage(stats = []) {
        if (!urlUsageList) return;
        urlUsageList.innerHTML = '';
        if (!Array.isArray(stats) || stats.length === 0) {
            urlUsageList.innerHTML = '<p class="text-muted text-center mb-0">暂无下载数据。</p>';
            updateUrlPieChart([]);
            return;
        }
        stats.forEach((item) => {
            const percent = Number(item.percentage || 0);
            const count = Number(item.count || 0);
            const url = item.url || '未知链接';
            const wrapper = document.createElement('div');
            wrapper.className = 'url-usage-entry mb-2';
            wrapper.innerHTML = `
                <div class="d-flex align-items-center gap-2 small mb-1 overflow-hidden">
                    <span class="usage-url flex-grow-1 text-truncate" title="${escapeAttr(url)}">${escapeHtml(url)}</span>
                    <span class="fw-bold text-nowrap">${percent.toFixed(1)}% · ${count} 次</span>
                </div>
                <div class="progress progress-sm">
                    <div class="progress-bar" style="width: ${Math.max(0, Math.min(100, percent))}%"></div>
                </div>
            `;
            urlUsageList.appendChild(wrapper);
        });
        updateUrlPieChart(stats);
    }

    function applyStatus(data = {}) {
        if (typeof data.running === 'boolean') applyRunningState(data.state, data.running);
        if (data.speed !== undefined) document.getElementById('speed-text').textContent = data.speed || '0 B/s';
        if (data.total_bytes !== undefined) document.getElementById('total-bytes').textContent = data.total_bytes || '0 B';
        if (data.download_count !== undefined) document.getElementById('download-count').textContent = data.download_count || '0';
        if (currentConfigEl && typeof data.config === 'string') {
            const name = data.config.trim() || '未命名配置';
            currentConfigEl.textContent = name;
            currentConfigEl.title = name;
        }
        if (data.speed) {
            const speedValue = data.speed.match(/(\d+(?:\.\d+)?)\s*MB\/s/i);
            addDataToChart(new Date().toLocaleTimeString(), speedValue ? parseFloat(speedValue[1]) : 0);
        }
        renderThreadStatus(data.thread_status, data.thread_count);
        renderUrlUsage(data.url_usage_stats);
    }

    async function pollStatus() {
        try {
            applyStatus(await apiFetch('/api/runtime/status'));
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    }

    async function pollScheduler() {
        try {
            const data = await apiFetch('/api/runtime/scheduler');
            jobDetailsEl.textContent = data.job_details || '无';
            stopSchedulerBtn.disabled = !data.job_details && !(Array.isArray(data.plans) && data.plans.some((plan) => plan.scheduler_running));
            renderRuntimePlans(data.plans || []);
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    }

    async function pollLogs() {
        if (!logSwitch || !logSwitch.checked) return;
        try {
            const data = await apiFetch(`/api/runtime/logs?since=${logCursor}`);
            logCursor = Number(data.latest || logCursor);
            (data.logs || []).forEach(renderLog);
        } catch (error) {
            pushAlert({ message: error.message, title: '日志错误：', variant: 'danger' });
        }
    }

    async function pollEvents() {
        try {
            const data = await apiFetch(`/api/runtime/events?since=${eventCursor}`);
            eventCursor = Number(data.latest || eventCursor);
            for (const event of (data.events || [])) {
                if (event.type === 'history_update') {
                    await pollScheduler();
                    await pollStatus();
                    continue;
                }
                if (event.type === 'status_update' || event.type === 'error' || event.type === 'download_failure' || event.event_type === 'invalid_url') {
                    pushAlert({
                        message: event.message || '发生未知事件。',
                        title: event.title || (event.type === 'error' ? '操作失败：' : '状态更新：'),
                        variant: event.variant || (event.type === 'error' ? 'danger' : 'info'),
                        error: event.error
                    });
                }
            }
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    }

    function renderLog(data = {}) {
        if (!logContainer) return;
        const initialMessage = logContainer.querySelector('.text-muted');
        if (initialMessage) initialMessage.remove();
        const logEntry = document.createElement('p');
        logEntry.className = 'mb-1 log-line';
        const timestampSpan = document.createElement('span');
        timestampSpan.className = 'text-muted me-2';
        timestampSpan.textContent = `[${new Date(data.timestamp || Date.now()).toLocaleTimeString()}]`;
        const messageSpan = document.createElement('span');
        messageSpan.textContent = data.message || '';
        if (data.color) messageSpan.style.color = data.color;
        logEntry.append(timestampSpan, messageSpan);
        logContainer.append(logEntry);
        logContainer.scrollTop = logContainer.scrollHeight;
        while (logContainer.children.length > 200) logContainer.removeChild(logContainer.firstChild);
    }

    function pushAlert(data = {}) {
        if (!notificationArea) return;
        const wrapper = document.createElement('div');
        wrapper.className = `alert alert-${data.variant || 'danger'} alert-dismissible fade show`;
        wrapper.setAttribute('role', 'alert');
        const title = document.createElement('strong');
        title.textContent = data.title || '下载失败：';
        const message = document.createElement('span');
        message.textContent = ` ${data.message || '发生未知错误。'}`;
        wrapper.append(title, message);
        if (data.error) {
            const detail = document.createElement('div');
            detail.className = 'small text-muted mt-1';
            detail.textContent = `详情：${data.error}`;
            wrapper.appendChild(detail);
        }
        const closeBtn = document.createElement('button');
        closeBtn.type = 'button';
        closeBtn.className = 'btn-close';
        closeBtn.setAttribute('data-bs-dismiss', 'alert');
        closeBtn.setAttribute('aria-label', '关闭');
        wrapper.appendChild(closeBtn);
        notificationArea.appendChild(wrapper);
        while (notificationArea.children.length > 3) notificationArea.removeChild(notificationArea.firstChild);
    }

    function formatDateTime(value) {
        if (!value) return '无';
        const date = new Date(typeof value === 'string' && !value.includes('T') ? value.replace(' ', 'T') : value);
        if (Number.isNaN(date.getTime())) return String(value);
        return date.toLocaleString();
    }

    function formatPlanSchedule(plan = {}) {
        if (plan.cron_expr) return `Cron ${plan.cron_expr}`;
        if (plan.interval) return `每 ${plan.interval} 分钟`;
        return '手动任务';
    }

    function renderRuntimePlans(plans = []) {
        if (!runtimePlanListEl) return;
        runtimePlanListEl.innerHTML = '';
        if (!Array.isArray(plans) || plans.length === 0) {
            runtimePlanListEl.innerHTML = '<div class="text-muted small py-2">暂无运行计划。</div>';
            return;
        }
        plans.forEach((plan) => {
            const card = document.createElement('div');
            card.className = 'plan-card';
            const runState = plan.running ? '执行中' : (plan.scheduler_running ? '等待调度' : '已停止');
            card.innerHTML = `
                <div class="plan-card-header">
                    <div>
                        <div class="plan-card-title">${escapeHtml(plan.name || '未命名计划')}</div>
                        <div class="plan-card-subtitle">${escapeHtml(formatPlanSchedule(plan))} · ${runState}</div>
                    </div>
                    <span class="badge ${plan.running ? 'bg-success' : (plan.scheduler_running ? 'bg-info text-dark' : 'bg-secondary')}">${plan.running ? '运行中' : (plan.scheduler_running ? '计划中' : '已停止')}</span>
                </div>
                <div class="plan-card-metrics">
                    <div class="plan-card-metric"><span class="label">下次执行</span><span class="value">${formatDateTime(plan.next_run_time)}</span></div>
                    <div class="plan-card-metric"><span class="label">累计流量</span><span class="value">${escapeHtml(plan.total_bytes || '0 B')}</span></div>
                    <div class="plan-card-metric"><span class="label">累计下载数</span><span class="value">${plan.download_count ?? 0}</span></div>
                    <div class="plan-card-metric"><span class="label">线程 / 链接</span><span class="value">${plan.threads ?? 0} / ${plan.url_count ?? 0}</span></div>
                </div>
                <div class="plan-card-actions">
                    <button type="button" class="btn btn-sm btn-outline-primary" data-plan-action="detail" data-plan-name="${escapeAttr(plan.name || '')}">查看详情</button>
                    <button type="button" class="btn btn-sm btn-outline-danger" data-plan-action="stop" data-plan-name="${escapeAttr(plan.name || '')}" ${plan.running || plan.scheduler_running ? '' : 'disabled'}>停止该计划</button>
                </div>
            `;
            runtimePlanListEl.appendChild(card);
        });
    }

    function renderPlanDetail(payload = {}) {
        const summary = payload.summary || {};
        if (planDetailNameEl) planDetailNameEl.textContent = payload.name || '-';
        if (planDetailNextRunEl) planDetailNextRunEl.textContent = formatDateTime(summary.next_run_time);
        if (planDetailTotalEl) planDetailTotalEl.textContent = `${summary.total_bytes || '0 B'} / ${summary.download_count ?? 0}`;
        renderDetailHistory(payload.history || []);
        if (planDetailModal) planDetailModal.show();
    }

    function renderDetailHistory(history = []) {
        if (!planDetailHistoryBody) return;
        planDetailHistoryBody.innerHTML = '';
        if (!Array.isArray(history) || history.length === 0) {
            planDetailHistoryBody.innerHTML = '<tr class="no-history"><td colspan="4" class="text-center text-muted py-3">暂无执行记录。</td></tr>';
            return;
        }
        history.forEach((item) => {
            const row = document.createElement('tr');
            row.innerHTML = `<td>${formatDateTime(item.timestamp)}</td><td>${escapeHtml(item.result || '未知')}</td><td>${escapeHtml(item.bytes_consumed || '0 B')}</td><td>${item.download_count ?? 0}</td>`;
            planDetailHistoryBody.appendChild(row);
        });
    }

    startBtn.addEventListener('click', async () => {
        if (!selectedConfigName || !selectedConfigDetail) {
            pushAlert({ message: '请选择有效的运行配置后再启动。', title: '提示：', variant: 'warning' });
            return;
        }
        try {
            applyStatus(await apiFetch('/api/runtime/start', { method: 'POST', body: JSON.stringify(normalizeConfigPayload(selectedConfigDetail, selectedConfigName)) }));
            await pollScheduler();
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    });

    if (tempTestBtn) {
        tempTestBtn.addEventListener('click', async () => {
            if (!selectedConfigName || !selectedConfigDetail) {
                pushAlert({ message: '请选择有效的运行配置后再发起临时测试。', title: '提示：', variant: 'warning' });
                return;
            }
            try {
                applyStatus(await apiFetch('/api/runtime/temp-test', { method: 'POST', body: JSON.stringify(normalizeConfigPayload(selectedConfigDetail, selectedConfigName)) }));
                await pollScheduler();
            } catch (error) {
                pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
            }
        });
    }

    stopBtn.addEventListener('click', async () => {
        try {
            await apiFetch('/api/runtime/stop', { method: 'POST', body: '{}' });
            await pollStatus();
            await pollScheduler();
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    });

    stopSchedulerBtn.addEventListener('click', async () => {
        try {
            await apiFetch('/api/runtime/scheduler/stop', { method: 'POST', body: '{}' });
            await pollStatus();
            await pollScheduler();
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    });

    if (runtimePlanListEl) {
        runtimePlanListEl.addEventListener('click', async (event) => {
            const button = event.target.closest('[data-plan-action]');
            if (!button) return;
            const planName = button.dataset.planName || '';
            if (!planName) {
                pushAlert({ message: '无法识别目标计划。', title: '提示：', variant: 'warning' });
                return;
            }
            try {
                if (button.dataset.planAction === 'detail') {
                    renderPlanDetail(await apiFetch(`/api/runtime/plans/${encodedName(planName)}`));
                    return;
                }
                if (window.confirm(`确定停止计划 "${planName}" 吗？`)) {
                    await apiFetch(`/api/runtime/plans/${encodedName(planName)}/stop`, { method: 'POST', body: '{}' });
                    await pollStatus();
                    await pollScheduler();
                }
            } catch (error) {
                pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
            }
        });
    }

    if (deleteConfigBtn) {
        deleteConfigBtn.addEventListener('click', async () => {
            const targetName = editorActiveConfig || selectedConfigName || (configInputs.name && configInputs.name.value.trim());
            if (!targetName) {
                pushAlert({ message: '请先选择或输入要删除的配置。', title: '提示：', variant: 'warning' });
                return;
            }
            if (!window.confirm(`确定删除配置 "${targetName}" 吗？`)) return;
            try {
                await apiFetch(`/api/configs/${encodedName(targetName)}`, { method: 'DELETE' });
                await loadConfigs();
                await pollScheduler();
            } catch (error) {
                pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
            }
        });
    }

    saveConfigBtn.addEventListener('click', async () => {
        const config = getConfigFromForm();
        if (!config.name) {
            pushAlert({ message: '请填写配置名称后再保存。', title: '提示：', variant: 'warning' });
            return;
        }
        try {
            await apiFetch(`/api/configs/${encodedName(config.name)}`, { method: 'PUT', body: JSON.stringify(config) });
            editorActiveConfig = config.name;
            await loadConfigs();
            pushAlert({ message: `配置 "${config.name}" 已保存。`, title: '状态更新：', variant: 'success' });
        } catch (error) {
            pushAlert({ message: error.message, title: '操作失败：', variant: 'danger' });
        }
    });

    if (configSelect) {
        configSelect.addEventListener('change', () => {
            selectedConfigName = configSelect.value || null;
            selectedConfigDetail = null;
            if (selectedConfigName) requestConfigDetails(selectedConfigName, 'runtime');
        });
    }

    if (editorConfigSelect) {
        editorConfigSelect.addEventListener('change', () => {
            const value = editorConfigSelect.value;
            if (!value) {
                resetConfigForm(true);
                return;
            }
            requestConfigDetails(value, 'editor');
        });
    }

    if (configEditorEl) {
        configEditorEl.addEventListener('show.bs.offcanvas', () => {
            const preferredName = selectedConfigName || editorActiveConfig || '';
            if (preferredName) {
                if (editorConfigSelect) editorConfigSelect.value = preferredName;
                requestConfigDetails(preferredName, 'editor');
                return;
            }
            resetConfigForm(true);
        });
    }

    if (resetConfigBtn) resetConfigBtn.addEventListener('click', () => resetConfigForm());

    if (logSwitch) {
        logSwitch.addEventListener('change', () => {
            if (logSwitch.checked) {
                const initialMessage = logContainer.querySelector('.text-muted');
                if (initialMessage) initialMessage.textContent = '暂无日志。';
                pollLogs();
            }
        });
    }

    if (clearLogBtn) {
        clearLogBtn.addEventListener('click', () => {
            logContainer.innerHTML = '<p class="text-muted">日志已清空。</p>';
        });
    }

    let debounceTimer;
    if (configInputs.cron_expr) {
        configInputs.cron_expr.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(async () => {
                const cronExpr = configInputs.cron_expr.value;
                if (!cronExpr) {
                    cronPreviewEl.innerHTML = '';
                    return;
                }
                try {
                    const data = await apiFetch('/api/preview_cron', {
                        method: 'POST',
                        body: JSON.stringify({ cron_expr: cronExpr })
                    });
                    cronPreviewEl.innerHTML = `<strong>接下来5次运行时间:</strong><ul>${data.map((ts) => `<li>${formatDateTime(ts)}</li>`).join('')}</ul>`;
                } catch (error) {
                    cronPreviewEl.innerHTML = `<span class="text-danger">错误: ${escapeHtml(error.message)}</span>`;
                }
            }, 500);
        });
    }

    document.querySelectorAll('.cron-preset').forEach((button) => {
        button.addEventListener('click', (event) => {
            event.preventDefault();
            if (!configInputs.cron_expr) return;
            configInputs.cron_expr.value = event.currentTarget.dataset.cron;
            configInputs.cron_expr.dispatchEvent(new Event('input'));
        });
    });

    function escapeHtml(value) {
        return String(value ?? '').replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
    }

    function escapeAttr(value) {
        return escapeHtml(value).replace(/`/g, '&#96;');
    }

    loadConfigs();
    pollStatus();
    pollScheduler();
    pollEvents();
    setInterval(pollStatus, 1000);
    setInterval(pollScheduler, 2000);
    setInterval(pollLogs, 1000);
    setInterval(pollEvents, 1000);
});
