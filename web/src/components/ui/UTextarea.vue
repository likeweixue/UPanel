<template>
  <textarea
    :id="id"
    v-bind="$attrs"
    :value="modelValue"
    class="textarea"
    :class="{ 'textarea--code': code, 'input--sm': size === 'sm' }"
    :aria-invalid="invalid ? 'true' : undefined"
    @input="$emit('update:modelValue', $event.target.value)"
  />
</template>

<script setup>
defineOptions({ inheritAttrs: false })

defineProps({
  modelValue: { type: String, default: '' },
  id: { type: String, default: '' },
  code: { type: Boolean, default: false },
  size: { type: String, default: 'md' },
  invalid: { type: Boolean, default: false },
})

defineEmits(['update:modelValue'])
</script>
