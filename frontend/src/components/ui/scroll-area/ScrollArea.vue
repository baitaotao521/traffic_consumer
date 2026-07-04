<script setup lang="ts">
import type { ScrollAreaRootProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { ref } from "vue"
import {
  ScrollAreaCorner,
  ScrollAreaRoot,
  ScrollAreaViewport,
} from "reka-ui"
import { cn } from "@/lib/utils"
import ScrollBar from "./ScrollBar.vue"

const props = defineProps<ScrollAreaRootProps & { class?: HTMLAttributes["class"] }>()

const delegatedProps = reactiveOmit(props, "class")
const root = ref<{
  viewport?: HTMLElement
  scrollTop?: () => void
  scrollTopLeft?: () => void
} | null>(null)

function scrollTop() {
  root.value?.scrollTop?.()
}

function scrollTopLeft() {
  root.value?.scrollTopLeft?.()
}

function scrollBottom() {
  const viewport = root.value?.viewport
  if (!viewport) return
  viewport.scrollTo({
    top: viewport.scrollHeight,
  })
}

function scrollBottomLeft() {
  const viewport = root.value?.viewport
  if (!viewport) return
  viewport.scrollTo({
    top: viewport.scrollHeight,
    left: 0,
  })
}

defineExpose({
  scrollBottom,
  scrollBottomLeft,
  scrollTop,
  scrollTopLeft,
})
</script>

<template>
  <ScrollAreaRoot
    ref="root"
    data-slot="scroll-area"
    v-bind="delegatedProps"
    :class="cn('relative', props.class)"
  >
    <ScrollAreaViewport
      data-slot="scroll-area-viewport"
      class="size-full rounded-[inherit] transition-[color,box-shadow] outline-none focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-1"
    >
      <slot />
    </ScrollAreaViewport>
    <ScrollBar />
    <ScrollAreaCorner />
  </ScrollAreaRoot>
</template>
