<script setup lang="ts">
import { CircleAlertIcon, Trash2Icon } from '@lucide/vue'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Spinner } from '@/components/ui/spinner'

withDefaults(
  defineProps<{
    confirmLabel: string
    description: string
    pending?: boolean
    title: string
    variant?: 'default' | 'destructive'
  }>(),
  {
    pending: false,
    variant: 'default',
  },
)

const open = defineModel<boolean>('open', { required: true })

defineEmits<{
  cancel: []
  confirm: []
}>()
</script>

<template>
  <AlertDialog v-model:open="open">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogMedia :class="variant === 'destructive' ? 'bg-destructive/10 text-destructive' : undefined">
          <Trash2Icon v-if="variant === 'destructive'" />
          <CircleAlertIcon v-else />
        </AlertDialogMedia>
        <AlertDialogTitle>{{ title }}</AlertDialogTitle>
        <AlertDialogDescription>{{ description }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="pending" @click="$emit('cancel')">
          取消
        </AlertDialogCancel>
        <AlertDialogAction
          :variant="variant === 'destructive' ? 'destructive' : 'default'"
          :disabled="pending"
          @click="$emit('confirm')"
        >
          <Spinner v-if="pending" />
          {{ confirmLabel }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
