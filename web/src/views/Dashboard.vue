<template>
  <div>
    <h1 class="text-2xl font-bold text-gray-800 mb-6">服务器总览</h1>
    
    <!-- 系统信息卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">操作系统</h3>
          <div class="w-8 h-8 bg-blue-50 rounded-lg flex items-center justify-center">
            <svg class="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"></path>
            </svg>
          </div>
        </div>
        <p class="text-2xl font-bold text-gray-800">{{ systemInfo.os }}</p>
        <p class="text-xs text-gray-400 mt-2">{{ systemInfo.arch }} / {{ systemInfo.cpu }} 核心</p>
      </div>

      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">运行时间</h3>
          <div class="w-8 h-8 bg-green-50 rounded-lg flex items-center justify-center">
            <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
          </div>
        </div>
        <p class="text-2xl font-bold text-gray-800">{{ systemInfo.uptime || '计算中...' }}</p>
        <p class="text-xs text-gray-400 mt-2">面板运行时间</p>
      </div>

      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">Docker 版本</h3>
          <div class="w-8 h-8 bg-purple-50 rounded-lg flex items-center justify-center">
            <svg class="w-5 h-5 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"></path>
            </svg>
          </div>
        </div>
        <p class="text-2xl font-bold text-gray-800">{{ systemInfo.docker_version || '未连接' }}</p>
        <p class="text-xs text-gray-400 mt-2">Docker Engine</p>
      </div>

      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">Go 版本</h3>
          <div class="w-8 h-8 bg-orange-50 rounded-lg flex items-center justify-center">
            <svg class="w-5 h-5 text-orange-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path>
            </svg>
          </div>
        </div>
        <p class="text-2xl font-bold text-gray-800">{{ systemInfo.go_version }}</p>
        <p class="text-xs text-gray-400 mt-2">UPanel 后端</p>
      </div>
    </div>

    <!-- 容器统计卡片 -->
    <h2 class="text-xl font-bold text-gray-800 mb-4">容器统计</h2>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">总容器数</h3>
          <span class="text-3xl font-bold text-gray-800">{{ systemInfo.containers || 0 }}</span>
        </div>
        <div class="flex justify-between text-sm">
          <span class="text-green-600">运行中: {{ systemInfo.running || 0 }}</span>
          <span class="text-gray-400">已停止: {{ systemInfo.stopped || 0 }}</span>
        </div>
      </div>

      <div class="bg-white rounded-xl shadow-sm border border-gray-100 p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-gray-500 text-sm font-medium">镜像数量</h3>
          <span class="text-3xl font-bold text-gray-800">{{ systemInfo.images || 0 }}</span>
        </div>
        <div class="text-sm text-gray-400">Docker 镜像</div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="text-center py-12">
      <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
      <p class="mt-2 text-gray-500">加载中...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const systemInfo = ref({
  os: '加载中...',
  arch: '',
  cpu: 0,
  go_version: '',
  uptime: '',
  docker_version: '',
  containers: 0,
  running: 0,
  stopped: 0,
  images: 0
})
const loading = ref(true)

const fetchSystemInfo = async () => {
  try {
    const res = await axios.get('/api/system/info')
    systemInfo.value = res.data.data
  } catch (err) {
    console.error('获取系统信息失败:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSystemInfo()
})
</script>