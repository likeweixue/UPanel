<template>
  <section class="widget dashboard-widget" :class="`widget--${span}`">
    <header class="widget__header">
      <div class="widget__title-wrap">
        <UIcon v-if="icon" :name="icon" :size="17" class="widget__icon" />
        <div>
          <h2 class="widget__title">{{ title }}</h2>
          <p v-if="subtitle" class="widget__subtitle">{{ subtitle }}</p>
        </div>
      </div>
      <div v-if="$slots.actions" class="widget__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="widget__body" :class="{ 'widget__body--flush': flush }">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="widget__footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<script setup>
import UIcon from './UIcon.vue'

defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  icon: { type: String, default: '' },
  span: { type: [Number, String], default: 12 },
  flush: { type: Boolean, default: false },
})
</script>
