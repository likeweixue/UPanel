<template>
  <div class="empty">
    <div class="empty__icon"><UIcon :name="icon" :size="22" /></div>
    <p class="empty__title">{{ title }}</p>
    <p v-if="description" class="empty__desc">{{ description }}</p>
    <div v-if="$slots.default" class="empty__action"><slot /></div>
  </div>
</template>

<script setup>
import UIcon from './UIcon.vue'

defineProps({
  icon: { type: String, default: 'box' },
  title: { type: String, default: '暂无数据' },
  description: { type: String, default: '' },
})
</script>
