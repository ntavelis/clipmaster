import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const themeName = ref('tokyo-night')

  // applyColors maps ThemeColors from Go onto :root CSS custom properties.
  function applyColors(colors) {
    const root = document.documentElement
    root.style.setProperty('--color-background',     colors.background)
    root.style.setProperty('--color-foreground',     colors.foreground)
    root.style.setProperty('--color-accent',         colors.accent)
    root.style.setProperty('--color-cursor',         colors.cursor)
    root.style.setProperty('--color-selection-bg',   colors.selectionBackground)
    root.style.setProperty('--color-selection-fg',   colors.selectionForeground)
    root.style.setProperty('--color-color0',          colors.color0)
    root.style.setProperty('--color-color1',          colors.color1)
    root.style.setProperty('--color-color2',          colors.color2)
    root.style.setProperty('--color-color3',          colors.color3)
    root.style.setProperty('--color-color4',          colors.color4)
    root.style.setProperty('--color-color5',          colors.color5)
    root.style.setProperty('--color-color6',          colors.color6)
    root.style.setProperty('--color-color7',          colors.color7)
    root.style.setProperty('--color-color8',          colors.color8)
    root.style.setProperty('--color-color9',          colors.color9)
    root.style.setProperty('--color-color10',         colors.color10)
    root.style.setProperty('--color-color11',         colors.color11)
    root.style.setProperty('--color-color12',         colors.color12)
    root.style.setProperty('--color-color13',         colors.color13)
    root.style.setProperty('--color-color14',         colors.color14)
    root.style.setProperty('--color-color15',         colors.color15)
  }

  return { themeName, applyColors }
})
