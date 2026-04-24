<template>
  <router-view />
</template>

<script setup>
import { onMounted } from 'vue'
import { initTheme } from '@/themes'

onMounted(() => {
  initTheme()
})
</script>
