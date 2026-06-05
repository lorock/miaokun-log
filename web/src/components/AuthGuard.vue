<template>
  <div v-if="isLoading" class="auth-loading">
    <el-spinner size="large" />
    <span class="loading-text">正在验证身份...</span>
  </div>
  <div v-else-if="isAuthenticated">
    <slot />
  </div>
  <div v-else>
    <LoginPage @login-success="handleLoginSuccess" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue';
import { useAuth, useAuthState } from '../composables/useAuth';
import LoginPage from './LoginPage.vue';

const { initAuth } = useAuth();
const { isAuthenticated, isLoading } = useAuthState();

const isReady = ref(false);

const handleLoginSuccess = () => {
  isReady.value = true;
  setTimeout(() => {
    initAuth();
  }, 100);
};

const checkAuth = async () => {
  initAuth();
  setTimeout(() => {
    isReady.value = true;
  }, 500);
};

watch(isAuthenticated, (newVal, oldVal) => {
  if (newVal) {
    isReady.value = true;
  } else if (oldVal === true && newVal === false) {
    // User was logged in, now logged out - reset state
    isReady.value = false;
    setTimeout(() => {
      isReady.value = true;
    }, 300);
  }
});

onMounted(() => {
  checkAuth();
});
</script>

<style scoped>
.auth-loading {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e8f0 100%);
  gap: 16px;
}

.loading-text {
  font-size: 14px;
  color: #64748b;
}
</style>
