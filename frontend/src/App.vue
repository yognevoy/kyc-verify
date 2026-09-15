<script setup>
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { api } from './api/client'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function logout() {
  await api.post('/api/auth/logout').catch(() => {})
  auth.clear()
  router.push('/login')
}
</script>

<template>
  <nav v-if="auth.isAuthenticated">
    <RouterLink to="/">Home</RouterLink>
    <RouterLink v-if="auth.role === 'reviewer'" to="/admin">Queue</RouterLink>
    <button @click="logout">Log out</button>
  </nav>
  <RouterView />
</template>
