<script setup>
import { RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
</script>

<template>
  <main>
    <h1>Welcome</h1>
    <template v-if="auth.role === 'applicant'">
      <p><RouterLink to="/upload">Upload documents</RouterLink></p>
      <p><RouterLink to="/status">Check verification status</RouterLink></p>
    </template>
    <template v-else>
      <p><RouterLink to="/admin">Review queue</RouterLink></p>
    </template>
  </main>
</template>
