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
        <p class="login__name">{{ panelName || 'UPanel' }}</p>
      </div>

      <div class="login__heading">
        <h1 id="login-title" class="login__title">登录</h1>
      </div>

      <div v-if="error" class="alert alert--danger" role="alert">
        <UIcon name="alert" :size="16" class="alert__icon" />
        <div class="alert__body">{{ error }}</div>
      </div>

      <form class="login__form" novalidate @submit.prevent="submit">
        <div class="login__field">
          <UInput
            id="login-username"
            v-model="username"
            :invalid="!!fieldErrors.username"
            autocomplete="username"
            placeholder="用户名"
            aria-label="用户名"
            autofocus
            aria-describedby="login-username-error"
            @keyup.enter="submit"
          />
          <p v-if="fieldErrors.username" id="login-username-error" class="field__error">
            {{ fieldErrors.username }}
          </p>
        </div>

        <div class="login__field">
          <UInput
            id="login-password"
            v-model="password"
            type="password"
            revealable
            :invalid="!!fieldErrors.password"
            autocomplete="current-password"
            placeholder="请输入管理员密码"
            aria-label="密码"
            aria-describedby="login-password-error"
            @keyup.enter="submit"
          />
          <p v-if="fieldErrors.password" id="login-password-error" class="field__error">
            {{ fieldErrors.password }}
          </p>
        </div>

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
    </section>
  </main>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UButton from '@/components/ui/UButton.vue'
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
const fieldErrors = reactive({ username: '', password: '' })

const isDark = computed(() => isDarkTheme(themeState.effective))

onMounted(async () => {
  try {
    const info = await panelApi.info()
    panelName.value = info.name || 'UPanel'
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
  justify-content: center;
  text-align: center;
}

.login__name {
  font-size: var(--text-lg);
  font-weight: 700;
  letter-spacing: -0.01em;
}

.login__heading {
  margin: 10px 0 22px;
  text-align: center;
}

.login__title {
  font-size: 24px;
  font-weight: 650;
  letter-spacing: -0.02em;
}

.login__form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 16px;
}

.login__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.login__field :deep(.input),
.login__field :deep(.input-affix .input) {
  height: 48px;
  padding: 0 14px;
  border-color: transparent;
  background: var(--color-surface-3);
  font-size: var(--text-base);
  transition: background var(--transition), color var(--transition);
}

.login__field :deep(.input::placeholder) {
  color: var(--color-text-subtle);
}

.login__field :deep(.input:hover) {
  border-color: transparent;
  background: var(--color-surface-2);
}

.login__field :deep(.input:focus) {
  border-color: transparent;
  background: var(--color-primary-soft);
  box-shadow: none;
}

.login__field :deep(.input[aria-invalid="true"]) {
  border-color: transparent;
  background: var(--color-danger-soft);
}

.login__field :deep(.input-affix) {
  border-radius: var(--radius-sm);
}

.login__field :deep(.input-affix .input) {
  padding-right: 46px;
}

.login__field :deep(.input-affix__btn) {
  right: 7px;
}

.login__field :deep(.input:-webkit-autofill) {
  -webkit-box-shadow: 0 0 0 1000px var(--color-surface-3) inset;
  -webkit-text-fill-color: var(--color-text);
}

@media (max-width: 480px) {
  .login {
    justify-content: flex-start;
  }

  .login__card {
    padding: 22px 18px;
  }
}
</style>
