<script setup>
import { ref } from 'vue'

defineProps({
  label: { type: String, required: true },
  uploaded: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['select'])

const input = ref(null)
const dragging = ref(false)

function open() {
  input.value.click()
}

function onDrop(e) {
  dragging.value = false
  handleFiles(e.dataTransfer.files)
}

function onChange(e) {
  handleFiles(e.target.files)
}

function handleFiles(files) {
  const file = files?.[0]
  if (file) {
    emit('select', file)
  }
}
</script>

<template>
  <div
    class="dropzone"
    :class="{ dragging, uploaded, disabled }"
    @click="!disabled && open()"
    @dragover.prevent="!disabled && (dragging = true)"
    @dragleave.prevent="dragging = false"
    @drop.prevent="!disabled && onDrop($event)"
  >
    <p class="label">{{ label }}</p>
    <p class="hint">
      <template v-if="uploaded">Uploaded</template>
      <template v-else-if="disabled">Uploading...</template>
      <template v-else>Click or drop file</template>
    </p>
    <input ref="input" type="file" accept="image/*,application/pdf" hidden @change="onChange" />
  </div>
</template>

<style scoped>
.dropzone {
  border: 2px dashed var(--border);
  border-radius: 6px;
  padding: 24px 16px;
  text-align: center;
  cursor: pointer;
}

.dropzone.dragging {
  border-color: var(--accent);
}

.dropzone.uploaded {
  border-style: solid;
  border-color: #1a7f37;
}

.dropzone.disabled {
  cursor: default;
  opacity: 0.7;
}

.label {
  margin: 0 0 4px;
  font-weight: 600;
}

.hint {
  margin: 0;
  color: var(--text-muted);
  font-size: 14px;
}
</style>
