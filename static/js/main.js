AOS.init({ once: true });

document.addEventListener('DOMContentLoaded', (event) => {
    const socket = io.connect(location.protocol + '//' + document.domain + ':' + location.port);

    // --- 元素获取 ---
    const startBtn = document.getElementById('start-btn');
    const stopBtn = document.getElementById('stop-btn');
    const stopSchedulerBtn = document.getElementById('stop-scheduler-btn');
    const saveConfigBtn = document.getElementById('save-config-btn');
    const configSelect = document.getElementById('config-select');
    const runningStatus = document.getElementById('running-status');
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
        auto_remove_failed_url: document.getElementById('auto-remove-failed-url')
    };
    const jobDetailsEl = document.getElementById('job-details');
    const nextRunTimeEl = document.getElementById('next-run-time');
    const countdownEl = document.getElementById('countdown');
    const historyTableBody = document.getElementById('history-table-body');
    const cronPreviewEl = document.getElementById('cron-preview');
    const logSwitch = document.getElementById('log-switch');
    const logContainer = document.getElementById('log-container');
    const clearLogBtn = document.getElementById('clear-log-btn');
    const editorConfigSelect = document.getElementById('config-editor-select');
    const resetConfigBtn = document.getElementById('reset-config-btn');
    const threadStatusList = document.getElementById('thread-status-list');
    const urlUsageList = document.getElementById('url-usage-list');
    const notificationArea = document.getElementById('notification-area');
    const activeThreadCountEl = document.getElementById('active-thread-count');
    const idleThreadCountEl = document.getElementById('idle-thread-count');
    const totalThreadCountEl = document.getElementById('total-thread-count');
    const erroredThreadCountEl = document.getElementById('errored-thread-count');
    const currentConfigEl = document.getElementById('current-config');
    const multiConfigSelect = document.getElementById('multi-config-select');
    const startMultiBtn = document.getElementById('start-multi-btn');
    const stopMultiBtn = document.getElementById('stop-multi-btn');
    const multiJobList = document.getElementById('multi-job-list');

    let selectedConfigName = null;
    let selectedConfigDetail = null;
    let editorActiveConfig = null;
    let pendingConfigTarget = null;
    let pendingConfigName = null;

    // --- Chart.js 初始化 ---
    const speedChartCtx = document.getElementById('speed-chart').getContext('2d');
    const speedChart = new Chart(speedChartCtx, {
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
                y: {
                    beginAtZero: true,
                    ticks: { color: '#FF69B4' }
                }
            },
            plugins: {
                legend: { display: false }
            }
        }
    });

    const urlPieChartCanvas = document.getElementById('url-pie-chart');
    const urlPieChart = urlPieChartCanvas ? new Chart(urlPieChartCanvas.getContext('2d'), {
        type: 'doughnut',
        data: {
            labels: ['无数据'],
            datasets: [{
                data: [1],
                backgroundColor: ['rgba(255, 228, 240, 0.9)'],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '58%',
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        boxWidth: 14,
                        boxHeight: 14,
                        padding: 12,
                        color: '#666'
                    }
                },
                tooltip: {
                    callbacks: {
                        label(context) {
                            const label = context.label || '';
                            const value = context.parsed;
                            return `${label}: ${value} 次`;
                        }
                    }
                }
            }
        }
    }) : null;

    const threadUsageChartCanvas = document.getElementById('thread-usage-chart');
    const threadUsageChart = threadUsageChartCanvas ? new Chart(threadUsageChartCanvas.getContext('2d'), {
        type: 'doughnut',
        data: {
            labels: ['活跃', '空闲', '失效'],
            datasets: [{
                data: [0, 0, 0],
                backgroundColor: [
                    'rgba(255, 105, 180, 0.85)',
                    'rgba(255, 182, 193, 0.85)',
                    'rgba(220, 53, 69, 0.75)'
                ],
                borderWidth: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            cutout: '65%',
            plugins: {
                legend: {
                    position: 'bottom',
                    labels: {
                        boxWidth: 12,
                        boxHeight: 12,
                        padding: 10,
                        color: '#666'
                    }
                },
                tooltip: {
                    callbacks: {
                        label(context) {
                            const label = context.label || '';
                            const value = context.parsed;
                            return `${label}: ${value} 条线程`;
                        }
                    }
                }
            }
        }
    }) : null;


    function addDataToChart(label, data) {
        speedChart.data.labels.push(label);
        speedChart.data.datasets.forEach((dataset) => {
            dataset.data.push(data);
        });
        if (speedChart.data.labels.length > 30) {
            speedChart.data.labels.shift();
            speedChart.data.datasets[0].data.shift();
        }
        speedChart.update('none'); // 'none' for no animation
    }

    function updateUrlPieChart(stats = []) {
        if (!urlPieChart) return;

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

        const labels = [];
        const values = [];
        const colors = [];

        stats.forEach((item, index) => {
            const label = item.url || `链接 ${index + 1}`;
            const value = Number(item.count) || 0;
            labels.push(label);
            values.push(value);
            colors.push(palette[index % palette.length]);
        });

        urlPieChart.data.labels = labels;
        urlPieChart.data.datasets[0].data = values;
        urlPieChart.data.datasets[0].backgroundColor = colors;
        urlPieChart.update('none');
    }

    function updateThreadUsageChart(active = 0, idle = 0, errored = 0) {
        if (!threadUsageChart) return;
        threadUsageChart.data.datasets[0].data = [active, idle, errored];
        threadUsageChart.update('none');
    }

    function resetConfigForm(keepSelection = false) {
        Object.keys(configInputs).forEach((key) => {
            const element = configInputs[key];
            if (!element) return;
            if (element.type === 'checkbox') {
                element.checked = false;
            } else if (element.tagName === 'TEXTAREA' || element.tagName === 'INPUT') {
                element.value = '';
            } else {
                element.value = '';
            }
        });
        if (!keepSelection && editorConfigSelect) {
            editorConfigSelect.value = '';
        }
        if (cronPreviewEl) {
            cronPreviewEl.innerHTML = '';
        }
        editorActiveConfig = null;
        if (configInputs.cron_expr) {
            configInputs.cron_expr.dispatchEvent(new Event('input'));
        }
    }

    function populateEditorForm(name, config = {}) {
        if (configInputs.name) {
            configInputs.name.value = name || '';
        }
        if (configInputs.urls) {
            const urls = Array.isArray(config.urls) ? config.urls : [];
            configInputs.urls.value = urls.join('\n');
        }
        if (configInputs.threads) {
            configInputs.threads.value = config.threads ?? '';
        }
        if (configInputs.limit_speed) {
            configInputs.limit_speed.value = config.limit_speed ?? '';
        }
        if (configInputs.traffic_limit) {
            configInputs.traffic_limit.value = config.traffic_limit ?? '';
        }
        if (configInputs.duration) {
            configInputs.duration.value = config.duration ?? '';
        }
        if (configInputs.count) {
            configInputs.count.value = config.count ?? '';
        }
        if (configInputs.cron_expr) {
            configInputs.cron_expr.value = config.cron_expr ?? '';
        }
        if (configInputs.interval) {
            configInputs.interval.value = config.interval ?? '';
        }
        if (configInputs.url_strategy) {
            configInputs.url_strategy.value = config.url_strategy ?? '';
        }
        if (configInputs.auto_remove_failed_url) {
            configInputs.auto_remove_failed_url.checked = Boolean(config.auto_remove_failed_url);
        }
        editorActiveConfig = name || null;
        if (cronPreviewEl) {
            cronPreviewEl.innerHTML = '';
        }
        if (configInputs.cron_expr) {
            configInputs.cron_expr.dispatchEvent(new Event('input'));
        }
    }

    function normalizeConfigPayload(config = {}, name = null) {
        const payload = {
            urls: Array.isArray(config.urls) ? [...config.urls] : [],
            url_strategy: config.url_strategy ?? null,
            threads: config.threads ?? null,
            limit_speed: config.limit_speed ?? null,
            traffic_limit: config.traffic_limit ?? null,
            duration: config.duration ?? null,
            count: config.count ?? null,
            cron_expr: config.cron_expr ?? null,
            interval: config.interval ?? null,
            config_name: name || config.config_name || null,
            auto_remove_failed_url: Boolean(config.auto_remove_failed_url)
        };

        payload.name = name || config.name || '';

        payload.urls = payload.urls
            .map((url) => {
                if (typeof url === 'string') {
                    return url.trim();
                }
                return url != null ? String(url).trim() : '';
            })
            .filter((url) => url !== '');

        const integerKeys = ['threads', 'traffic_limit', 'duration', 'count', 'interval'];
        integerKeys.forEach((key) => {
            if (payload[key] === null || payload[key] === undefined || payload[key] === '') {
                payload[key] = null;
                return;
            }
            const parsed = parseInt(payload[key], 10);
            payload[key] = Number.isFinite(parsed) ? parsed : null;
        });

        if (payload.limit_speed !== null && payload.limit_speed !== undefined && payload.limit_speed !== '') {
            const parsed = parseFloat(payload.limit_speed);
            payload.limit_speed = Number.isFinite(parsed) ? parsed : null;
        } else {
            payload.limit_speed = null;
        }

        if (!payload.url_strategy) {
            payload.url_strategy = null;
        }

        return payload;
    }

    function requestConfigDetails(name, target) {
        if (!name) {
            if (target === 'runtime') {
                selectedConfigName = null;
                selectedConfigDetail = null;
                if (currentConfigEl) {
                    currentConfigEl.textContent = '未选择';
                    currentConfigEl.title = '未选择';
                }
            }
            if (target === 'editor') {
                resetConfigForm(true);
            }
            return;
        }
        if (target === 'runtime') {
            selectedConfigName = name;
            if (currentConfigEl) {
                const displayName = name || '未命名配置';
                currentConfigEl.textContent = displayName;
                currentConfigEl.title = displayName;
            }
        }
        if (target === 'editor') {
            editorActiveConfig = name;
        }
        pendingConfigTarget = target;
        pendingConfigName = name;
        socket.emit('get_config_details', { name, target });
    }

    function renderThreadStatus(threadMap = {}, totalThreads = 0) {
        if (!threadStatusList) return;

        const entries = Object.entries(threadMap || {});
        const expectedThreads = Number(totalThreads) || entries.length;

        threadStatusList.innerHTML = '';

        if (!expectedThreads) {
            threadStatusList.innerHTML = '<tr class="placeholder-row"><td colspan="3" class="text-center text-muted py-3">暂无线程数据。</td></tr>';
            if (totalThreadCountEl) totalThreadCountEl.textContent = '0';
            if (activeThreadCountEl) activeThreadCountEl.textContent = '0';
            if (idleThreadCountEl) idleThreadCountEl.textContent = '0';
            if (erroredThreadCountEl) erroredThreadCountEl.textContent = '0';
            updateThreadUsageChart(0, 0, 0);
            return;
        }

        const mapped = new Map(entries.map(([key, value]) => [Number(key), value]));
        let activeCount = 0;
        let idleCount = 0;
        let errorCount = 0;

        for (let i = 1; i <= expectedThreads; i += 1) {
            const rawValue = mapped.has(i) ? mapped.get(i) : null;
            let displayUrl = rawValue;
            if (displayUrl === null || displayUrl === undefined) {
                displayUrl = '';
            } else if (typeof displayUrl !== 'string') {
                displayUrl = String(displayUrl);
            }
            let status = 'idle';

            if (!displayUrl || displayUrl.trim() === '' || displayUrl.trim() === '等待分配...') {
                displayUrl = '等待分配...';
                status = 'idle';
            } else if (displayUrl.includes('已失效') || displayUrl.includes('无可用')) {
                status = 'error';
            } else {
                status = 'active';
            }

            const row = document.createElement('tr');

            const threadCell = document.createElement('td');
            threadCell.className = 'text-muted';
            threadCell.textContent = `线程 ${i}`;
            row.appendChild(threadCell);

            const urlCell = document.createElement('td');
            const urlSpan = document.createElement('span');
            urlSpan.className = 'thread-url text-truncate';
            const tooltipText = status === 'idle' ? '' : displayUrl;
            urlSpan.textContent = displayUrl || '等待分配...';
            if (tooltipText) {
                urlSpan.title = tooltipText;
            } else {
                urlSpan.removeAttribute('title');
            }
            if (status === 'error') {
                urlSpan.classList.add('text-danger');
            }
            urlCell.appendChild(urlSpan);
            row.appendChild(urlCell);

            const statusCell = document.createElement('td');
            statusCell.className = 'text-end';
            const badge = document.createElement('span');
            badge.className = 'badge thread-status-badge';

            if (status === 'active') {
                badge.classList.add('bg-success');
                badge.textContent = '运行中';
                activeCount += 1;
            } else if (status === 'error') {
                badge.classList.add('bg-danger');
                badge.textContent = displayUrl.includes('无可用') ? '耗尽' : '失效';
                errorCount += 1;
            } else {
                badge.classList.add('bg-secondary');
                badge.textContent = '待命';
                idleCount += 1;
            }

            statusCell.appendChild(badge);
            row.appendChild(statusCell);

            threadStatusList.appendChild(row);
        }

        if (totalThreadCountEl) totalThreadCountEl.textContent = expectedThreads.toString();
        if (activeThreadCountEl) activeThreadCountEl.textContent = activeCount.toString();
        if (idleThreadCountEl) idleThreadCountEl.textContent = idleCount.toString();
        if (erroredThreadCountEl) erroredThreadCountEl.textContent = errorCount.toString();

        updateThreadUsageChart(activeCount, idleCount, errorCount);
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
            const percent = Number(item.percentage ?? 0);
            const safePercent = Number.isFinite(percent) ? percent : 0;
            const safeCount = Number.isFinite(Number(item.count)) ? Number(item.count) : 0;
            const safeUrl = item.url || '未知链接';

            const wrapper = document.createElement('div');
            wrapper.className = 'url-usage-entry mb-2';

            const header = document.createElement('div');
            header.className = 'd-flex align-items-center gap-2 small mb-1 overflow-hidden';
            const urlLabel = document.createElement('span');
            urlLabel.className = 'usage-url flex-grow-1 text-truncate';
            urlLabel.textContent = safeUrl;
            urlLabel.title = safeUrl;

            const statLabel = document.createElement('span');
            statLabel.className = 'fw-bold text-nowrap';
            statLabel.textContent = `${safePercent.toFixed(1)}% · ${safeCount} 次`;

            header.appendChild(urlLabel);
            header.appendChild(statLabel);

            const progressOuter = document.createElement('div');
            progressOuter.className = 'progress progress-sm';

            const progressBar = document.createElement('div');
            progressBar.className = 'progress-bar';
            const width = Math.max(0, Math.min(100, safePercent));
            progressBar.style.width = `${width}%`;
            progressBar.setAttribute('aria-valuenow', width.toString());
            progressBar.setAttribute('aria-valuemin', '0');
            progressBar.setAttribute('aria-valuemax', '100');

            progressOuter.appendChild(progressBar);
            wrapper.appendChild(header);
            wrapper.appendChild(progressOuter);

            urlUsageList.appendChild(wrapper);
        });

        updateUrlPieChart(stats);
    }

    function renderMultiJobs(jobs = []) {
        if (!multiJobList) return;
        multiJobList.innerHTML = '';
        if (!Array.isArray(jobs) || jobs.length === 0) {
            multiJobList.innerHTML = '<div class="text-muted small px-2 py-2">暂无多任务计划。</div>';
            return;
        }

        jobs.forEach((job) => {
            const item = document.createElement('div');
            item.className = 'list-group-item';

            const title = document.createElement('div');
            title.className = 'job-title';
            title.textContent = job.config_name || '未命名配置';

            const meta = document.createElement('div');
            meta.className = 'job-meta';
            const statusText = job.running ? '运行中' : '已停止';
            const detailText = job.job_details || '未配置';
            const nextRun = job.next_run_time ? new Date(job.next_run_time).toLocaleString() : '未计划';
            meta.textContent = `${statusText} · ${detailText} · 下次: ${nextRun}`;

            item.appendChild(title);
            item.appendChild(meta);
            multiJobList.appendChild(item);
        });
    }

    function pushAlert(data = {}) {
        if (!notificationArea) return;
        const wrapper = document.createElement('div');
        wrapper.className = 'alert alert-danger alert-dismissible fade show';
        wrapper.setAttribute('role', 'alert');

        const title = document.createElement('strong');
        const prefix = data.config_name ? `[${data.config_name}] ` : '';
        title.textContent = `${prefix}下载失败：`;

        const message = document.createElement('span');
        const baseMessage = data.message || (data.url ? `链接 ${data.url} 已连续失败，已停止重试。` : '存在下载链接失效，已停止重试。');
        message.textContent = ` ${baseMessage}`;

        const detail = document.createElement('div');
        if (data.error) {
            detail.className = 'small text-muted mt-1';
            detail.textContent = `详情：${data.error}`;
        }

        const closeBtn = document.createElement('button');
        closeBtn.type = 'button';
        closeBtn.className = 'btn-close';
        closeBtn.setAttribute('data-bs-dismiss', 'alert');
        closeBtn.setAttribute('aria-label', '关闭');

        wrapper.appendChild(title);
        wrapper.appendChild(message);
        if (data.error) {
            wrapper.appendChild(detail);
        }
        wrapper.appendChild(closeBtn);

        notificationArea.appendChild(wrapper);

        while (notificationArea.children.length > 3) {
            notificationArea.removeChild(notificationArea.firstChild);
        }
    }

    function pushFeedback({ message, level = 'info' }) {
        if (!notificationArea || !message) return;
        const wrapper = document.createElement('div');
        wrapper.className = `alert alert-${level} alert-dismissible fade show`;
        wrapper.setAttribute('role', 'alert');
        wrapper.textContent = message;
        const closeBtn = document.createElement('button');
        closeBtn.type = 'button';
        closeBtn.className = 'btn-close';
        closeBtn.setAttribute('data-bs-dismiss', 'alert');
        closeBtn.setAttribute('aria-label', '关闭');
        wrapper.appendChild(closeBtn);
        notificationArea.appendChild(wrapper);
        while (notificationArea.children.length > 3) {
            notificationArea.removeChild(notificationArea.firstChild);
        }
    }

    // Bootstrap 管理标签切换，无需额外 JS

    // --- Socket.IO 事件处理 ---
    socket.on('connect', () => {
        console.log('已连接到服务器');
        socket.emit('get_configs');
    });

    socket.on('configs_list', (data) => {
        const configs = Array.isArray(data.configs) ? data.configs : [];

        if (configSelect) {
            const previousSelection = selectedConfigName;
            configSelect.innerHTML = '';
            const placeholder = document.createElement('option');
            placeholder.value = '';
            placeholder.textContent = configs.length ? '请选择运行配置' : '暂无配置，请先创建';
            configSelect.appendChild(placeholder);

            configs.forEach((name) => {
                const option = document.createElement('option');
                option.value = name;
                option.textContent = name;
                configSelect.appendChild(option);
            });

            if (configs.length > 0) {
                const runtimeSelection = configs.includes(previousSelection) ? previousSelection : configs[0];
                selectedConfigName = runtimeSelection;
                configSelect.value = runtimeSelection;
                requestConfigDetails(runtimeSelection, 'runtime');
            } else {
                selectedConfigName = null;
                selectedConfigDetail = null;
                configSelect.value = '';
            }
        }

        if (editorConfigSelect) {
            const previousEditor = editorActiveConfig || editorConfigSelect.value;
            editorConfigSelect.innerHTML = '';
            const editorPlaceholder = document.createElement('option');
            editorPlaceholder.value = '';
            editorPlaceholder.textContent = '新建配置';
            editorConfigSelect.appendChild(editorPlaceholder);

            configs.forEach((name) => {
                const option = document.createElement('option');
                option.value = name;
                option.textContent = name;
                editorConfigSelect.appendChild(option);
            });

            if (previousEditor && configs.includes(previousEditor)) {
                editorConfigSelect.value = previousEditor;
                requestConfigDetails(previousEditor, 'editor');
            } else {
                editorActiveConfig = null;
                editorConfigSelect.value = '';
            }
        }

        if (multiConfigSelect) {
            multiConfigSelect.innerHTML = '';
            if (!configs.length) {
                const option = document.createElement('option');
                option.disabled = true;
                option.textContent = '暂无配置';
                multiConfigSelect.appendChild(option);
            } else {
                configs.forEach((name) => {
                    const option = document.createElement('option');
                    option.value = name;
                    option.textContent = name;
                    multiConfigSelect.appendChild(option);
                });
            }
        }
    });

    socket.on('config_details', (data) => {
        const target = data.target || pendingConfigTarget || 'runtime';
        const name = data.name;
        const config = data.config || {};

        if (target === 'editor') {
            populateEditorForm(name, config);
            editorActiveConfig = name;
            if (editorConfigSelect && editorConfigSelect.value !== name) {
                editorConfigSelect.value = name;
            }
        } else {
            selectedConfigName = name;
            selectedConfigDetail = normalizeConfigPayload(config, name);
            if (configSelect && configSelect.value !== name) {
                configSelect.value = name;
            }
        }

        if (pendingConfigName === name) {
            pendingConfigName = null;
            pendingConfigTarget = null;
        }
    });

    socket.on('status_update', (data) => {
        if (data.running) {
            runningStatus.textContent = '运行中';
            runningStatus.className = 'badge bg-success';
        } else {
            runningStatus.textContent = '已停止';
            runningStatus.className = 'badge bg-secondary';
        }
        document.getElementById('speed-text').textContent = data.speed || '0 B/s';
        document.getElementById('total-bytes').textContent = data.total_bytes || '0 B';
        document.getElementById('download-count').textContent = data.download_count || '0';
        if (currentConfigEl) {
            const safeConfigName = typeof data.config === 'string' && data.config.trim()
                ? data.config.trim()
                : '未命名配置';
            currentConfigEl.textContent = safeConfigName;
            currentConfigEl.title = safeConfigName;
        }

        const speedValue = data.speed ? data.speed.match(/(\d+\.\d+)\s*MB\/s/i) : null;
        const speedMB = speedValue ? parseFloat(speedValue[1]) : 0;
        addDataToChart(new Date().toLocaleTimeString(), speedMB);

        startBtn.disabled = data.running;
        stopBtn.disabled = !data.running;

        renderThreadStatus(data.thread_status, data.thread_count);
        renderUrlUsage(data.url_usage_stats);
    });

    socket.on('history_update', (record) => {
        const row = historyTableBody.insertRow(0);
        const configLabel = record.config_name || record.config || '未命名配置';
        row.innerHTML = `<td>${new Date(record.timestamp).toLocaleString()}</td><td>${configLabel}</td><td>${record.result}</td><td>${record.bytes_consumed}</td><td>${record.download_count || 'N/A'}</td>`;
        const noHistoryRow = historyTableBody.querySelector('.no-history');
        if (noHistoryRow) noHistoryRow.remove();
        while (historyTableBody.rows.length > 50) {
            historyTableBody.deleteRow(historyTableBody.rows.length - 1);
        }
    });

    socket.on('log_message', (data = {}) => {
        if (!logSwitch.checked) return;
        const initialMessage = logContainer.querySelector('.text-muted');
        if (initialMessage) {
            initialMessage.remove();
        }

        const logEntry = document.createElement('p');
        logEntry.className = 'mb-1 log-line';

        const timestampSpan = document.createElement('span');
        timestampSpan.className = 'text-muted me-2';
        timestampSpan.textContent = `[${new Date().toLocaleTimeString()}]`;

        const messageSpan = document.createElement('span');
        messageSpan.textContent = data.message || '';
        if (data.color) {
            messageSpan.style.color = data.color;
        }

        logEntry.append(timestampSpan, messageSpan);
        logContainer.append(logEntry);

        logContainer.scrollTop = logContainer.scrollHeight;

        while (logContainer.children.length > 200) {
            logContainer.removeChild(logContainer.firstChild);
        }
    });

    socket.on('invalid_url', (data) => {
        pushAlert(data);
    });

    socket.on('multi_scheduler_feedback', (payload = {}) => {
        pushFeedback(payload);
    });

    let countdownInterval;
    socket.on('scheduler_status_update', (data) => {
        jobDetailsEl.textContent = data.job_details || '无';
        stopSchedulerBtn.disabled = !data.job_details;

        if (data.next_run_time) {
            const nextRunDate = new Date(data.next_run_time);
            nextRunTimeEl.textContent = nextRunDate.toLocaleString();
            if (countdownInterval) clearInterval(countdownInterval);
            countdownInterval = setInterval(() => {
                const diff = nextRunDate - new Date();
                if (diff <= 0) {
                    countdownEl.textContent = '运行中...';
                    clearInterval(countdownInterval);
                    return;
                }
                const h = Math.floor(diff / 3600000).toString().padStart(2, '0');
                const m = Math.floor((diff % 3600000) / 60000).toString().padStart(2, '0');
                const s = Math.floor((diff % 60000) / 1000).toString().padStart(2, '0');
                countdownEl.textContent = `${h}:${m}:${s}`;
            }, 1000);
        } else {
            nextRunTimeEl.textContent = '无';
            countdownEl.textContent = '无';
            if (countdownInterval) clearInterval(countdownInterval);
        }

        historyTableBody.innerHTML = '';
        if (data.history && data.history.length > 0) {
            data.history.forEach(item => {
                const configLabel = item.config_name || item.config || '未命名配置';
                const row = historyTableBody.insertRow();
                row.innerHTML = `<td>${new Date(item.timestamp).toLocaleString()}</td><td>${configLabel}</td><td>${item.result}</td><td>${item.bytes_consumed}</td><td>${item.download_count || 'N/A'}</td>`;
            });
        } else {
            historyTableBody.innerHTML = '<tr class="no-history text-center"><td colspan="5">暂无历史记录</td></tr>';
        }

        renderMultiJobs(Array.isArray(data.multi_jobs) ? data.multi_jobs : []);
    });
    
    // --- 事件监听 ---
    function getConfigFromForm() {
        const raw = {};
        Object.keys(configInputs).forEach((key) => {
            const element = configInputs[key];
            if (!element) {
                raw[key] = null;
                return;
            }
            if (element.type === 'checkbox') {
                raw[key] = Boolean(element.checked);
                return;
            }
            if (key === 'urls') {
                raw.urls = element.value || '';
                return;
            }
            const value = typeof element.value === 'string' ? element.value.trim() : element.value;
            raw[key] = value === '' ? null : value;
        });

        raw.urls = raw.urls
            ? raw.urls
                .split(/\r?\n/)
                .map((url) => url.trim())
                .filter((url) => url !== '')
            : [];

        const normalized = normalizeConfigPayload(raw, raw.name || null);
        normalized.name = raw.name || '';
        return normalized;
    }

    function getSelectedMultiConfigs(selectElement = multiConfigSelect) {
        if (!selectElement) return [];
        return Array.from(selectElement.selectedOptions || [])
            .map((option) => option.value)
            .filter((value) => value);
    }

    startBtn.addEventListener('click', () => {
        if (!selectedConfigName || !selectedConfigDetail) {
            pushAlert({ message: '请选择有效的运行配置后再启动。' });
            return;
        }
        const payload = normalizeConfigPayload(selectedConfigDetail, selectedConfigName);
        payload.config_name = selectedConfigName;
        payload.name = selectedConfigName;
        payload.urls = Array.isArray(payload.urls) ? [...payload.urls] : [];
        socket.emit('start_consumer', payload);
    });

    stopBtn.addEventListener('click', () => {
        socket.emit('stop_consumer');
    });

    stopSchedulerBtn.addEventListener('click', () => socket.emit('stop_scheduler'));

    if (startMultiBtn) {
        startMultiBtn.addEventListener('click', () => {
            const selections = getSelectedMultiConfigs();
            if (!selections.length) {
                pushFeedback({ level: 'warning', message: '请先选择要启动的配置。' });
                return;
            }
            socket.emit('start_multi_configs', { config_names: selections });
        });
    }

    if (stopMultiBtn) {
        stopMultiBtn.addEventListener('click', () => {
            socket.emit('stop_multi_configs', {});
        });
    }

    saveConfigBtn.addEventListener('click', () => {
        const config = getConfigFromForm();
        if (!config.name) {
            pushAlert({ message: '请填写配置名称后再保存。' });
            return;
        }
        editorActiveConfig = config.name;
        socket.emit('save_config', { name: config.name, data: config });
    });

    if (configSelect) {
        configSelect.addEventListener('change', () => {
            const value = configSelect.value;
            if (!value) {
                selectedConfigName = null;
                selectedConfigDetail = null;
                return;
            }
            requestConfigDetails(value, 'runtime');
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

    if (resetConfigBtn) {
        resetConfigBtn.addEventListener('click', () => {
            resetConfigForm();
        });
    }

    logSwitch.addEventListener('change', () => {
        socket.emit('toggle_logs', { enabled: logSwitch.checked });
        if(logSwitch.checked) {
            const initialMessage = logContainer.querySelector('.text-muted');
            if (initialMessage) {
               initialMessage.textContent = '正在等待日志...';
            }
        }
    });

    clearLogBtn.addEventListener('click', () => {
        logContainer.innerHTML = '<p class="text-muted">日志已清空。</p>';
    });

    // --- Cron 表达式预览 ---
    let debounceTimer;
    if (configInputs.cron_expr) {
        configInputs.cron_expr.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            debounceTimer = setTimeout(() => {
                const cronExpr = configInputs.cron_expr.value;
                if (cronExpr) {
                    fetch('/api/preview_cron', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ cron_expr: cronExpr })
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.error) {
                            cronPreviewEl.innerHTML = `<span class="text-danger">错误: ${data.error}</span>`;
                        } else {
                            let html = '<strong>接下来5次运行时间:</strong><ul>';
                            data.forEach(ts => {
                                html += `<li>${new Date(ts).toLocaleString()}</li>`;
                            });
                            html += '</ul>';
                            cronPreviewEl.innerHTML = html;
                        }
                    })
                    .catch(() => {
                        cronPreviewEl.innerHTML = `<span class="text-danger">请求预览失败</span>`;
                    });
                } else if (cronPreviewEl) {
                    cronPreviewEl.innerHTML = '';
                }
            }, 500);
        });
    }

    document.querySelectorAll('.cron-preset').forEach(button => {
        button.addEventListener('click', (e) => {
            e.preventDefault();
            if (!configInputs.cron_expr) return;
            configInputs.cron_expr.value = e.target.dataset.cron;
            configInputs.cron_expr.dispatchEvent(new Event('input'));
        });
    });

    // --- 初始加载 ---
});
