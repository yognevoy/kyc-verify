<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/client'
import DocumentDropzone from '../components/DocumentDropzone.vue'

const router = useRouter()

const profile = ref(null)
const profileLoading = ref(true)
const profileForm = reactive({ full_name: '', birth_date: '', country: '' })
const profileError = ref('')
const creatingProfile = ref(false)

const documents = reactive({ passport: null, selfie: null, proof_of_address: null })
const uploadErrors = reactive({ passport: '', selfie: '', proof_of_address: '' })
const uploading = reactive({ passport: false, selfie: false, proof_of_address: false })

const submitError = ref('')
const submitting = ref(false)

const allUploaded = computed(() => documents.passport && documents.selfie && documents.proof_of_address)

onMounted(async () => {
  try {
    profile.value = await api.get('/api/applicants/me')
    documents.passport = null
    documents.selfie = null
    documents.proof_of_address = null
    const docs = await api.get('/api/documents/me')
    for (const doc of docs) {
      documents[doc.type] = doc
    }
  } catch (err) {
    if (err.status !== 404) {
      profileError.value = err.message
    }
  } finally {
    profileLoading.value = false
  }
})

async function createProfile() {
  creatingProfile.value = true
  profileError.value = ''
  try {
    profile.value = await api.post('/api/applicants', { ...profileForm })
  } catch (err) {
    profileError.value = err.message
  } finally {
    creatingProfile.value = false
  }
}

async function upload(type, file) {
  uploading[type] = true
  uploadErrors[type] = ''
  try {
    const form = new FormData()
    form.append('type', type)
    form.append('file', file)
    documents[type] = await api.post('/api/documents', form)
  } catch (err) {
    uploadErrors[type] = err.message
  } finally {
    uploading[type] = false
  }
}

async function submit() {
  submitting.value = true
  submitError.value = ''
  try {
    await api.post('/api/verification-cases')
    router.push('/status')
  } catch (err) {
    if (err.status === 409) {
      router.push('/status')
      return
    }
    submitError.value = err.message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main>
    <p v-if="profileLoading">Loading...</p>
    <template v-else-if="!profile">
      <h1>Create your profile</h1>
      <form @submit.prevent="createProfile">
        <label>
          Full name
          <input v-model="profileForm.full_name" required />
        </label>
        <label>
          Birth date
          <input v-model="profileForm.birth_date" type="date" required />
        </label>
        <label>
          Country (ISO code)
          <input v-model="profileForm.country" maxlength="2" required />
        </label>
        <button type="submit" :disabled="creatingProfile">{{ creatingProfile ? 'Saving...' : 'Save' }}</button>
        <p v-if="profileError" role="alert">{{ profileError }}</p>
      </form>
    </template>
    <template v-else>
      <h1>Upload documents</h1>
      <div class="dropzones">
        <div>
          <DocumentDropzone
            label="Passport"
            :uploaded="!!documents.passport"
            :disabled="uploading.passport"
            @select="(file) => upload('passport', file)"
          />
          <p v-if="uploadErrors.passport" role="alert">{{ uploadErrors.passport }}</p>
        </div>
        <div>
          <DocumentDropzone
            label="Selfie"
            :uploaded="!!documents.selfie"
            :disabled="uploading.selfie"
            @select="(file) => upload('selfie', file)"
          />
          <p v-if="uploadErrors.selfie" role="alert">{{ uploadErrors.selfie }}</p>
        </div>
        <div>
          <DocumentDropzone
            label="Proof of address"
            :uploaded="!!documents.proof_of_address"
            :disabled="uploading.proof_of_address"
            @select="(file) => upload('proof_of_address', file)"
          />
          <p v-if="uploadErrors.proof_of_address" role="alert">{{ uploadErrors.proof_of_address }}</p>
        </div>
      </div>
      <button :disabled="!allUploaded || submitting" @click="submit">
        {{ submitting ? 'Submitting...' : 'Submit for verification' }}
      </button>
      <p v-if="submitError" role="alert">{{ submitError }}</p>
    </template>
  </main>
</template>

<style scoped>
.dropzones {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin: 24px 0;
}
</style>
