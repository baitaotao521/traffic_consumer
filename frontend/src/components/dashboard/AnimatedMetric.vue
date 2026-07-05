<script setup lang="ts">
import { TransitionPresets, usePreferredReducedMotion, useTransition } from '@vueuse/core'
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number | string
  fallback?: string
  decimals?: number
  duration?: number
}>(), {
  fallback: '0',
  duration: 650,
})

const metricPattern = /^(-?\d+(?:\.\d+)?)(.*)$/

interface ParsedMetric {
  canAnimate: boolean
  decimals: number
  displayScale: number
  rawText: string
  suffix: string
  sourceValue: number
}

const unitScales: Record<string, number> = {
  B: 1,
  GB: 1024 ** 3,
  KB: 1024,
  MB: 1024 ** 2,
}

function countDecimals(value: string) {
  const decimalIndex = value.indexOf('.')

  return decimalIndex === -1 ? 0 : value.length - decimalIndex - 1
}

function metricScale(suffix: string) {
  const unit = suffix.trim().replace(/\/s$/, '').toUpperCase()

  return unitScales[unit] ?? 1
}

function normalizeNumber(value: number, decimals: number) {
  const zeroThreshold = 1 / 10 ** Math.max(decimals + 1, 1)

  return Math.abs(value) < zeroThreshold ? 0 : value
}

function formatNumber(value: number, decimals: number) {
  return normalizeNumber(value, decimals).toFixed(decimals)
}

function parseMetric(value: number | string, fallback: string): ParsedMetric {
  if (typeof value === 'number') {
    return {
      canAnimate: Number.isFinite(value),
      decimals: props.decimals ?? 0,
      displayScale: 1,
      rawText: Number.isFinite(value) ? String(value) : fallback,
      suffix: '',
      sourceValue: Number.isFinite(value) ? value : 0,
    }
  }

  const rawText = value.trim() || fallback
  const match = rawText.match(metricPattern)

  if (!match) {
    return {
      canAnimate: false,
      decimals: 0,
      displayScale: 1,
      rawText,
      suffix: '',
      sourceValue: 0,
    }
  }

  const numericText = match[1]
  const numericValue = Number(numericText)
  const suffix = match[2] ?? ''
  const displayScale = metricScale(suffix)

  return {
    canAnimate: Number.isFinite(numericValue),
    decimals: props.decimals ?? countDecimals(numericText),
    displayScale,
    rawText,
    suffix,
    sourceValue: Number.isFinite(numericValue) ? numericValue * displayScale : 0,
  }
}

const preferredMotion = usePreferredReducedMotion()
const metric = computed(() => parseMetric(props.value, props.fallback))
const animatedSource = computed(() => metric.value.sourceValue)

const animatedValue = useTransition(animatedSource, {
  disabled: computed(() => preferredMotion.value === 'reduce' || !metric.value.canAnimate),
  duration: computed(() => props.duration),
  easing: TransitionPresets.easeOutCubic,
})

const displayText = computed(() => {
  if (!metric.value.canAnimate) return metric.value.rawText

  // 流量和速率的单位会随后端展示值变化，统一按字节基准过渡，避免单位切换时出现虚假的中间值。
  return `${formatNumber(animatedValue.value / metric.value.displayScale, metric.value.decimals)}${metric.value.suffix}`
})
</script>

<template>
  <span>{{ displayText }}</span>
</template>
