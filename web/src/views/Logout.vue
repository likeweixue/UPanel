<template>
  <div class="text-center py-12">
    <h1 class="text-2xl font-bold text-gray-800">退出登录</h1>
    <p class="mt-4 text-gray-500">正在退出...</p>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

onMounted(() => {
  localStorage.removeItem('token')
  setTimeout(() => {
    router.push('/login')
  }, 1000)
})
</script>