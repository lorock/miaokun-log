<template>
  <div class="app-container">
    <header class="header">
      <div class="logo">
        <el-tooltip content="喵坤®" placement="bottom">
          <img 
            src="/assets/logo.png" 
            alt="喵坤®" 
            :class="['logo-icon', { zoomed: logoZoomed }]" 
            @click="toggleLogoZoom"
          />
        </el-tooltip>
      </div>
      <div class="header-title">
        <span class="title-main">喵坤®日志排查工具</span>
      </div>
      <div class="header-features">
        <span class="feature-tag">极速搜索</span>
        <span class="feature-tag">全链路追踪</span>
        <span class="feature-tag">流式处理</span>
      </div>
    </header>

    <SearchForm :is-streaming="isStreaming" :stats="stats" @search="handleSearch" @stop="stopSearch" />

    <div class="log-container">
      <LogList :logs="logs" />
    </div>

    <footer class="footer">
      <div class="footer-left">
        <span class="version">{{ version }}</span>
        <span class="footer-dot">·</span>
        <span>基于 Ripgrep 引擎</span>
      </div>
      <div class="footer-center">
        <span class="footer-text">为开发者打造的轻量生产力工具</span>
      </div>
      <div class="footer-right">
        <a href="https://gitee.com/lorock/miaokun-log" target="_blank" class="footer-link">Gitee</a>
        <span class="footer-dot">·</span>
        <a href="https://gitee.com/lorock/miaokun-log/issues" target="_blank" class="footer-link">反馈</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useLogStream } from './composables/useLogStream';
import { useVersion } from './composables/useVersion';
import SearchForm from './components/SearchForm.vue';
import LogList from './components/LogList.vue';
import type { SearchRequest } from './types';

const { logs, stats, isStreaming, start, stop: stopSearch } = useLogStream();
const { version } = useVersion();

const logoZoomed = ref(false);

const toggleLogoZoom = () => {
  logoZoomed.value = !logoZoomed.value;
};

const handleSearch = (request: SearchRequest) => {
  start(request);
};
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #f5f7fa;
}

.header {
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 50%, #818cf8 100%);
  padding: 16px 24px;
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.35);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  position: relative;
  overflow: hidden;
}

.header::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255,255,255,0.15) 0%, transparent 50%, rgba(255,255,255,0.08) 100%);
  pointer-events: none;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  object-fit: cover;
  cursor: pointer;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.logo-icon:hover {
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.logo-icon.zoomed {
  width: 120px;
  height: 120px;
  position: fixed;
  top: 20px;
  left: 20px;
  z-index: 9999;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  border: 3px solid white;
}



.header-title {
  display: flex;
  align-items: center;
}

.title-main {
  color: white;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
}

.header-features {
  display: flex;
  gap: 10px;
}

.feature-tag {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  padding: 5px 14px;
  border-radius: 14px;
  font-size: 12px;
  font-weight: 500;
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.log-container {
  flex: 1;
  min-height: 300px;
  margin: 16px 24px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.footer {
  padding: 14px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #909399;
  font-size: 12px;
  background: white;
  border-top: 1px solid #ebeef5;
  flex-wrap: wrap;
  gap: 8px;
}

.footer-left,
.footer-center,
.footer-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.footer-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
}

.footer-right {
  justify-content: flex-end;
}

.footer-dot {
  margin: 0 6px;
}

.footer-link {
  color: #667eea;
  text-decoration: none;
}

.footer-link:hover {
  text-decoration: underline;
}

.version {
  font-weight: 500;
  color: #667eea;
}

@media (max-width: 768px) {
  .header {
    flex-direction: column;
    text-align: center;
  }
  
  .header-features {
    justify-content: center;
  }
  
  .footer-center {
    position: static;
    transform: none;
    order: 3;
    width: 100%;
    justify-content: center;
  }
  
  .footer-left,
  .footer-right {
    width: 50%;
  }
  
  .footer-right {
    justify-content: flex-end;
  }
}
</style>
