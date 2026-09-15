<script setup>
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
</script>

<template>
  <main>
    <h1>Signed in</h1>
    <p>Role: {{ auth.role }}</p>
    <p>Document upload is not available yet.</p>
  </main>
</template>
