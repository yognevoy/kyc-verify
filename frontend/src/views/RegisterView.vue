<script setup>
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
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
    const data = await api.post('/api/auth/register', { email: email.value, password: password.value })
    auth.setAccessToken(data.access_token)
    router.push('/')
  } catch (err) {
    error.value = err.message || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="login">
    <h1>Register</h1>
    <form @submit.prevent="onSubmit">
      <label>
        Email
        <input v-model="email" type="email" required autocomplete="username" />
      </label>
      <label>
        Password
        <input v-model="password" type="password" required autocomplete="new-password" minlength="8" />
      </label>
      <button type="submit" :disabled="loading">{{ loading ? 'Registering...' : 'Register' }}</button>
      <p v-if="error" role="alert">{{ error }}</p>
    </form>
    <p><RouterLink to="/login">Already have an account? Log in</RouterLink></p>
  </main>
</template>

<style scoped>
.login {
  min-height: 80vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
</style>
