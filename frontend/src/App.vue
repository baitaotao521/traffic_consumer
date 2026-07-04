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
import { Skeleton } from '@/components/ui/skeleton'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useDashboard } from '@/composables/useDashboard'

const dashboard = useDashboard()

const isStopped = computed(() => dashboard.status.value.state === 'stopped')
const hasActiveDownload = computed(() => dashboard.status.value.state === 'running')
const hasScheduler = computed(() => {
  return Boolean(dashboard.scheduler.value.job_details) || dashboard.plans.value.some((plan) => plan.scheduler_running)
})
</script>

<template>
  <main>
    <TooltipProvider>
      <template v-if="dashboard.isDashboardLoading.value">
        <div class="sticky top-0 z-30 border-b bg-background/95 backdrop-blur">
          <div class="mx-auto flex max-w-[1600px] flex-col gap-3 px-6 py-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex flex-wrap items-center gap-3">
              <Skeleton class="h-6 w-36" />
              <Skeleton class="h-6 w-20" />
              <Skeleton class="h-9 w-[220px]" />
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <Skeleton class="h-9 w-24" />
              <Skeleton class="h-9 w-24" />
              <Skeleton class="h-9 w-24" />
            </div>
          </div>
        </div>

        <section class="mx-auto flex max-w-[1600px] flex-col gap-6 px-6 py-6">
          <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <Skeleton v-for="item in 4" :key="`kpi-skeleton-${item}`" class="h-[116px] rounded-xl" />
          </div>

          <Skeleton class="h-[420px] rounded-xl" />

          <div class="grid items-start gap-6 xl:grid-cols-12">
            <div class="grid gap-6 xl:col-span-8">
              <Skeleton class="h-[360px] rounded-xl" />
              <Skeleton class="h-[420px] rounded-xl" />
            </div>

            <div class="grid auto-rows-min content-start gap-6 xl:col-span-4">
              <Skeleton class="h-[360px] rounded-xl" />
            </div>
          </div>

          <Skeleton class="h-[380px] rounded-xl" />
        </section>
      </template>

      <template v-else>
        <TopBar
          v-model:selected-config-name="dashboard.selectedConfigName.value"
          :configs="dashboard.configs.value"
          :has-scheduler="hasScheduler"
          :is-stopped="isStopped"
          :status-label="dashboard.statusLabel.value"
          :status-variant="dashboard.statusVariant.value"
          @create-config="dashboard.openCreateConfig"
          @stop="dashboard.runAction(dashboard.stopSelected)"
          @stop-scheduler="dashboard.runAction(dashboard.stopSelectedScheduler)"
        />

        <section class="mx-auto flex max-w-[1600px] flex-col gap-6 px-6 py-6">
          <KpiGrid
            :active-config-count="dashboard.activeConfigCount.value"
            :download-count="dashboard.status.value.download_count || 0"
            :speed="dashboard.status.value.speed || '0 B/s'"
            :total-bytes="dashboard.status.value.total_bytes || '0 B'"
          />

          <PlanPanel
            :format-date-time="dashboard.formatDateTime"
            :format-plan-schedule="dashboard.formatPlanSchedule"
            :plan-state="dashboard.planState"
            :plans="dashboard.plans.value"
            @delete-plan="dashboard.requestDeleteSelectedConfig($event.name)"
            @edit-plan="dashboard.runAction(() => dashboard.openPlanEditor($event))"
            @open-plan-detail="dashboard.runAction(() => dashboard.openPlanDetail($event))"
            @start-plan="dashboard.runAction(() => dashboard.startPlan($event))"
            @stop-plan="dashboard.requestStopNamedPlan($event)"
          />

          <div class="grid items-start gap-6 xl:grid-cols-12">
            <div class="grid gap-6 xl:col-span-8">
              <SpeedPanel
                :labels="dashboard.speedLabels.value"
                :refreshing="dashboard.isRefreshingDashboard.value"
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
            </div>
          </div>

          <LogHistoryPanel
            v-model:log-enabled="dashboard.logEnabled.value"
            :format-date-time="dashboard.formatDateTime"
            :log-level-class="dashboard.logLevelClass"
            :log-level-options="dashboard.logLevelOptions"
            :log-source-label="dashboard.logSourceLabel"
            :log-source-options="dashboard.logSourceOptions"
            :log-source-variant="dashboard.logSourceVariant"
            :logs="dashboard.logs.value"
            :plans="dashboard.plans.value"
            :selected-log-levels="dashboard.selectedLogLevels.value"
            :selected-log-sources="dashboard.selectedLogSources.value"
            @clear-logs="dashboard.clearLogs"
            @update:selected-log-levels="dashboard.setLogLevels"
            @update:selected-log-sources="dashboard.setLogSources"
          />
        </section>
      </template>

      <Toaster
        position="top-right"
        :expand="false"
        :mobile-offset="{ top: 76, right: 16, bottom: 16, left: 16 }"
        :offset="{ top: 84, right: 24, bottom: 24, left: 24 }"
        :rich-colors="false"
        :visible-toasts="2"
        close-button
        close-button-position="top-right"
      />

      <ConfirmActionDialog
        v-model:open="dashboard.confirmDialogOpen.value"
        :confirm-label="dashboard.confirmDialogActionLabel.value"
        :description="dashboard.confirmDialogDescription.value"
        :pending="dashboard.isConfirmingAction.value"
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
        :last-run="dashboard.editorLastRun.value"
        :saving="dashboard.isSavingConfig.value"
        :test-disabled="hasActiveDownload"
        :testing="dashboard.isStartingTempTest.value"
        @reset="dashboard.resetEditor"
        @save="dashboard.runAction(dashboard.saveEditorConfig)"
        @test="dashboard.runAction(dashboard.startEditorTempTest)"
      />

      <PlanDetailDialog
        v-model:open="dashboard.planDialogOpen.value"
        :format-date-time="dashboard.formatDateTime"
        :plan-detail="dashboard.selectedPlanDetail.value"
      />
    </TooltipProvider>
  </main>
</template>
