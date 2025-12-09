<script setup lang="ts">
type UrlRow = {
  url: string;
  headers: string;
  body: string;
};

const props = defineProps<{
  modelValue: UrlRow[];
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: UrlRow[]): void;
}>();

const rows = ref<UrlRow[]>(props.modelValue?.length ? [...props.modelValue] : [{ url: '', headers: '', body: '' }]);
const showHeadersModal = ref(false);
const showBodyModal = ref(false);
const showPasteModal = ref(false);
const batchHeadersText = ref('');
const batchBodyText = ref('');
const pasteText = ref('');

watch(
  () => props.modelValue,
  (val) => {
    rows.value = val?.length ? [...val] : [{ url: '', headers: '', body: '' }];
  },
  { deep: true }
);

function sync() {
  emit('update:modelValue', rows.value);
}

function addRow() {
  rows.value.push({ url: '', headers: '', body: '' });
  sync();
}

function removeRow(index: number) {
  rows.value.splice(index, 1);
  if (!rows.value.length) {
    rows.value.push({ url: '', headers: '', body: '' });
  }
  sync();
}

function applyHeaders() {
  let value = '';
  if (batchHeadersText.value.trim()) {
    try {
      const parsed = JSON.parse(batchHeadersText.value);
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('请求头必须是 JSON 对象');
      }
      value = JSON.stringify(parsed, null, 2);
    } catch (err: any) {
      window.alert(`请求头格式有误：${err.message}`);
      return;
    }
  }
  rows.value = rows.value.map((row) => ({ ...row, headers: value }));
  sync();
  showHeadersModal.value = false;
}

function applyBody() {
  rows.value = rows.value.map((row) => ({ ...row, body: batchBodyText.value || '' }));
  sync();
  showBodyModal.value = false;
}

function appendFromPaste() {
  const lines = pasteText.value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  if (!lines.length) {
    showPasteModal.value = false;
    return;
  }
  const newRows = lines.map((url) => ({ url, headers: '', body: '' }));
  rows.value = rows.value.filter((row) => row.url || row.headers || row.body);
  rows.value.push(...newRows);
  if (!rows.value.length) {
    rows.value.push({ url: '', headers: '', body: '' });
  }
  sync();
  showPasteModal.value = false;
  pasteText.value = '';
}
</script>

<template>
  <div class="url-actions mb-3">
    <div class="section-title mb-0">下载链接</div>
    <div class="flex gap-2 flex-wrap">
      <UButton icon="i-heroicons-plus-circle" color="primary" size="sm" variant="outline" @click="addRow">添加</UButton>
      <UButton icon="i-heroicons-clipboard" color="gray" size="sm" variant="outline" @click="showPasteModal = true">批量粘贴</UButton>
      <UButton icon="i-heroicons-code-bracket-square" color="primary" size="sm" variant="outline" @click="showHeadersModal = true">批量请求头</UButton>
      <UButton icon="i-heroicons-document-text" color="primary" size="sm" variant="outline" @click="showBodyModal = true">批量请求体</UButton>
    </div>
  </div>

  <div class="url-grid-wrapper">
    <div
      v-for="(row, index) in rows"
      :key="index"
      class="url-row-grid"
    >
      <div class="url-cell">
        <p class="section-label mb-1">下载链接</p>
        <UInput v-model="row.url" size="sm" placeholder="https://example.com/file" @update:model-value="sync" />
      </div>
      <div class="url-cell">
        <p class="section-label mb-1">请求头 (JSON)</p>
        <UTextarea
          v-model="row.headers"
          size="sm"
          autoresize
          :rows="2"
          placeholder='{"Authorization": "Bearer xxx"}'
          @update:model-value="sync"
        />
      </div>
      <div class="url-cell">
        <p class="section-label mb-1">请求体</p>
        <UTextarea
          v-model="row.body"
          size="sm"
          autoresize
          :rows="2"
          placeholder="可选：请求体内容"
          @update:model-value="sync"
        />
      </div>
      <div class="url-cell actions">
        <UButton
          color="red"
          variant="ghost"
          size="sm"
          icon="i-heroicons-trash"
          aria-label="删除"
          @click="removeRow(index)"
        />
      </div>
    </div>
  </div>
  <p class="section-label mt-1">为每个链接单独设置请求头/体，留空则使用默认 GET 请求。</p>

  <UModal v-model="showHeadersModal">
    <UCard>
      <template #header>
        <div class="section-title">批量请求头</div>
      </template>
      <p class="section-label">请输入 JSON 对象，留空则清除</p>
      <UTextarea v-model="batchHeadersText" :rows="6" placeholder='{"Authorization":"Bearer xxx"}' />
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="gray" variant="outline" @click="showHeadersModal = false">取消</UButton>
          <UButton color="primary" @click="applyHeaders">应用</UButton>
        </div>
      </template>
    </UCard>
  </UModal>

  <UModal v-model="showBodyModal">
    <UCard>
      <template #header>
        <div class="section-title">批量请求体</div>
      </template>
      <p class="section-label">请输入要应用的请求体，留空则清除</p>
      <UTextarea v-model="batchBodyText" :rows="6" placeholder="示例：{'key':'value'}" />
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="gray" variant="outline" @click="showBodyModal = false">取消</UButton>
          <UButton color="primary" @click="applyBody">应用</UButton>
        </div>
      </template>
    </UCard>
  </UModal>

  <UModal v-model="showPasteModal">
    <UCard>
      <template #header>
        <div class="section-title">批量粘贴链接</div>
      </template>
      <p class="section-label">每行一个链接，将追加到列表</p>
      <UTextarea v-model="pasteText" :rows="8" placeholder="https://example.com/file1&#10;https://example.com/file2" />
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="gray" variant="outline" @click="showPasteModal = false">取消</UButton>
          <UButton color="primary" @click="appendFromPaste">追加</UButton>
        </div>
      </template>
    </UCard>
  </UModal>
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
</style>
