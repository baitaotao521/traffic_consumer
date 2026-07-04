<script setup lang="ts">
import { computed } from 'vue'

import ConfirmActionDialog from '@/components/dashboard/ConfirmActionDialog.vue'
import ConfigSheet from '@/components/dashboard/ConfigSheet.vue'
import KpiGrid from '@/components/dashboard/KpiGrid.vue'
import LogHistoryPanel from '@/components/dashboard/LogHistoryPanel.vue'
import PlanDetailDialog from '@/components/dashboard/PlanDetailDialog.vue'
import PlanPanel from '@/components/dashboard/PlanPanel.vue'
import SpeedPanel from '@/components/dashboard/SpeedPanel.vue'
import ThreadPanel from '@/components/dashboard/ThreadPanel.vue'
import TopBar from '@/components/dashboard/TopBar.vue'
import UrlUsagePanel from '@/components/dashboard/UrlUsagePanel.vue'
import { Toaster } from '@/components/ui/sonner'
import { useDashboard } from '@/composables/useDashboard'

const dashboard = useDashboard()

const isStopped = computed(() => dashboard.status.value.state === 'stopped')
const isRunning = computed(() => dashboard.status.value.state === 'running')
const hasScheduler = computed(() => {
  return Boolean(dashboard.scheduler.value.job_details) || dashboard.plans.value.some((plan) => plan.scheduler_running)
})
</script>

<template>
  <main>
    <TopBar
      v-model:selected-config-name="dashboard.selectedConfigName.value"
      :configs="dashboard.configs.value"
      :has-scheduler="hasScheduler"
      :is-running="isRunning"
      :is-stopped="isStopped"
      :status-label="dashboard.statusLabel.value"
      :status-variant="dashboard.statusVariant.value"
      @create-config="dashboard.openCreateConfig"
      @delete-config="dashboard.requestDeleteSelectedConfig"
      @open-editor="dashboard.openEditor"
      @start="dashboard.runAction(dashboard.startSelected)"
      @start-temp="dashboard.runAction(dashboard.startSelectedTempTest)"
      @stop="dashboard.runAction(dashboard.stopSelected)"
      @stop-scheduler="dashboard.runAction(dashboard.stopSelectedScheduler)"
    />

    <section class="mx-auto flex max-w-[1600px] flex-col gap-6 px-6 py-6">
      <KpiGrid
        :current-config="dashboard.currentConfig.value"
        :download-count="dashboard.status.value.download_count || 0"
        :speed="dashboard.status.value.speed || '0 B/s'"
        :total-bytes="dashboard.status.value.total_bytes || '0 B'"
      />

      <div class="grid items-start gap-6 xl:grid-cols-12">
        <div class="grid gap-6 xl:col-span-8">
          <SpeedPanel
            :labels="dashboard.speedLabels.value"
            :values="dashboard.speedValues.value"
            @refresh="dashboard.runAction(dashboard.refreshDashboard)"
          />
          <ThreadPanel :thread-counts="dashboard.threadCounts.value" :thread-rows="dashboard.threadRows.value" />
        </div>

        <div class="grid auto-rows-min content-start gap-6 xl:col-span-4">
          <UrlUsagePanel
            :labels="dashboard.urlChartLabels.value"
            :stats="dashboard.urlStats.value"
            :values="dashboard.urlChartValues.value"
          />
          <PlanPanel
            :format-date-time="dashboard.formatDateTime"
            :format-plan-schedule="dashboard.formatPlanSchedule"
            :job-details="dashboard.scheduler.value.job_details || ''"
            :plan-state="dashboard.planState"
            :plans="dashboard.plans.value"
            @open-plan-detail="dashboard.runAction(() => dashboard.openPlanDetail($event))"
            @stop-plan="dashboard.requestStopNamedPlan($event)"
          />
        </div>
      </div>

      <LogHistoryPanel
        v-model:log-enabled="dashboard.logEnabled.value"
        :format-date-time="dashboard.formatDateTime"
        :log-level-class="dashboard.logLevelClass"
        :logs="dashboard.logs.value"
        :plans="dashboard.plans.value"
        @clear-logs="dashboard.clearLogs"
      />
    </section>

    <Toaster
      position="bottom-right"
      :expand="false"
      :rich-colors="true"
      :visible-toasts="2"
      close-button
    />

    <ConfirmActionDialog
      v-model:open="dashboard.confirmDialogOpen.value"
      :confirm-label="dashboard.confirmDialogActionLabel.value"
      :description="dashboard.confirmDialogDescription.value"
      :title="dashboard.confirmDialogTitle.value"
      :variant="dashboard.confirmDialogVariant.value"
      @cancel="dashboard.cancelPendingAction"
      @confirm="dashboard.confirmPendingAction"
    />

    <ConfigSheet
      v-model:open="dashboard.editorOpen.value"
      :cron-preview="dashboard.cronPreview.value"
      :form="dashboard.form"
      :format-date-time="dashboard.formatDateTime"
      @reset="dashboard.resetEditor"
      @save="dashboard.runAction(dashboard.saveEditorConfig)"
    />

    <PlanDetailDialog
      v-model:open="dashboard.planDialogOpen.value"
      :format-date-time="dashboard.formatDateTime"
      :plan-detail="dashboard.selectedPlanDetail.value"
    />
  </main>
</template>
