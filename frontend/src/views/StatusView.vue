<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '../api/client'
import StatusBadge from '../components/StatusBadge.vue'

const POLL_INTERVAL_MS = 5000

const caseInfo = ref(null)
const loading = ref(true)
const error = ref('')
const noCase = ref(false)

let timer = null

function isTerminal(status) {
  return status === 'approved' || status === 'rejected'
}

function stopPolling() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

async function load() {
  try {
    caseInfo.value = await api.get('/api/verification-cases/me')
    noCase.value = false
    if (isTerminal(caseInfo.value.status)) {
      stopPolling()
    }
  } catch (err) {
    if (err.status === 404) {
      noCase.value = true
      stopPolling()
    } else {
      error.value = err.message
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await load()
  if (!noCase.value && caseInfo.value && !isTerminal(caseInfo.value.status)) {
    timer = setInterval(load, POLL_INTERVAL_MS)
  }
})

onUnmounted(stopPolling)
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
