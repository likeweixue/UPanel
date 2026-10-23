<template>
  <div class="toast-host" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div
        v-for="item in toastState.items"
        :key="item.id"
        class="toast"
        :class="`toast--${item.type}`"
        role="status"
      >
        <UIcon :name="item.icon" :size="16" class="toast__icon" />
        <div class="toast__body">
          <p v-if="item.title" class="toast__title">{{ item.title }}</p>
          <p v-if="item.description" class="toast__desc">{{ item.description }}</p>
        </div>
        <UButton
          class="toast__close"
          variant="ghost"
          size="sm"
          icon-only
          icon="x"
          aria-label="关闭提示"
          @click="removeToast(item.id)"
        />
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import UIcon from './UIcon.vue'
import UButton from './UButton.vue'
import { removeToast, toastState } from '@/composables/useToast'
</script>
