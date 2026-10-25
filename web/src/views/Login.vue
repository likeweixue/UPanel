<template>
  <main class="login">
    <div class="login__topbar">
      <UButton
        variant="ghost"
        icon-only
        :icon="isDark ? 'sun' : 'moon'"
        :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
        @click="toggleTheme"
      />
    </div>

    <section class="login__card" aria-labelledby="login-title">
      <div class="login__brand">
        <ULogo size="lg" />
        <div>
          <p class="login__name">{{ panelName || 'UPanel' }}</p>
          <p class="login__tagline">轻量级 Linux 服务器管理面板</p>
        </div>
      </div>

      <div class="login__heading">
        <h1 id="login-title" class="login__title">登录管理面板</h1>
        <p class="login__subtitle">使用管理员账号继续</p>
      </div>

      <div v-if="error" class="alert alert--danger" role="alert">
        <UIcon name="alert" :size="16" class="alert__icon" />
        <div class="alert__body">{{ error }}</div>
      </div>

      <form class="login__form" novalidate @submit.prevent="submit">
        <UField label="用户名" for-id="login-username" required>
          <UInput
            id="login-username"
            v-model="username"
            :invalid="!!fieldErrors.username"
            autocomplete="username"
            placeholder="admin"
            autofocus
            aria-describedby="login-username-error"
            @keyup.enter="submit"
          />
          <p v-if="fieldErrors.username" id="login-username-error" class="field__error">
            {{ fieldErrors.username }}
          </p>
        </UField>

        <UField label="密码" for-id="login-password" required>
          <UInput
            id="login-password"
            v-model="password"
            type="password"
            revealable
            :invalid="!!fieldErrors.password"
            autocomplete="current-password"
            placeholder="请输入管理员密码"
            @keyup.enter="submit"
          />
          <p v-if="fieldErrors.password" class="field__error">{{ fieldErrors.password }}</p>
        </UField>

        <UButton
          type="submit"
          variant="primary"
          size="lg"
          block
          :loading="loading"
          icon="lock"
          label="登录"
        />
      </form>

      <div class="login__note">
        <UIcon name="shield" :size="14" />
        <span>连续登录失败会触发临时锁定；请勿在公共网络保存密码。</span>
      </div>
    </section>

    <footer class="login__footer">
      <span>{{ panelName || 'UPanel' }} <span v-if="version">· {{ version }}</span></span>
      <span>Docker 容器化管理</span>
    </footer>
  </main>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
import ULogo from '@/components/ui/ULogo.vue'
import UField from '@/components/ui/UField.vue'
import UIcon from '@/components/ui/UIcon.vue'
import UInput from '@/components/ui/UInput.vue'
import { errText, panelApi } from '@/api'
import { useAuth } from '@/composables/useAuth'
import { isDarkTheme, toggleTheme, useTheme } from '@/composables/useTheme'
import { toastSuccess } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()
const { login, clear } = useAuth()
const { themeState } = useTheme()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const panelName = ref('UPanel')
const version = ref('')
const fieldErrors = reactive({ username: '', password: '' })

const isDark = computed(() => isDarkTheme(themeState.effective))

onMounted(async () => {
  try {
    const info = await panelApi.info()
    panelName.value = info.name || 'UPanel'
    version.value = info.version || ''
  } catch (e) {
    /* 面板信息不可用时保持默认 */
  }
})

function validate() {
  fieldErrors.username = username.value.trim() ? '' : '请输入用户名'
  fieldErrors.password = password.value ? '' : '请输入密码'
  return !fieldErrors.username && !fieldErrors.password
}

async function submit() {
  if (loading.value) return
  error.value = ''
  if (!validate()) return
  loading.value = true
  try {
    await login(username.value.trim(), password.value)
    toastSuccess('登录成功', '正在进入管理面板')
    const requested = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    const redirect =
      requested.startsWith('/') && !requested.startsWith('//') ? requested : '/dashboard'
    router.replace(redirect)
  } catch (e) {
    error.value = errText(e, '登录失败，请检查用户名和密码')
    clear()
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 72px 16px 24px;
  background: var(--color-bg);
}

.login__topbar {
  position: absolute;
  top: 12px;
  right: 12px;
}

.login__card {
  width: 100%;
  max-width: 400px;
  padding: 28px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.login__brand {
  display: flex;
  align-items: center;
  gap: 11px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--color-border);
}

.login__name {
  font-size: var(--text-lg);
  font-weight: 700;
  letter-spacing: -0.01em;
}

.login__tagline {
  color: var(--color-text-muted);
  font-size: var(--text-sm);
}

.login__heading {
  margin: 22px 0 18px;
}

.login__title {
  font-size: 21px;
  font-weight: 650;
  letter-spacing: -0.02em;
}

.login__subtitle {
  margin-top: 4px;
  color: var(--color-text-muted);
  font-size: var(--text-md);
}

.login__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 16px;
}

.login__note {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px solid var(--color-border);
  color: var(--color-text-subtle);
  font-size: var(--text-sm);
  line-height: 1.5;
}

.login__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  max-width: 400px;
  margin-top: 18px;
  color: var(--color-text-subtle);
  font-size: var(--text-sm);
}

@media (max-width: 480px) {
  .login {
    justify-content: flex-start;
  }

  .login__card {
    padding: 22px 18px;
  }

  .login__footer {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}
</style>
