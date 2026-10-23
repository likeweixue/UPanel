<template>
  <div class="field" :class="{ 'field--full': full }">
    <label v-if="label" class="field__label" :for="forId">
      {{ label }}
      <span v-if="required" class="field__required" aria-hidden="true">*</span>
      <span v-if="required" class="sr-only">（必填）</span>
    </label>
    <slot />
    <p v-if="error" class="field__error" role="alert">{{ error }}</p>
    <p v-else-if="hint" class="field__hint">{{ hint }}</p>
  </div>
</template>

<script setup>
defineProps({
  label: { type: String, default: '' },
  forId: { type: String, default: '' },
  hint: { type: String, default: '' },
  error: { type: String, default: '' },
  required: { type: Boolean, default: false },
  full: { type: Boolean, default: false },
})
</script>
