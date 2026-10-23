<template>
  <span
    class="status-dot"
    :class="`status-dot--${tone}`"
    role="status"
    :aria-label="ariaLabel || `${text || '状态'}: ${statusText}`"
  >
    <span class="status-dot__dot" aria-hidden="true" />
    <span v-if="text" class="status-dot__label">{{ text }}</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { statusLabel, statusTone } from '@/composables/useFormat'

const props = defineProps({
  status: { type: String, default: 'unknown' },
  text: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
})

const tone = computed(() => statusTone(props.status))
const statusText = computed(() => props.text || statusLabel(props.status))
</script>
