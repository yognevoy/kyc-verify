<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '../../api/client'

const items = ref([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    items.value = await api.get('/api/verification-cases')
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main>
    <h1>Review queue</h1>
    <p v-if="loading">Loading...</p>
    <p v-else-if="error" role="alert">{{ error }}</p>
    <p v-else-if="items.length === 0">No pending cases.</p>
    <table v-else>
      <thead>
        <tr>
          <th>Applicant</th>
          <th>Risk</th>
          <th>Prior rejections</th>
          <th>Status</th>
          <th>Submitted</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.case_id">
          <td><RouterLink :to="`/admin/cases/${item.case_id}`">{{ item.full_name }}</RouterLink></td>
          <td>{{ item.risk_level }}</td>
          <td>{{ item.prior_rejections }}</td>
          <td>{{ item.status }}</td>
          <td>{{ new Date(item.created_at).toLocaleString() }}</td>
        </tr>
      </tbody>
    </table>
  </main>
</template>
