import { reactive, computed } from 'vue'
import { authApi, getToken, setToken } from '@/api'

export const authState = reactive({
  token: getToken(),
  user: null,
  loading: false,
})

let mePromise = null

export function useAuth() {
  const isAuthenticated = computed(() => !!authState.token)

  async function fetchMe(force = false) {
    if (!authState.token) {
      authState.user = null
      return null
    }
    if (authState.user && !force) return authState.user
    if (mePromise) return mePromise
    mePromise = authApi
      .me()
      .then((res) => {
        authState.user = (res && res.data) || { username: 'admin' }
        return authState.user
      })
      .catch((error) => {
        authState.user = null
        authState.token = ''
        setToken('')
        throw error
      })
      .finally(() => {
        mePromise = null
      })
    return mePromise
  }

  async function login(username, password) {
    const res = await authApi.login(username, password)
    if (!res || !res.token) throw new Error('登录响应异常')
    authState.token = res.token
    setToken(res.token)
    await fetchMe(true)
    return res
  }

  function clear() {
    authState.token = ''
    authState.user = null
    setToken('')
  }

  async function logout() {
    try {
      if (authState.token) await authApi.logout()
    } catch (error) {
      /* 忽略登出接口错误 */
    }
    clear()
  }

  return { authState, isAuthenticated, login, logout, fetchMe, clear }
}
