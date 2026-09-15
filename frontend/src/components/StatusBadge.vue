<script setup>
defineProps({
  status: { type: String, required: true },
})
</script>

<template>
  <span class="status-badge" :data-status="status">{{ status.replace('_', ' ') }}</span>
</template>

<style scoped>
.status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  text-transform: capitalize;
  background: var(--border);
  color: var(--text);
}

.status-badge[data-status='submitted'] {
  background: #ddf0ff;
  color: #0969da;
}

.status-badge[data-status='in_review'] {
  background: #fff3cd;
  color: #9a6700;
}

.status-badge[data-status='approved'] {
  background: #d9f5df;
  color: #1a7f37;
}

.status-badge[data-status='rejected'] {
  background: #ffe0e0;
  color: #cf222e;
}

@media (prefers-color-scheme: dark) {
  .status-badge[data-status='submitted'] {
    background: #143a5c;
    color: #4493f8;
  }

  .status-badge[data-status='in_review'] {
    background: #4d3b00;
    color: #d4a72c;
  }

  .status-badge[data-status='approved'] {
    background: #1a3d24;
    color: #3fb950;
  }

  .status-badge[data-status='rejected'] {
    background: #4c1414;
    color: #ff7b72;
  }
}
</style>
