import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetRemoteClipboards, CopyRemoteItem } from '../../wailsjs/go/app/App'

export const useRemoteStore = defineStore('remote', () => {
  const peers = ref([])
  const lastCopiedContent = ref(null)

  async function fetchRemote() {
    const result = await GetRemoteClipboards()
    peers.value = result || []
  }

  async function copyContent(content) {
    lastCopiedContent.value = content
    await CopyRemoteItem(content)
    setTimeout(() => {
      lastCopiedContent.value = null
    }, 1000)
  }

  return { peers, lastCopiedContent, fetchRemote, copyContent }
})
