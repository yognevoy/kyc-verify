<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client'

const route = useRoute()
const router = useRouter()

const caseInfo = ref(null)
const documents = ref([])
const loading = ref(true)
const error = ref('')
const comment = ref('')
const submitting = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [c, docs] = await Promise.all([
      api.get(`/api/verification-cases/${route.params.id}`),
      api.get(`/api/verification-cases/${route.params.id}/documents`),
    ])
    caseInfo.value = c
    documents.value = docs
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(load)

async function decide(action) {
  submitting.value = true
  error.value = ''
  try {
    await api.post(`/api/verification-cases/${route.params.id}/${action}`, { comment: comment.value })
    router.push('/admin')
  } catch (err) {
    error.value = err.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main>
    <p><RouterLink to="/admin">&larr; Back to queue</RouterLink></p>
    <p v-if="loading">Loading...</p>
    <template v-else-if="caseInfo">
      <h1>Case {{ caseInfo.id }}</h1>
      <p>Status: {{ caseInfo.status }}</p>
      <p v-if="caseInfo.applicant_risk_level">Applicant risk: {{ caseInfo.applicant_risk_level }}</p>
      <p v-if="caseInfo.applicant_prior_rejections != null">Prior rejections: {{ caseInfo.applicant_prior_rejections }}</p>

      <h2>Documents</h2>
      <ul>
        <li v-for="doc in documents" :key="doc.id">
          {{ doc.type }} - {{ new Date(doc.uploaded_at).toLocaleString() }}
        </li>
      </ul>

      <h2>Decision</h2>
      <label>
        Comment
        <textarea v-model="comment"></textarea>
      </label>
      <div class="actions">
        <button :disabled="submitting" @click="decide('approve')">Approve</button>
        <button :disabled="submitting" @click="decide('reject')">Reject</button>
      </div>
    </template>
    <p v-if="error" role="alert">{{ error }}</p>
  </main>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}
</style>
