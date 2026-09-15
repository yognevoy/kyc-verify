<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { api } from '../api/client'
import DocumentDropzone from '../components/DocumentDropzone.vue'

const PENDING_STATUSES = ['submitted', 'in_review']

const router = useRouter()

const profile = ref(null)
const profileLoading = ref(true)
const profileForm = reactive({ full_name: '', birth_date: '', country: '' })
const profileError = ref('')
const savingProfile = ref(false)
const step = ref('profile')

const documents = reactive({ passport: null, selfie: null, proof_of_address: null })
const uploadErrors = reactive({ passport: '', selfie: '', proof_of_address: '' })
const uploading = reactive({ passport: false, selfie: false, proof_of_address: false })

const submitError = ref('')
const submitting = ref(false)

const pendingCaseStatus = ref(null)

const allUploaded = computed(() => documents.passport && documents.selfie && documents.proof_of_address)
const pendingCaseLabel = computed(() => pendingCaseStatus.value?.replace('_', ' '))

onMounted(async () => {
  try {
    profile.value = await api.get('/api/applicants/me')
    fillProfileForm(profile.value)
    step.value = 'documents'
  } catch (err) {
    if (err.status !== 404) {
      profileError.value = err.message
    }
    profileLoading.value = false
    return
  }

  try {
    const docs = await api.get('/api/documents/me')
    for (const doc of docs) {
      documents[doc.type] = doc
    }
  } catch (err) {
    profileError.value = err.message
  }

  try {
    const latestCase = await api.get('/api/verification-cases/me')
    if (PENDING_STATUSES.includes(latestCase.status)) {
      pendingCaseStatus.value = latestCase.status
    }
  } catch (err) {
    if (err.status !== 404) {
      profileError.value = err.message
    }
  }

  profileLoading.value = false
})

function fillProfileForm(source) {
  profileForm.full_name = source.full_name
  profileForm.birth_date = source.birth_date
  profileForm.country = source.country
}

function goToStep(target) {
  if (target === 'documents' && !profile.value) {
    return
  }
  if (target === 'profile' && profile.value) {
    fillProfileForm(profile.value)
  }
  step.value = target
}

async function saveProfile() {
  savingProfile.value = true
  profileError.value = ''
  try {
    profile.value = profile.value
      ? await api.put('/api/applicants/me', { ...profileForm })
      : await api.post('/api/applicants', { ...profileForm })
    step.value = 'documents'
  } catch (err) {
    profileError.value = err.message
  } finally {
    savingProfile.value = false
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
    <template v-else-if="pendingCaseStatus">
      <h1>Verification in progress</h1>
      <p>Your verification case is already <strong>{{ pendingCaseLabel }}</strong> - you can't submit new documents until it's decided.</p>
      <p><RouterLink to="/status">View status</RouterLink></p>
    </template>
    <template v-else>
      <ol class="steps">
        <li :class="{ active: step === 'profile', done: !!profile }" @click="goToStep('profile')">1. Profile</li>
        <li :class="{ active: step === 'documents', clickable: !!profile }" @click="goToStep('documents')">2. Documents</li>
      </ol>
      <template v-if="step === 'profile'">
        <h1>{{ profile ? 'Edit your profile' : 'Create your profile' }}</h1>
        <form @submit.prevent="saveProfile">
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
          <button type="submit" :disabled="savingProfile">{{ savingProfile ? 'Saving...' : 'Save' }}</button>
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
    </template>
  </main>
</template>

<style scoped>
.steps {
  display: flex;
  gap: 24px;
  margin: 0 0 24px;
  padding: 0;
  list-style: none;
}

.steps li {
  padding-bottom: 8px;
  color: var(--text-muted);
  font-weight: 600;
  border-bottom: 2px solid transparent;
}

.steps li.done,
.steps li.clickable {
  color: var(--text);
  cursor: pointer;
}

.steps li.active {
  color: var(--text);
  border-bottom-color: var(--accent);
}

.dropzones {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin: 24px 0;
}
</style>
