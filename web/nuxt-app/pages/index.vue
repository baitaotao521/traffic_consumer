<script setup lang="ts">
import Chart from 'chart.js/auto';
import UrlTable from '~/components/UrlTable.vue';
import { useNuxtApp, useRuntimeConfig } from '#app';

type UrlRequestMap = Record<string, { headers?: Record<string, string>; body?: string }>;

const { $socket } = useNuxtApp();

const configs = ref<string[]>([]);
const runtimeSelection = ref<string>('');
const editorSelection = ref<string>('');
const multiSelections = ref<string[]>([]);

const form = reactive({
  name: '',
  url_strategy: '',
  threads: null as number | null,
  limit_speed: null as number | null,
  traffic_limit: null as number | null,
  duration: null as number | null,
  count: null as number | null,
  cron_expr: '',
  interval: null as number | null,
  auto_remove_failed_url: false
});

const urlRows = ref([{ url: '', headers: '', body: '' }]);
const runtimeConfigDetail = ref<any>(null);
const editorActiveConfig = ref<string | null>(null);

const logs = ref<{ message: string; color?: string; ts: string }[]>([]);
const alerts = ref<{ level: 'info' | 'warning' | 'error'; message: string }[]>([]);
const history = ref<any[]>([]);
const schedulerHistory = ref<any[]>([]);
const multiJobs = ref<any[]>([]);
const menuItems = [
  { label: '运行', icon: 'i-heroicons-play', to: '#runtime' },
  { label: '状态', icon: 'i-heroicons-chart-bar', to: '#status' },
  { label: '日志', icon: 'i-heroicons-document-text', to: '#logs' },
  { label: '历史', icon: 'i-heroicons-archive-box', to: '#history' },
  { label: '调度', icon: 'i-heroicons-calendar-days', to: '#scheduler' }
];

const status = reactive({
  running: false,
  speed: '0 B/s',
  total_bytes: '0',
  download_count: '0',
  current_config: '未选择',
  thread_status: [] as any[],
  thread_count: {} as any
});

const schedulerStatus = reactive({
  job_details: null as string | null,
  next_run_time: null as string | null,
  countdown: '无'
});

const cronPreview = ref<string[]>([]);
const logEnabled = ref(true);
const showEditorModal = ref(false);

const speedChartRef = ref<HTMLCanvasElement | null>(null);
const urlChartRef = ref<HTMLCanvasElement | null>(null);
const threadChartRef = ref<HTMLCanvasElement | null>(null);
let speedChart: Chart | null = null;
let urlChart: Chart | null = null;
let threadChart: Chart | null = null;
let countdownTimer: any = null;

const loading = reactive({
  saving: false,
  starting: false
});

function pushAlert(payload: { message: string; level?: 'info' | 'warning' | 'error' }) {
  alerts.value.unshift({ message: payload.message, level: payload.level || 'warning' });
  if (alerts.value.length > 5) {
    alerts.value.pop();
  }
}

function normalizeConfigPayload(raw: any, name: string | null = null) {
  const payload: any = {
    url_strategy: raw.url_strategy || null,
    threads: raw.threads ?? null,
    limit_speed: raw.limit_speed ?? null,
    traffic_limit: raw.traffic_limit ?? null,
    duration: raw.duration ?? null,
    count: raw.count ?? null,
    cron_expr: raw.cron_expr || null,
    interval: raw.interval ?? null,
    config_name: name || raw.config_name || null,
    auto_remove_failed_url: Boolean(raw.auto_remove_failed_url)
  };

  payload.name = name || raw.name || '';

  const urls = Array.isArray(raw.urls) ? raw.urls : [];
  payload.urls = urls
    .map((item: any) => {
      if (typeof item === 'string') return item.trim();
      if (item && typeof item === 'object' && typeof item.url === 'string') return item.url.trim();
      return '';
    })
    .filter((url: string) => url !== '');

  const rawRequests = raw.url_requests && typeof raw.url_requests === 'object' ? raw.url_requests : {};
  const normalizedRequests: UrlRequestMap = {};
  payload.urls.forEach((url: string) => {
    const options = rawRequests[url];
    if (!options || typeof options !== 'object') return;
    const normalizedOption: any = {};
    if (options.headers && typeof options.headers === 'object' && !Array.isArray(options.headers)) {
      const headers = options.headers as Record<string, string>;
      if (Object.keys(headers).length) normalizedOption.headers = headers;
    }
    if (options.body !== undefined && options.body !== null && String(options.body).trim() !== '') {
      normalizedOption.body = typeof options.body === 'string' ? options.body : String(options.body);
    }
    if (Object.keys(normalizedOption).length) {
      normalizedRequests[url] = normalizedOption;
    }
  });
  payload.url_requests = normalizedRequests;

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

function buildPayloadFromForm() {
  const urls: string[] = [];
  const urlRequests: UrlRequestMap = {};
  let error = '';

  urlRows.value.forEach((row, idx) => {
    const url = (row.url || '').trim();
    if (!url) return;
    urls.push(url);

    const options: any = {};
    if (row.headers && row.headers.trim()) {
      try {
        const parsed = JSON.parse(row.headers);
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
          throw new Error('请求头需要是 JSON 对象');
        }
        options.headers = parsed;
      } catch (err: any) {
        error = `第 ${idx + 1} 行请求头格式错误：${err.message}`;
      }
    }

    if (row.body && row.body.trim() !== '') {
      options.body = row.body;
    }

    if (Object.keys(options).length) {
      urlRequests[url] = options;
    }
  });

  if (error) {
    pushAlert({ message: error, level: 'error' });
    return null;
  }

  const payload = normalizeConfigPayload(
    {
      ...form,
      urls,
      url_requests: urlRequests
    },
    form.name || editorActiveConfig.value
  );

  payload.name = payload.name || editorActiveConfig.value || '';
  return payload;
}

function resetForm(keepSelection = false) {
  form.name = '';
  form.url_strategy = '';
  form.threads = null;
  form.limit_speed = null;
  form.traffic_limit = null;
  form.duration = null;
  form.count = null;
  form.cron_expr = '';
  form.interval = null;
  form.auto_remove_failed_url = false;
  urlRows.value = [{ url: '', headers: '', body: '' }];
  cronPreview.value = [];
  if (!keepSelection) {
    editorSelection.value = '';
    editorActiveConfig.value = null;
  }
}

function populateForm(configName: string | null, config: any) {
  const normalized = normalizeConfigPayload(config, configName);
  form.name = normalized.name || configName || '';
  form.url_strategy = normalized.url_strategy || '';
  form.threads = normalized.threads;
  form.limit_speed = normalized.limit_speed;
  form.traffic_limit = normalized.traffic_limit;
  form.duration = normalized.duration;
  form.count = normalized.count;
  form.cron_expr = normalized.cron_expr || '';
  form.interval = normalized.interval;
  form.auto_remove_failed_url = Boolean(normalized.auto_remove_failed_url);

  const rows = (normalized.urls || []).map((url: string) => {
    const options = normalized.url_requests?.[url] || {};
    const headers = options.headers ? JSON.stringify(options.headers, null, 2) : '';
    const body = options.body || '';
    return { url, headers, body };
  });
  urlRows.value = rows.length ? rows : [{ url: '', headers: '', body: '' }];
}

function requestConfigs() {
  $socket.emit('get_configs');
}

function requestConfigDetails(name: string, target: 'runtime' | 'editor') {
  $socket.emit('get_config_details', { name, target });
}

function saveConfig() {
  const payload = buildPayloadFromForm();
  if (!payload) return;
  if (!payload.name) {
    pushAlert({ message: '请填写配置名称后再保存。', level: 'warning' });
    return;
  }
  loading.saving = true;
  $socket.emit('save_config', { name: payload.name, data: payload });
  editorActiveConfig.value = payload.name;
}

function selectRuntime(name: string) {
  runtimeSelection.value = name;
  runtimeConfigDetail.value = null;
  if (name) {
    requestConfigDetails(name, 'runtime');
  }
}

function selectEditor(name: string) {
  editorSelection.value = name;
  if (name) {
    requestConfigDetails(name, 'editor');
  } else {
    resetForm(true);
  }
}

function startConsumer() {
  if (!runtimeSelection.value || !runtimeConfigDetail.value) {
    pushAlert({ message: '请选择有效的运行配置后再启动。', level: 'warning' });
    return;
  }
  loading.starting = true;
  const payload = normalizeConfigPayload(runtimeConfigDetail.value, runtimeSelection.value);
  payload.config_name = runtimeSelection.value;
  payload.name = runtimeSelection.value;
  payload.urls = Array.isArray(payload.urls) ? [...payload.urls] : [];
  $socket.emit('start_consumer', payload);
}

function stopConsumer() {
  $socket.emit('stop_consumer');
}

function stopScheduler() {
  $socket.emit('stop_scheduler');
}

function startMulti() {
  if (!multiSelections.value.length) {
    pushAlert({ message: '请先选择要启动的配置。', level: 'warning' });
    return;
  }
  $socket.emit('start_multi_configs', { config_names: multiSelections.value });
}

function stopMulti() {
  $socket.emit('stop_multi_configs', {});
}

function toggleLogs() {
  $socket.emit('toggle_logs', { enabled: logEnabled.value });
  if (logEnabled.value && !logs.value.length) {
    logs.value.push({ message: '正在等待日志...', ts: new Date().toLocaleTimeString() });
  }
}

async function previewCron() {
  if (!form.cron_expr) {
    pushAlert({ message: '请先填写 Cron 表达式。', level: 'warning' });
    return;
  }
  try {
    const res = await $fetch<string[]>('/api/preview_cron', {
      method: 'POST',
      body: { cron_expr: form.cron_expr }
    });
    cronPreview.value = res;
  } catch (err: any) {
    pushAlert({ message: err?.data?.error || '预览失败' });
  }
}

function addLog(entry: { message?: string; color?: string }) {
  if (!logEnabled.value) return;
  const message = (entry.message || '').trim();
  const color = entry.color || '#f8f9fa';
  logs.value.push({
    message,
    color,
    ts: new Date().toLocaleTimeString()
  });
  if (logs.value.length > 300) {
    logs.value.shift();
  }
  nextTick(() => {
    const box = document.getElementById('log-box');
    if (box) {
      box.scrollTop = box.scrollHeight;
    }
  });
}

function updateSpeedChart(label: string, value: number) {
  if (!speedChart) return;
  const data = speedChart.data.datasets[0].data as number[];
  const labels = speedChart.data.labels as string[];
  labels.push(label);
  data.push(value);
  if (labels.length > 30) {
    labels.shift();
    data.shift();
  }
  speedChart.update('none');
}

function renderUrlUsage(stats: any) {
  if (!urlChart || !stats) return;
  const labels: string[] = [];
  const values: number[] = [];
  if (Array.isArray(stats)) {
    stats.forEach((item: any, idx: number) => {
      labels.push(item.url || `链接 ${idx + 1}`);
      values.push(Number(item.count) || 0);
    });
  } else if (typeof stats === 'object') {
    Object.entries(stats).forEach(([url, count]) => {
      labels.push(url);
      values.push(Number(count as any) || 0);
    });
  }
  urlChart.data.labels = labels.length ? labels : ['无数据'];
  urlChart.data.datasets[0].data = values.length ? values : [1];
  urlChart.update('none');
}

function renderThreadUsage(countInfo: any) {
  if (!threadChart) return;
  let active = 0;
  let idle = 0;
  let errored = 0;
  if (countInfo && typeof countInfo === 'object') {
    active = Number(countInfo.active || 0);
    idle = Number(countInfo.idle || 0);
    errored = Number(countInfo.errored || 0);
  }
  threadChart.data.datasets[0].data = [active, idle, errored];
  threadChart.update('none');
}

function handleConfigsList(data: any) {
  configs.value = Array.isArray(data.configs) ? data.configs : [];
  // 保持选择
  if (runtimeSelection.value && !configs.value.includes(runtimeSelection.value)) {
    runtimeSelection.value = '';
    runtimeConfigDetail.value = null;
  }
  if (editorSelection.value && !configs.value.includes(editorSelection.value)) {
    editorSelection.value = '';
    editorActiveConfig.value = null;
  }
  if (!configs.value.length) {
    multiSelections.value = [];
  }
}

function handleConfigDetails(data: any) {
  const target = data.target || 'runtime';
  const name = data.name;
  const config = data.config || {};
  if (target === 'editor') {
    editorSelection.value = name;
    editorActiveConfig.value = name;
    populateForm(name, config);
  } else {
    runtimeSelection.value = name;
    runtimeConfigDetail.value = normalizeConfigPayload(config, name);
  }
}

function handleStatusUpdate(data: any) {
  status.running = Boolean(data.running);
  status.speed = data.speed || '0 B/s';
  status.total_bytes = data.total_bytes || '0';
  status.download_count = data.download_count || '0';
  status.current_config = data.config || runtimeSelection.value || '未命名配置';
  status.thread_status = Array.isArray(data.thread_status) ? data.thread_status : [];
  status.thread_count = data.thread_count || {};
  renderThreadUsage(status.thread_count);
  renderUrlUsage(data.url_usage_stats);

  const speedValue = data.speed ? data.speed.match(/(\d+\.?\d*)\s*MB\/s/i) : null;
  const speedMB = speedValue ? parseFloat(speedValue[1]) : 0;
  updateSpeedChart(new Date().toLocaleTimeString(), speedMB);

  loading.starting = false;
  loading.saving = false;
}

function handleHistoryUpdate(record: any) {
  history.value.unshift({
    ...record,
    timestamp: record.timestamp || new Date().toISOString()
  });
  if (history.value.length > 50) history.value.pop();
}

function handleSchedulerStatus(data: any) {
  schedulerStatus.job_details = data.job_details || null;
  schedulerStatus.next_run_time = data.next_run_time || null;
  schedulerHistory.value = Array.isArray(data.history) ? data.history : [];
  if (Array.isArray(data.history)) {
    history.value = data.history.map((item: any) => ({
      ...item,
      timestamp: item.timestamp || item.end_time || item.start_time
    }));
  }
  multiJobs.value = Array.isArray(data.multi_jobs) ? data.multi_jobs : [];
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
  if (schedulerStatus.next_run_time) {
    const target = new Date(schedulerStatus.next_run_time);
    const updateCountdown = () => {
      const diff = target.getTime() - Date.now();
      if (diff <= 0) {
        schedulerStatus.countdown = '运行中...';
        clearInterval(countdownTimer);
        return;
      }
      const h = Math.floor(diff / 3600000).toString().padStart(2, '0');
      const m = Math.floor((diff % 3600000) / 60000).toString().padStart(2, '0');
      const s = Math.floor((diff % 60000) / 1000).toString().padStart(2, '0');
      schedulerStatus.countdown = `${h}:${m}:${s}`;
    };
    updateCountdown();
    countdownTimer = setInterval(updateCountdown, 1000);
  } else {
    schedulerStatus.countdown = '无';
  }
}

function initCharts() {
  if (speedChartRef.value) {
    speedChart = new Chart(speedChartRef.value.getContext('2d')!, {
      type: 'line',
      data: {
        labels: [],
        datasets: [
          {
            label: '速度 (MB/s)',
            data: [],
            borderColor: 'rgba(255, 105, 180, 0.9)',
            backgroundColor: 'rgba(255, 105, 180, 0.25)',
            fill: true,
            tension: 0.35
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { display: false } },
        scales: {
          x: { display: false },
          y: { beginAtZero: true }
        }
      }
    });
  }

  if (urlChartRef.value) {
    urlChart = new Chart(urlChartRef.value.getContext('2d')!, {
      type: 'doughnut',
      data: {
        labels: ['无数据'],
        datasets: [
          {
            data: [1],
            backgroundColor: ['rgba(255, 105, 180, 0.85)', 'rgba(255, 182, 193, 0.75)', 'rgba(255, 228, 240, 0.8)'],
            borderWidth: 0
          }
        ]
      },
      options: {
        cutout: '58%',
        plugins: {
          legend: {
            position: 'bottom',
            labels: { boxWidth: 14, boxHeight: 14, padding: 12 }
          }
        }
      }
    });
  }

  if (threadChartRef.value) {
    threadChart = new Chart(threadChartRef.value.getContext('2d')!, {
      type: 'doughnut',
      data: {
        labels: ['活跃', '空闲', '失效'],
        datasets: [
          {
            data: [0, 0, 0],
            backgroundColor: [
              'rgba(255, 105, 180, 0.85)',
              'rgba(255, 182, 193, 0.85)',
              'rgba(220, 53, 69, 0.75)'
            ],
            borderWidth: 0
          }
        ]
      },
      options: {
        cutout: '58%',
        plugins: {
          legend: {
            position: 'bottom',
            labels: { boxWidth: 14, boxHeight: 14, padding: 12 }
          }
        }
      }
    });
  }
}

onMounted(() => {
  initCharts();
  requestConfigs();
  $socket.on('configs_list', handleConfigsList);
  $socket.on('config_details', handleConfigDetails);
  $socket.on('status_update', handleStatusUpdate);
  $socket.on('history_update', handleHistoryUpdate);
  $socket.on('log_message', (data: any) => addLog(data || {}));
  $socket.on('invalid_url', (data: any) => pushAlert({ message: data?.message || '链接无效', level: 'warning' }));
  $socket.on('scheduler_status_update', handleSchedulerStatus);
  $socket.on('multi_scheduler_feedback', (payload: any) =>
    pushAlert({ message: payload?.message || '批量任务反馈', level: payload?.level === 'error' ? 'error' : 'info' })
  );
});

onBeforeUnmount(() => {
  if (countdownTimer) clearInterval(countdownTimer);
  if ($socket?.off) {
    $socket.off('configs_list', handleConfigsList);
    $socket.off('config_details', handleConfigDetails);
    $socket.off('status_update', handleStatusUpdate);
    $socket.off('history_update', handleHistoryUpdate);
    $socket.off('log_message');
    $socket.off('invalid_url');
    $socket.off('scheduler_status_update', handleSchedulerStatus);
    $socket.off('multi_scheduler_feedback');
  }
});

function scrollToSection(target: string) {
  const el = document.querySelector(target);
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }
}
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="py-3 px-4 flex items-center justify-between flex-wrap gap-3 w-full">
        <div class="flex items-center gap-3 flex-wrap">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-bolt" class="text-pink" size="24" />
            <span class="font-bold text-lg">Traffic Consumer</span>
          </div>
          <UHorizontalNavigation
            :links="menuItems"
            color="primary"
            @select="(item) => scrollToSection(item.to as string)"
          />
        </div>
        <div class="flex items-center gap-2 flex-wrap">
          <UBadge :color="status.running ? 'green' : 'gray'" variant="soft">
            {{ status.running ? '运行中' : '已停止' }}
          </UBadge>
          <UButton color="gray" variant="outline" size="sm" icon="i-heroicons-arrow-path" @click="requestConfigs">刷新</UButton>
        </div>
      </div>
    </header>

    <div class="py-4 px-4 flex-grow w-full">
      <div class="layout-grid">
        <div class="space-y-4">
          <UCard id="runtime">
            <template #header>
              <div class="panel-header">
                <div class="section-title">运行配置</div>
                <div class="flex gap-2 flex-wrap">
                  <UButton color="primary" variant="outline" size="sm" :loading="loading.starting" icon="i-heroicons-play" @click="startConsumer">启动</UButton>
                  <UButton color="gray" variant="outline" size="sm" icon="i-heroicons-stop" @click="stopConsumer">停止</UButton>
                  <UButton color="pink" variant="solid" size="sm" icon="i-heroicons-adjustments-horizontal" @click="showEditorModal = true">配置编辑</UButton>
                </div>
              </div>
            </template>
            <div class="space-y-3">
              <div>
                <p class="section-label">选择配置</p>
                <USelect v-model="runtimeSelection" :options="['', ...configs]" option-attribute="label" @update:model-value="selectRuntime(runtimeSelection)">
                  <template #option="{ option }">
                    <span>{{ option || '请选择' }}</span>
                  </template>
                </USelect>
              </div>
              <div class="flex gap-2 flex-wrap">
                <UButton color="amber" variant="outline" size="sm" icon="i-heroicons-calendar-x-mark" @click="stopScheduler">停止调度</UButton>
                <UButton color="primary" variant="outline" size="sm" icon="i-heroicons-queue-list" @click="startMulti">启动多任务</UButton>
                <UButton color="gray" variant="outline" size="sm" icon="i-heroicons-pause" @click="stopMulti">停止多任务</UButton>
              </div>
              <div>
                <p class="section-label">多任务选择</p>
                <USelectMenu v-model="multiSelections" multiple :options="configs" placeholder="选择多个配置" />
              </div>
            </div>
          </UCard>

          <!-- 配置编辑移至弹窗 -->
        </div>

        <div class="space-y-4">
          <UCard id="status">
            <template #header>
              <div class="panel-header">
                <div class="section-title">实时状态</div>
                <div class="flex items-center gap-2 flex-wrap">
                  <UBadge :color="status.running ? 'green' : 'gray'" variant="solid">{{ status.running ? '运行中' : '已停止' }}</UBadge>
                  <span class="pill">{{ status.current_config }}</span>
                </div>
              </div>
            </template>
            <div class="grid-stretch" style="grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));">
              <div>
                <p class="section-label">速度</p>
                <p class="font-semibold">{{ status.speed }}</p>
              </div>
              <div>
                <p class="section-label">总流量</p>
                <p class="font-semibold">{{ status.total_bytes }}</p>
              </div>
              <div>
                <p class="section-label">下载次数</p>
                <p class="font-semibold">{{ status.download_count }}</p>
              </div>
              <div>
                <p class="section-label">下次运行</p>
                <p class="font-semibold">{{ schedulerStatus.next_run_time || '无' }}</p>
              </div>
            </div>
            <div class="mt-3" style="height: 220px;">
              <canvas ref="speedChartRef" />
            </div>
          </UCard>

          <div class="grid-two">
            <UCard>
              <template #header>
                <div class="section-title">URL 使用分布</div>
              </template>
              <div style="height: 180px;">
                <canvas ref="urlChartRef" />
              </div>
            </UCard>

            <UCard>
              <template #header>
                <div class="section-title">线程状态</div>
              </template>
              <div style="height: 180px;">
                <canvas ref="threadChartRef" />
              </div>
            </UCard>
          </div>

          <UCard id="logs">
            <template #header>
              <div class="panel-header">
                <div class="section-title">日志</div>
                <div class="flex items-center gap-2">
                  <UToggle v-model="logEnabled" @change="toggleLogs" />
                  <span>接收日志</span>
                </div>
              </div>
            </template>
            <div id="log-box" class="log-container">
              <div
                v-for="(item, idx) in logs"
                :key="idx"
                class="log-entry"
                :style="{ color: item.color || '#f8f9fa' }"
              >
                <span class="timestamp">[{{ item.ts }}]</span>
                <span class="message">{{ item.message }}</span>
              </div>
              <div v-if="!logs.length" class="text-muted">日志将显示在这里...</div>
            </div>
            <div class="mt-2 flex gap-2">
              <UButton color="red" variant="outline" size="sm" icon="i-heroicons-trash" @click="logs = []">清空</UButton>
            </div>
          </UCard>

          <UCard id="history">
            <template #header>
              <div class="section-title">下载历史</div>
            </template>
            <div class="table-wrapper">
              <table class="table-simple">
                <thead>
                  <tr>
                    <th>时间</th>
                    <th>配置</th>
                    <th>结果</th>
                    <th>流量</th>
                    <th>次数</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="!history.length">
                    <td colspan="5" class="text-muted">暂无历史记录</td>
                  </tr>
                  <tr v-for="(item, idx) in history" :key="idx">
                    <td>{{ new Date(item.timestamp).toLocaleString() }}</td>
                    <td>{{ item.config_name || item.config || '未命名配置' }}</td>
                    <td>{{ item.result }}</td>
                    <td>{{ item.bytes_consumed }}</td>
                    <td>{{ item.download_count || 'N/A' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </UCard>

          <UCard id="scheduler">
            <template #header>
              <div class="section-title">调度状态</div>
            </template>
            <div class="grid-stretch">
              <div>
                <p class="section-label">下次运行</p>
                <p class="font-semibold">{{ schedulerStatus.next_run_time || '无' }}</p>
              </div>
              <div>
                <p class="section-label">倒计时</p>
                <p class="font-semibold">{{ schedulerStatus.countdown }}</p>
              </div>
              <div>
                <p class="section-label">任务</p>
                <p class="font-semibold">{{ schedulerStatus.job_details || '无' }}</p>
              </div>
            </div>
            <hr class="my-3" />
            <div class="section-title mb-2">多任务</div>
            <div class="table-wrapper">
              <table class="table-simple">
                <thead>
                  <tr>
                    <th>配置</th>
                    <th>状态</th>
                    <th>下次运行</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="!multiJobs.length">
                    <td colspan="3" class="text-muted">暂无计划任务</td>
                  </tr>
                  <tr v-for="(job, idx) in multiJobs" :key="idx">
                    <td>{{ job.config_name || job.name || '未命名' }}</td>
                    <td>{{ job.status || '待定' }}</td>
                    <td>{{ job.next_run_time || '无' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </UCard>
        </div>
      </div>

      <div class="mt-3" v-if="alerts.length">
        <UAlert
          v-for="(item, idx) in alerts"
          :key="idx"
          class="mb-2"
          :color="item.level === 'error' ? 'red' : item.level === 'warning' ? 'amber' : 'blue'"
          variant="soft"
        >
          {{ item.message }}
        </UAlert>
      </div>
    </div>

    <Transition name="fade">
      <div v-if="showEditorModal" class="modal-backdrop" @click.self="showEditorModal = false">
        <div class="modal-card">
          <div class="panel-header">
            <div class="section-title">配置编辑</div>
            <div class="flex gap-2 flex-wrap">
              <UButton color="gray" variant="outline" size="sm" icon="i-heroicons-eraser" @click="resetForm(true)">清空</UButton>
              <UButton color="green" size="sm" :loading="loading.saving" icon="i-heroicons-check-circle" @click="saveConfig">保存</UButton>
              <UButton color="gray" variant="ghost" size="sm" icon="i-heroicons-x-mark" @click="showEditorModal = false">关闭</UButton>
            </div>
          </div>

          <div class="modal-body">
            <div class="modal-left space-y-3">
              <div>
                <p class="section-label">选择配置</p>
                <USelect v-model="editorSelection" :options="['', ...configs]" placeholder="新建配置" @update:model-value="selectEditor(editorSelection)">
                  <template #option="{ option }">
                    <span>{{ option || '新建配置' }}</span>
                  </template>
                </USelect>
              </div>

              <div class="grid-stretch">
                <div class="col-span-2">
                  <p class="section-label">配置名称</p>
                  <UInput v-model="form.name" size="md" placeholder="例如：pink_task" />
                </div>
                <div>
                  <p class="section-label">线程数</p>
                  <UInput v-model.number="form.threads" type="number" size="md" placeholder="默认：4" />
                </div>
                <div>
                  <p class="section-label">URL 策略</p>
                  <USelect v-model="form.url_strategy" :options="[{ label: '默认', value: '' }, { label: '随机均衡', value: 'random' }, { label: '轮询顺序', value: 'round_robin' }]" option-attribute="label" value-attribute="value" />
                </div>
                <div>
                  <p class="section-label">限速 (MB/s)</p>
                  <UInput v-model.number="form.limit_speed" type="number" size="md" placeholder="0 表示不限速" />
                </div>
                <div>
                  <p class="section-label">总流量 (MB)</p>
                  <UInput v-model.number="form.traffic_limit" type="number" size="md" placeholder="默认：无限制" />
                </div>
                <div>
                  <p class="section-label">时长 (秒)</p>
                  <UInput v-model.number="form.duration" type="number" size="md" placeholder="默认：无限制" />
                </div>
                <div>
                  <p class="section-label">下载次数</p>
                  <UInput v-model.number="form.count" type="number" size="md" placeholder="默认：无限制" />
                </div>
                <div>
                  <p class="section-label">间隔 (分钟)</p>
                  <UInput v-model.number="form.interval" type="number" size="md" placeholder="例如：15" />
                </div>
                <div>
                  <p class="section-label">Cron 表达式</p>
                  <div class="flex gap-2">
                    <UInput v-model="form.cron_expr" placeholder="例如：*/5 * * * *" class="flex-1" />
                    <UButton color="primary" variant="outline" size="sm" @click="previewCron">预览</UButton>
                  </div>
                  <p class="section-label" v-if="cronPreview.length">下次运行：{{ cronPreview.join('，') }}</p>
                </div>
                <div class="col-span-2 flex items-center gap-2">
                  <UToggle v-model="form.auto_remove_failed_url" />
                  <span>失败链接自动移除</span>
                </div>
              </div>
            </div>

            <div class="modal-right">
              <UrlTable v-model="urlRows" />
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.url-grid-wrapper {
  display: grid;
  gap: 12px;
}

.url-row-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr auto;
  gap: 12px;
  align-items: start;
  padding: 12px;
  border: 1px solid #e9ecef;
  border-radius: 10px;
  background: #fff;
}

.url-cell.actions {
  display: flex;
  align-items: center;
  justify-content: center;
}

@media (max-width: 992px) {
  .url-row-grid {
    grid-template-columns: 1fr;
  }
  .url-cell.actions {
    justify-content: flex-start;
  }
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 24px;
}

.modal-card {
  width: 96vw;
  max-width: none;
  max-height: 92vh;
  min-height: 65vh;
  overflow: hidden;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.12);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-body {
  display: grid;
  grid-template-columns: minmax(520px, 1.25fr) minmax(480px, 1fr);
  gap: 20px;
  min-height: 0;
  overflow: auto;
}

.modal-left,
.modal-right {
  min-width: 0;
}

.modal-right {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 992px) {
  .modal-body {
    grid-template-columns: 1fr;
  }
  .modal-card {
    width: 96vw;
    max-height: 96vh;
  }
}
</style>
