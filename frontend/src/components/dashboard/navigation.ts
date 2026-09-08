import { ChartNoAxesCombinedIcon, LayoutDashboardIcon, ListChecksIcon, ScrollTextIcon } from '@lucide/vue'

export const dashboardPages = [
  { id: 'overview', label: '总览', title: '流量总览', description: '实时掌握流量消耗与任务运行情况。', icon: LayoutDashboardIcon },
  { id: 'plans', label: '计划管理', title: '计划管理', description: '管理下载配置、执行策略与自动调度。', icon: ListChecksIcon },
  { id: 'threads', label: '线程监控', title: '线程监控', description: '查看每个线程的任务、响应与下载进度。', icon: ChartNoAxesCombinedIcon },
  { id: 'logs', label: '日志与历史', title: '日志与历史', description: '追踪运行事件，回看计划执行记录。', icon: ScrollTextIcon },
] as const
