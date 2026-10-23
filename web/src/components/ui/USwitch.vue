<template>
  <label class="switch">
    <input
      class="switch__input"
      type="checkbox"
      :checked="modelValue"
      :disabled="disabled"
      :aria-label="label || ariaLabel || undefined"
      @change="$emit('update:modelValue', $event.target.checked)"
    />
    <span class="switch__track" aria-hidden="true" />
    <span v-if="label" class="switch__label">{{ label }}</span>
  </label>
</template>

<script setup>
defineProps({
  modelValue: { type: Boolean, default: false },
  label: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})

defineEmits(['update:modelValue'])
</script>
