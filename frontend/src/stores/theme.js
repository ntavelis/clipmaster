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
    root.style.setProperty('--color-black',          colors.color0)
    root.style.setProperty('--color-red',            colors.color1)
    root.style.setProperty('--color-green',          colors.color2)
    root.style.setProperty('--color-yellow',         colors.color3)
    root.style.setProperty('--color-blue',           colors.color4)
    root.style.setProperty('--color-magenta',        colors.color5)
    root.style.setProperty('--color-cyan',           colors.color6)
    root.style.setProperty('--color-white',          colors.color7)
    root.style.setProperty('--color-bright-black',   colors.color8)
    root.style.setProperty('--color-bright-red',     colors.color9)
    root.style.setProperty('--color-bright-green',   colors.color10)
    root.style.setProperty('--color-bright-yellow',  colors.color11)
    root.style.setProperty('--color-bright-blue',    colors.color12)
    root.style.setProperty('--color-bright-magenta', colors.color13)
    root.style.setProperty('--color-bright-cyan',    colors.color14)
    root.style.setProperty('--color-bright-white',   colors.color15)
  }

  return { themeName, applyColors }
})
