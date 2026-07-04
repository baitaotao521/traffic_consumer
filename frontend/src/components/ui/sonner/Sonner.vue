<script lang="ts" setup>
import {
  CircleCheckIcon,
  Loader2Icon,
  InfoIcon,
  OctagonXIcon,
  TriangleAlertIcon,
  XIcon,
} from '@lucide/vue'
import { computed } from 'vue'

import { reactiveOmit } from '@vueuse/core'
import type { ToasterProps } from 'vue-sonner'
import { Toaster as Sonner } from 'vue-sonner'

import { cn } from '@/lib/utils'

const props = defineProps<ToasterProps>()
const delegatedProps = reactiveOmit(props, 'class', 'toastOptions')
const mergedToastOptions = computed(() => ({
  ...props.toastOptions,
  classes: {
    ...props.toastOptions?.classes,
    toast: cn(
      'rounded-xl border border-border/70 bg-popover/95 px-4 py-3 shadow-lg backdrop-blur-sm',
      props.toastOptions?.classes?.toast,
    ),
    content: cn('grid gap-1', props.toastOptions?.classes?.content),
    title: cn('text-sm font-medium leading-5 text-popover-foreground', props.toastOptions?.classes?.title),
    description: cn('text-sm leading-5 whitespace-pre-line text-muted-foreground', props.toastOptions?.classes?.description),
    icon: cn('mt-0.5 text-muted-foreground [&>svg]:size-4', props.toastOptions?.classes?.icon),
    closeButton: cn(
      'border-border/70 bg-background/90 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground [&>svg]:size-3.5',
      props.toastOptions?.classes?.closeButton,
    ),
    success: cn('border-l-4 border-l-primary', props.toastOptions?.classes?.success),
    info: cn('border-l-4 border-l-ring', props.toastOptions?.classes?.info),
    warning: cn('border-l-4 border-l-chart-4', props.toastOptions?.classes?.warning),
    error: cn('border-l-4 border-l-destructive', props.toastOptions?.classes?.error),
    default: cn('border-l-4 border-l-border', props.toastOptions?.classes?.default),
    actionButton: cn('bg-primary text-primary-foreground hover:bg-primary/90', props.toastOptions?.classes?.actionButton),
    cancelButton: cn('bg-secondary text-secondary-foreground hover:bg-secondary/80', props.toastOptions?.classes?.cancelButton),
  },
}))
</script>

<template>
  <Sonner
    :class="cn('toaster group', props.class)"
    :style="{
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--border-radius': 'var(--radius)',
      '--gray2': 'var(--popover)',
      '--gray3': 'var(--border)',
      '--gray4': 'var(--border)',
      '--gray5': 'var(--border)',
      '--gray12': 'var(--popover-foreground)',
    }"
    :toast-options="mergedToastOptions"
    v-bind="delegatedProps"
  >
    <template #success-icon>
      <CircleCheckIcon />
    </template>
    <template #info-icon>
      <InfoIcon />
    </template>
    <template #warning-icon>
      <TriangleAlertIcon />
    </template>
    <template #error-icon>
      <OctagonXIcon />
    </template>
    <template #loading-icon>
      <div class="animate-spin [&>svg]:size-4">
        <Loader2Icon />
      </div>
    </template>
    <template #close-icon>
      <XIcon />
    </template>
  </Sonner>
</template>
