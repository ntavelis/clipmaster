<script setup>
import { onMounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { GetTheme } from '../wailsjs/go/app/App'
import { useThemeStore } from './stores/theme'
import ClipboardHistory from './components/ClipboardHistory.vue'

const themeStore = useThemeStore()

onMounted(async () => {
  EventsOn('theme:loaded', themeStore.applyColors)

  const colors = await GetTheme()
  if (colors && colors.background) {
    themeStore.applyColors(colors)
  }
})
</script>

<template>
  <div class="min-h-screen bg-background text-foreground font-mono">
    <ClipboardHistory />
  </div>
</template>
