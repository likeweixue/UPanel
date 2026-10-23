import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  // 使用相对路径 + 服务端注入的 <base>，确保安全入口（/entry/）下资源仍能加载
  base: './',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
  build: {
    target: 'es2019',
    chunkSizeWarningLimit: 900,
    cssCodeSplit: true,
  },
})
