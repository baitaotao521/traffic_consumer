<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { useBrowserLocation } from '@vueuse/core'
import { ArrowUpRightIcon, WifiOffIcon } from '@lucide/vue'
import DashboardSidebar from '@/components/dashboard/DashboardSidebar.vue'
import { dashboardPages } from '@/components/dashboard/navigation'
import { SidebarProvider } from '@/components/ui/sidebar'
import { Alert, AlertAction, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

import KpiGrid from '@/components/dashboard/KpiGrid.vue'
import LogHistoryPanel from '@/components/dashboard/LogHistoryPanel.vue'
import PlanPanel from '@/components/dashboard/PlanPanel.vue'
import SpeedPanel from '@/components/dashboard/SpeedPanel.vue'
import ThreadPanel from '@/components/dashboard/ThreadPanel.vue'
import TopBar from '@/components/dashboard/TopBar.vue'
import { Skeleton } from '@/components/ui/skeleton'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useDashboard } from '@/composables/useDashboard'

// 配置编辑、计划详情和确认动作只在用户触发后出现，避免首屏等待弹层依赖。
const ConfigSheet = defineAsyncComponent(() => import('@/components/dashboard/ConfigSheet.vue'))
const PlanDetailDialog = defineAsyncComponent(() => import('@/components/dashboard/PlanDetailDialog.vue'))
const ConfirmActionDialog = defineAsyncComponent(() => import('@/components/dashboard/ConfirmActionDialog.vue'))

const mainContent = ref<HTMLElement | null>(null)
const dashboard = useDashboard()
const location = useBrowserLocation()
const currentPage = computed(() => dashboardPages.find((page) => `#${page.id}` === location.value.hash) ?? dashboardPages[0])

const isStopped = computed(() => dashboard.status.value.state === 'stopped')
const hasActiveDownload = computed(() => dashboard.status.value.state === 'running')
const hasScheduler = computed(() => {
  return Boolean(dashboard.scheduler.value.job_details) || dashboard.plans.value.some((plan) => plan.scheduler_running)
})
</script>

<template>
  <div>
    <a href="#main-content" @click.prevent="mainContent?.focus()" class="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:bg-card focus:p-3">跳转到主要内容</a>
    <TooltipProvider>
      <SidebarProvider>
        <DashboardSidebar :active-page="currentPage.id" :plan-count="dashboard.plans.value.length" :active-count="dashboard.activeConfigCount.value" />
        <div class="min-w-0 flex-1">
        <TopBar
          :page-label="currentPage.label"
          :loading="dashboard.isDashboardLoading.value || !dashboard.hasDashboardData.value"
          :connection-error="Boolean(dashboard.dashboardError.value)"
          :has-scheduler="hasScheduler"
          :is-stopped="isStopped"
          :status-label="dashboard.statusLabel.value"
          :status-variant="dashboard.statusVariant.value"
          @create-config="dashboard.openCreateConfig"
          @stop="dashboard.runAction(dashboard.stopSelected)"
          @stop-scheduler="dashboard.runAction(dashboard.stopSelectedScheduler)"
        />
          <main id="main-content" ref="mainContent" tabindex="-1" class="mx-auto flex w-full max-w-[1560px] flex-col gap-7 px-4 py-7 sm:px-7 lg:px-9">
            <div class="flex flex-wrap items-end justify-between gap-4">
              <div><p class="mb-2 text-xs font-medium tracking-widest text-primary">工作空间 / {{ currentPage.label }}</p><h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">{{ currentPage.title }}</h1><p class="mt-2 text-sm text-muted-foreground">{{ currentPage.description }}</p></div>
              <Button v-if="currentPage.id === 'overview'" variant="outline" as-child><a href="#plans">管理计划<ArrowUpRightIcon data-icon="inline-end" /></a></Button>
            </div>
            <Alert v-if="dashboard.dashboardError.value" variant="destructive">
              <WifiOffIcon /><AlertTitle>{{ dashboard.hasDashboardData.value ? '数据更新中断' : '无法连接服务' }}</AlertTitle>
              <AlertDescription>{{ dashboard.hasDashboardData.value ? '当前显示上次成功获取的数据。' : '请检查服务状态后重新连接。' }}</AlertDescription>
              <AlertAction><Button size="sm" variant="outline" :disabled="dashboard.isRefreshingDashboard.value" @click="dashboard.runAction(dashboard.refreshDashboard)">重新连接</Button></AlertAction>
            </Alert>
            <div v-if="dashboard.isDashboardLoading.value" class="flex flex-col gap-6" role="status" aria-label="正在加载控制台">
              <div class="grid grid-cols-2 gap-4 xl:grid-cols-4"><Skeleton v-for="item in 4" :key="item" class="h-36" /></div>
              <Skeleton class="h-[340px]" /><Skeleton class="h-[280px]" />
            </div>
            <template v-else-if="dashboard.hasDashboardData.value">
              <section v-if="currentPage.id === 'overview'" class="page-enter flex min-w-0 flex-col gap-6" aria-label="流量总览">
          <KpiGrid
            :active-config-count="dashboard.activeConfigCount.value"
            :download-count="dashboard.status.value.download_count || 0"
            :speed="dashboard.status.value.speed || '0 B/s'"
            :total-bytes="dashboard.status.value.total_bytes || '0 B'"
          />
          <SpeedPanel
            :current-speed="dashboard.status.value.speed || '0 B/s'"
            :labels="dashboard.speedLabels.value"
            :refreshing="dashboard.isRefreshingDashboard.value"
            :thread-active-values="dashboard.threadActiveValues.value"
            :url-chart-labels="dashboard.urlChartLabels.value"
            :url-chart-values="dashboard.urlChartValues.value"
            :url-stats="dashboard.urlStats.value"
            :values="dashboard.speedValues.value"
            @refresh="dashboard.runAction(dashboard.refreshDashboard)"
          />
              </section>
              <section v-if="currentPage.id === 'overview' || currentPage.id === 'plans'" class="page-enter min-w-0" aria-label="配置计划">
          <PlanPanel
            :compact="currentPage.id === 'overview'"
            @create-config="dashboard.openCreateConfig"
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
              </section>
              <section v-if="currentPage.id === 'threads'" class="page-enter min-w-0" aria-label="线程状态">
          <ThreadPanel :thread-counts="dashboard.threadCounts.value" :thread-rows="dashboard.threadRows.value" />
              </section>
              <section v-if="currentPage.id === 'logs'" class="page-enter min-w-0" aria-label="日志与历史">
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
            <footer class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground"><span>Traffic Consumer</span><span>{{ dashboard.dashboardError.value ? '等待重新连接' : '运行数据自动更新' }}</span></footer>
          </main>
        </div>
      </SidebarProvider>

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

      <ConfigSheet
        v-if="dashboard.editorOpen.value"
        v-model:open="dashboard.editorOpen.value"
        :cron-preview="dashboard.cronPreview.value"
        :errors="dashboard.editorValidationErrors.value"
        :form="dashboard.form"
        :format-date-time="dashboard.formatDateTime"
        :last-run="dashboard.editorLastRun.value"
        :saving="dashboard.isSavingConfig.value"
        :test-disabled="hasActiveDownload"
        :testing="dashboard.isStartingTempTest.value"
        @close-request="dashboard.requestCloseEditor"
        @reset="dashboard.requestResetEditor"
        @save="dashboard.runAction(dashboard.saveEditorConfig)"
        @test="dashboard.runAction(dashboard.startEditorTempTest)"
      />

      <PlanDetailDialog
        v-if="dashboard.planDialogOpen.value"
        v-model:open="dashboard.planDialogOpen.value"
        :format-date-time="dashboard.formatDateTime"
        :plan-detail="dashboard.selectedPlanDetail.value"
      />

      <ConfirmActionDialog
        v-if="dashboard.confirmDialogOpen.value"
        v-model:open="dashboard.confirmDialogOpen.value"
        :confirm-label="dashboard.confirmDialogActionLabel.value"
        :description="dashboard.confirmDialogDescription.value"
        :pending="dashboard.isConfirmingAction.value"
        :title="dashboard.confirmDialogTitle.value"
        :variant="dashboard.confirmDialogVariant.value"
        @cancel="dashboard.cancelPendingAction"
        @confirm="dashboard.confirmPendingAction"
      />
    </TooltipProvider>
  </div>
</template>
