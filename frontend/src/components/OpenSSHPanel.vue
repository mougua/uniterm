<template>
  <section class="ssh-panel" :aria-label="t('openSSH.title')">
    <div class="source"><code>~/.ssh/config</code></div>
    <div class="toolbar">
      <el-input
        v-model="query"
        :prefix-icon="Search"
        :placeholder="t('openSSH.search')"
        :aria-label="t('openSSH.search')"
        size="small"
        clearable
        @keyup.enter="connectFromSearch"
      />
      <el-button
        :icon="RefreshCw"
        :title="t('openSSH.refresh')"
        :aria-label="t('openSSH.refresh')"
        :disabled="loading"
        size="small"
        text
        @click="load"
      />
    </div>
    <div class="hosts" v-loading="loading" :aria-busy="loading">
      <p v-if="error" class="host-message" role="alert">{{ error }}</p>
      <p v-else-if="!loading && matches.length === 0" class="host-message" role="status">
        {{ t(hosts.length ? 'openSSH.noMatches' : 'openSSH.empty') }}
      </p>
      <button v-for="host in matches" :key="host" class="host" :title="host" @click="emit('connect', host)">
        <SquareTerminal :size="16" aria-hidden="true" />
        <span>{{ host }}</span>
        <ArrowRight :size="15" aria-hidden="true" />
      </button>
    </div>
    <el-button v-if="target && matches.length === 0" class="connect-button" size="small" @click="connectFromSearch">
      {{ t('openSSH.connect') }}
    </el-button>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, RefreshCw, Search, SquareTerminal } from '@lucide/vue'
import { ListOpenSSHHosts } from '../../bindings/github.com/ys-ll/uniterm/app'
import { useI18n } from '../i18n'
import { isOpenSSHHost } from '../utils/openSSH'

const emit = defineEmits<{ connect: [host: string] }>()
const { t } = useI18n()
const hosts = ref<string[]>([])
const query = ref('')
const error = ref('')
const loading = ref(false)
const search = computed(() => query.value.trim())
const matches = computed(() => hosts.value.filter(host => host.toLowerCase().includes(search.value.toLowerCase())))
const target = computed(() => {
  if (!search.value) return ''
  const exact = matches.value.find(host => host.toLowerCase() === search.value.toLowerCase())
  if (exact) return exact
  if (matches.value.length === 1) return matches.value[0]
  return matches.value.length === 0 && isOpenSSHHost(search.value) ? search.value : ''
})

onMounted(load)

async function load() {
  if (loading.value) return
  loading.value = true
  hosts.value = []
  error.value = ''
  try {
    hosts.value = (await ListOpenSSHHosts()) ?? []
  } catch (cause) {
    error.value = t('openSSH.readError', { error: String(cause) })
  } finally {
    loading.value = false
  }
}

function connectFromSearch() {
  if (!loading.value && target.value) emit('connect', target.value)
}
</script>

<style scoped>
.ssh-panel {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 0 10px 10px;
}

.source {
  margin: 2px 2px 8px;
  color: var(--el-text-color-secondary);
}

.toolbar {
  display: flex;
  gap: 4px;
}

.toolbar :deep(.el-input) {
  flex: 1;
  min-width: 0;
}

.hosts {
  min-height: 0;
  overflow-y: auto;
  margin-top: 8px;
  font-size: 13px;
}

.hosts[aria-busy='true'] {
  min-height: 48px;
}

.connect-button {
  align-self: flex-start;
  margin: 4px 8px;
}

.host {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.host span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.host:hover,
.host:focus-visible {
  background: var(--el-fill-color);
  color: var(--el-color-primary);
}

.host-message {
  padding: 8px;
  overflow-wrap: anywhere;
  color: var(--el-text-color-secondary);
}
</style>
