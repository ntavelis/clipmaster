<script setup>
import { computed, onMounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { GetTheme } from '../wailsjs/go/app/App'
import { useThemeStore } from './stores/theme'
import { useClipboardStore } from './stores/clipboard'
import ClipboardHistory from './components/ClipboardHistory.vue'

const themeStore = useThemeStore()
const clipboard = useClipboardStore()
const showToast = computed(() => clipboard.lastCopiedId !== null)

const shortcuts = [
  { keys: 'Up / Down', action: 'Navigate items' },
  { keys: 'Enter', action: 'Copy selected item' },
  { keys: 'Space', action: 'Expand / collapse selected' },
  { keys: 'Escape', action: 'Collapse all / clear selection' },
  { keys: 'Ctrl+1..9', action: 'Quick copy Nth item' },
  { keys: 'Ctrl+K', action: 'Toggle this panel' },
]

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

    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-300"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="showToast"
        class="fixed bottom-4 left-1/2 -translate-x-1/2 rounded bg-color2/90 px-3 py-1.5 text-xs font-medium text-background"
      >
        Copied to clipboard
      </div>
    </Transition>

    <Transition
      enter-active-class="transition-opacity duration-150"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="clipboard.showShortcuts"
        class="fixed inset-0 z-50 flex items-center justify-center bg-background/80"
        @click.self="clipboard.showShortcuts = false"
      >
        <div class="w-72 rounded-lg border border-color8 bg-color0 p-4">
          <h2 class="text-sm font-semibold text-accent mb-3">Keyboard Shortcuts</h2>
          <div class="space-y-2">
            <div v-for="s in shortcuts" :key="s.keys" class="flex items-center justify-between">
              <span class="text-xs text-foreground">{{ s.action }}</span>
              <kbd class="rounded bg-color8 px-1.5 py-0.5 text-[10px] text-color7">{{ s.keys }}</kbd>
            </div>
          </div>
          <p class="mt-3 text-center text-[10px] text-color7">Press Ctrl+K or click outside to close</p>
        </div>
      </div>
    </Transition>
  </div>
</template>
