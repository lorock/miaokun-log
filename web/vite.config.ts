import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:9528',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          // 将 node_modules 中的包分离到不同的 chunk
          if (id.includes('node_modules')) {
            // Element Plus 单独分块
            if (id.includes('element-plus')) {
              return 'element-plus'
            }
            // Vue 相关库
            if (id.includes('vue') || id.includes('@vue')) {
              return 'vue-vendor'
            }
            // 其他第三方库
            return 'vendor'
          }
        }
      }
    },
    chunkSizeWarningLimit: 800
  }
})
