<template>
  <div class="logout">
    <div class="logout__card">
      <span class="spinner" aria-hidden="true" />
      <p>正在退出登录…</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const { logout } = useAuth()

onMounted(async () => {
  await logout()
  router.replace('/login')
})
</script>

<style scoped>
.logout {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}

.logout__card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 22px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  color: var(--color-text-muted);
}
</style>
