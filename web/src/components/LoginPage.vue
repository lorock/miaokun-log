<template>
  <div class="login-container">
    <div class="login-wrapper">
      <!-- 左侧装饰区域 -->
      <div class="login-illustration">
        <div class="illustration-content">
          <div class="logo-large">
            <img src="/assets/logo.png" alt="喵坤®" />
          </div>
          <h1 class="brand-title">喵坤®日志排查工具</h1>
          <p class="brand-description">为开发者打造的轻量生产力工具</p>
          <div class="feature-list">
            <span class="feature-item">🚀 极速搜索</span>
            <span class="feature-item">📊 全链路追踪</span>
            <span class="feature-item">⚡ 流式处理</span>
          </div>
        </div>
      </div>

      <!-- 右侧登录表单 -->
      <div class="login-form-wrapper">
        <div class="form-card">
          <div class="form-header">
            <h2 class="form-title">欢迎回来</h2>
            <p class="form-subtitle">请登录您的账户</p>
          </div>

          <el-form
            ref="loginForm"
            :model="form"
            :rules="rules"
            class="login-form"
          >
            <el-form-item prop="username">
              <el-input
                v-model="form.username"
                type="text"
                placeholder="用户名"
                size="large"
                :prefix-icon="UserIcon"
                :disabled="isLoading"
                @keyup.enter="handleLogin"
              />
            </el-form-item>

            <el-form-item prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="密码"
                size="large"
                :prefix-icon="LockIcon"
                :disabled="isLoading"
                show-password
                @keyup.enter="handleLogin"
              />
            </el-form-item>

            <el-form-item class="remember-me">
              <el-checkbox v-model="form.rememberMe" size="default">
                记住我
              </el-checkbox>
              <a href="#" class="forgot-link">忘记密码？</a>
            </el-form-item>

            <el-form-item class="login-button-item">
              <el-button
                type="primary"
                size="large"
                :loading="isLoading"
                :disabled="isLoading"
                class="login-button"
                @click="handleLogin"
              >
                {{ isLoading ? '登录中...' : '登 录' }}
              </el-button>
            </el-form-item>
          </el-form>

          <!-- 错误提示 -->
          <el-alert
            v-if="error"
            :title="error"
            type="error"
            show-icon
            :closable="false"
            class="error-alert"
          />

          <div class="form-footer">
            <span class="footer-text">还没有账户？</span>
            <a href="#" class="register-link">联系管理员注册</a>
          </div>
        </div>

        <div class="security-info">
          <span class="security-icon">🔒</span>
          <span class="security-text">安全连接 · 数据加密</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive } from 'vue';
import { User, Lock } from '@element-plus/icons-vue';
import { useAuth } from '../composables/useAuth';

const emit = defineEmits<{
  loginSuccess: [];
}>();

const { login, isLoading, error } = useAuth();

const UserIcon = User;
const LockIcon = Lock;

const form = reactive({
  username: '',
  password: '',
  rememberMe: false,
});

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名长度在 3 到 50 个字符', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度至少 6 个字符', trigger: 'blur' },
  ],
};

const handleLogin = async () => {
  if (!form.username.trim()) {
    return;
  }
  if (!form.password) {
    return;
  }

  const success = await login({
    username: form.username.trim(),
    password: form.password,
  });

  if (success) {
    emit('loginSuccess');
  }
};
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e8f0 100%);
}

.login-wrapper {
  display: flex;
  width: 900px;
  max-width: 90%;
  background: white;
  border-radius: 16px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  overflow: hidden;
}

.login-illustration {
  flex: 1;
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 50%, #818cf8 100%);
  padding: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-illustration::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, rgba(255,255,255,0.15) 0%, transparent 50%, rgba(255,255,255,0.08) 100%);
}

.illustration-content {
  position: relative;
  z-index: 1;
  text-align: center;
  color: white;
}

.logo-large {
  margin-bottom: 24px;
}

.logo-large img {
  width: 100px;
  height: 100px;
  border-radius: 50%;
  border: 4px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.brand-title {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 12px;
  letter-spacing: 2px;
}

.brand-description {
  font-size: 14px;
  opacity: 0.9;
  margin-bottom: 32px;
}

.feature-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: center;
}

.feature-item {
  background: rgba(255, 255, 255, 0.15);
  padding: 8px 16px;
  border-radius: 20px;
  font-size: 12px;
  backdrop-filter: blur(8px);
}

.login-form-wrapper {
  flex: 1;
  padding: 48px;
  display: flex;
  flex-direction: column;
}

.form-card {
  flex: 1;
}

.form-header {
  text-align: center;
  margin-bottom: 32px;
}

.form-title {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 8px;
}

.form-subtitle {
  font-size: 14px;
  color: #64748b;
}

.login-form {
  margin-bottom: 16px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.login-form :deep(.el-input__wrapper) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

.login-form :deep(.el-input__wrapper:focus-within) {
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}

.login-form :deep(.el-input__prefix) {
  color: #94a3b8;
}

.remember-me {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px !important;
}

.forgot-link {
  font-size: 13px;
  color: #6366f1;
  text-decoration: none;
}

.forgot-link:hover {
  text-decoration: underline;
}

.login-button-item {
  margin-bottom: 16px !important;
}

.login-button {
  width: 100%;
  height: 44px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 100%);
  border: none;
}

.login-button:hover {
  background: linear-gradient(135deg, #4338ca 0%, #4f46e5 100%);
}

.error-alert {
  margin-bottom: 16px;
}

.form-footer {
  text-align: center;
  margin-top: 24px;
}

.footer-text {
  font-size: 14px;
  color: #64748b;
}

.register-link {
  margin-left: 8px;
  font-size: 14px;
  color: #6366f1;
  text-decoration: none;
  font-weight: 500;
}

.register-link:hover {
  text-decoration: underline;
}

.security-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: auto;
  padding-top: 24px;
}

.security-icon {
  font-size: 16px;
}

.security-text {
  font-size: 12px;
  color: #94a3b8;
}

@media (max-width: 768px) {
  .login-wrapper {
    flex-direction: column;
    width: 95%;
  }

  .login-illustration {
    padding: 32px;
  }

  .login-form-wrapper {
    padding: 32px;
  }

  .feature-list {
    gap: 8px;
  }

  .feature-item {
    padding: 6px 12px;
    font-size: 11px;
  }
}
</style>
