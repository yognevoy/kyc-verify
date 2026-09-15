import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

function decodeJWT(token) {
  const payload = token.split('.')[1]
  const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
  return JSON.parse(json)
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(null)

  const claims = computed(() => (accessToken.value ? decodeJWT(accessToken.value) : null))
  const userId = computed(() => claims.value?.sub ?? null)
  const role = computed(() => claims.value?.role ?? null)
  const isAuthenticated = computed(() => accessToken.value !== null)

  function setAccessToken(token) {
    accessToken.value = token
  }

  function clear() {
    accessToken.value = null
  }

  return { accessToken, userId, role, isAuthenticated, setAccessToken, clear }
})
