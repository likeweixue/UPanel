<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
    :aria-label="iconOnly ? ariaLabel || label : undefined"
    :title="iconOnly ? ariaLabel || label : title"
    :aria-busy="loading ? 'true' : undefined"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="btn__spinner" aria-hidden="true" />
    <UIcon
      v-else-if="icon"
      :name="icon"
      :size="size === 'sm' ? 14 : 16"
      class="btn__icon"
    />
    <span v-if="!iconOnly" class="btn__label"><slot>{{ label }}</slot></span>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import UIcon from './UIcon.vue'

const props = defineProps({
  label: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  title: { type: String, default: '' },
  icon: { type: String, default: '' },
  iconOnly: { type: Boolean, default: false },
  variant: { type: String, default: 'secondary' },
  size: { type: String, default: 'md' },
  type: { type: String, default: 'button' },
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  block: { type: Boolean, default: false },
})

defineEmits(['click'])

const classes = computed(() => [
  'btn',
  `btn--${props.variant}`,
  props.size === 'sm' ? 'btn--sm' : '',
  props.size === 'lg' ? 'btn--lg' : '',
  props.iconOnly ? 'btn--icon' : '',
  props.block ? 'btn--block' : '',
  props.loading ? 'is-loading' : '',
])
</script>
