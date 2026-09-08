<script setup lang="ts">
import { ActivityIcon, ArrowUpRightIcon, NetworkIcon } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import {
  Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupContent,
  SidebarGroupLabel, SidebarHeader, SidebarMenu, SidebarMenuButton, SidebarMenuItem,
  SidebarSeparator, useSidebar,
} from '@/components/ui/sidebar'
import { dashboardPages } from './navigation'

defineProps<{ activePage: string; planCount: number; activeCount: number }>()
const { setOpenMobile } = useSidebar()
</script>

<template>
  <Sidebar>
    <SidebarHeader>
      <a href="#overview" class="flex items-center gap-3 px-3 py-5" aria-label="Traffic Consumer 总览" @click="setOpenMobile(false)">
        <div class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground"><ActivityIcon class="size-6" /></div>
        <div class="min-w-0"><div class="text-[15px] font-semibold tracking-tight">Traffic Consumer</div><div class="mt-1 text-xs text-muted-foreground">流量管理控制台</div></div>
      </a>
    </SidebarHeader>
    <SidebarContent>
      <SidebarGroup>
        <SidebarGroupLabel>工作空间</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="page in dashboardPages" :key="page.id">
              <SidebarMenuButton :is-active="activePage === page.id" size="lg" as-child>
                <a :href="`#${page.id}`" :aria-current="activePage === page.id ? 'page' : undefined" @click="setOpenMobile(false)">
                  <component :is="page.icon" /><span>{{ page.label }}</span>
                  <Badge v-if="page.id === 'plans'" variant="outline" class="ml-auto">{{ planCount }}</Badge>
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>
    <SidebarFooter>
      <SidebarSeparator />
      <div class="flex items-center gap-3 px-3 py-4">
        <NetworkIcon class="size-5 text-primary" />
        <div class="flex-1"><div class="text-sm font-medium">{{ activeCount }} 个活动配置</div><div class="mt-1 text-xs text-muted-foreground">运行中与调度中</div></div>
        <ArrowUpRightIcon class="size-4 text-muted-foreground" />
      </div>
    </SidebarFooter>
  </Sidebar>
</template>
