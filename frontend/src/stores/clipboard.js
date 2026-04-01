import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetHistory, CopyItem } from '../../wailsjs/go/app/App'
import { useNavigation } from './navigation'

export const useClipboardStore = defineStore('clipboard', () => {
  const items = ref([])
  const lastCopiedId = ref(null)
  const showShortcuts = ref(false)
  const activeTab = ref('local')

  const nav = useNavigation(() => items.value)

  async function fetchHistory() {
    items.value = await GetHistory()
  }

  async function copyItem(id) {
    lastCopiedId.value = id
    await CopyItem(id)
    await fetchHistory()
    setTimeout(() => { lastCopiedId.value = null }, 1000)
  }

  return { items, lastCopiedId, showShortcuts, activeTab, ...nav, fetchHistory, copyItem }
})
