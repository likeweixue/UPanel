<template>
  <span class="app-logo" :class="`app-logo--${size}`" aria-hidden="true">
    <span class="app-logo__text">UP</span>
  </span>
</template>

<script setup>
defineProps({
  size: { type: String, default: 'md' },
})
</script>
