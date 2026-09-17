<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError, subscribeToStream } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'

const caseInfo = ref(null)
const loading = ref(true)
const error = ref('')
const noCase = ref(false)

let controller = null

onMounted(async () => {
  controller = new AbortController()
  try {
    await subscribeToStream(
      '/api/verification-cases/stream',
      (c) => {
        caseInfo.value = c
        noCase.value = false
        loading.value = false
      },
      { signal: controller.signal },
    )
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      noCase.value = true
    } else if (err.name !== 'AbortError') {
      error.value = err.message
    }
    loading.value = false
  }
})

onUnmounted(() => controller?.abort())
</script>

<template>
  <main>
    <h1>Verification status</h1>
    <p v-if="loading">Loading...</p>
    <template v-else-if="noCase">
      <p>You haven't submitted a verification case yet.</p>
      <RouterLink to="/upload">Upload documents</RouterLink>
    </template>
    <template v-else-if="caseInfo">
      <p><StatusBadge :status="caseInfo.status" /></p>
      <p>Submitted: {{ new Date(caseInfo.created_at).toLocaleString() }}</p>
      <p>Updated: {{ new Date(caseInfo.updated_at).toLocaleString() }}</p>
    </template>
    <p v-if="error" role="alert">{{ error }}</p>
  </main>
</template>
