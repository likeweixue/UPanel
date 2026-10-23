<template>
  <svg
    class="sparkline"
    :viewBox="`0 0 ${width} ${height}`"
    preserveAspectRatio="none"
    role="img"
    :aria-label="label"
  >
    <defs>
      <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="color" stop-opacity="0.22" />
        <stop offset="100%" :stop-color="color" stop-opacity="0" />
      </linearGradient>
    </defs>
    <line
      v-for="tick in ticks"
      :key="tick"
      :x1="0"
      :x2="width"
      :y1="tick"
      :y2="tick"
      stroke="currentColor"
      stroke-opacity="0.08"
      stroke-width="1"
    />
    <path v-if="areaPath" :d="areaPath" :fill="`url(#${gradientId})`" />
    <path
      v-if="linePath"
      :d="linePath"
      fill="none"
      :stroke="color"
      stroke-width="1.6"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  values: { type: Array, default: () => [] },
  color: { type: String, default: 'var(--color-primary)' },
  label: { type: String, default: '趋势图' },
  max: { type: Number, default: 100 },
})

const width = 320
const height = 56
const gradientId = `spark-${Math.random().toString(36).slice(2, 9)}`
const ticks = [14, 28, 42]

const points = computed(() => {
  const values = props.values.slice(-40)
  if (values.length < 2) return []
  const max = Math.max(Number(props.max) || 1, ...values, 1)
  return values.map((value, index) => {
    const x = (index / (values.length - 1)) * width
    const y = height - (Math.max(0, Number(value) || 0) / max) * (height - 4) - 2
    return [x, y]
  })
})

const linePath = computed(() => {
  if (points.value.length < 2) return ''
  return points.value.map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
})

const areaPath = computed(() => {
  if (points.value.length < 2) return ''
  const line = linePath.value
  return `${line} L${width},${height} L0,${height} Z`
})
</script>
