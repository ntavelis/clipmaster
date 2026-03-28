import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetHistory, CopyItem } from '../../wailsjs/go/app/App'

export const useClipboardStore = defineStore('clipboard', () => {
  const items = ref([])

  // fetchHistory loads the current clipboard history from Go.
  async function fetchHistory() {
    items.value = await GetHistory()
  }

  // prependEntry inserts a new entry at the top of the list, received via Wails event.
  function prependEntry(entry) {
    items.value.unshift(entry)
  }

  // copyItem writes the entry back to the system clipboard then refreshes.
  async function copyItem(id) {
    await CopyItem(id)
    await fetchHistory()
  }

  return { items, fetchHistory, prependEntry, copyItem }
})
