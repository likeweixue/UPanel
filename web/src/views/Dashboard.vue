<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-6">服务器总览</h1>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-gray-500 text-sm">Docker 版本</h3>
        <p class="text-2xl font-bold mt-2">{{ dockerVersion }}</p>
      </div>
      <div class="bg-white rounded-lg shadow p-6">
        <h3 class="text-gray-500 text-sm">API 状态</h3>
        <p class="text-2xl font-bold mt-2 text-green-500">● 运行中</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const dockerVersion = ref('加载中...')

onMounted(async () => {
  try {
    const res = await axios.get('/api/docker/version')
    dockerVersion.value = res.data.version
  } catch (err) {
    dockerVersion.value = '连接失败'
  }
})
</script>