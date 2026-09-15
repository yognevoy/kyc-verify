<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/client'
import { useAuthStore } from '../stores/auth'

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const auth = useAuthStore()
const router = useRouter()

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    const data = await api.post('/api/auth/login', { email: email.value, password: password.value })
    auth.setAccessToken(data.access_token)
    router.push(auth.role === 'reviewer' ? '/admin' : '/')
  } catch (err) {
    error.value = err.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main>
    <h1>Log in</h1>
    <form @submit.prevent="onSubmit">
      <label>
        Email
        <input v-model="email" type="email" required autocomplete="username" />
      </label>
      <label>
        Password
        <input v-model="password" type="password" required autocomplete="current-password" />
      </label>
      <button type="submit" :disabled="loading">{{ loading ? 'Logging in...' : 'Log in' }}</button>
      <p v-if="error" role="alert">{{ error }}</p>
    </form>
  </main>
</template>
