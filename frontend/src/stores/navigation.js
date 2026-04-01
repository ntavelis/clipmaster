import { ref } from 'vue'

export function useNavigation(getItems) {
  const selectedIndex = ref(-1)
  const keyboardActive = ref(false)
  const expandedIds = ref(new Set())

  function selectNext() {
    const items = getItems()
    if (items.length === 0) return
    keyboardActive.value = true
    selectedIndex.value = Math.min(selectedIndex.value + 1, items.length - 1)
  }

  function selectPrev() {
    const items = getItems()
    if (items.length === 0) return
    keyboardActive.value = true
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  }

  function clearSelection() {
    selectedIndex.value = -1
    keyboardActive.value = false
  }

  function deactivateKeyboard() {
    keyboardActive.value = false
  }

  function collapseAll() {
    expandedIds.value = new Set()
  }

  function toggleExpanded(id) {
    const s = new Set(expandedIds.value)
    if (s.has(id)) {
      s.delete(id)
    } else {
      s.add(id)
    }
    expandedIds.value = s
  }

  return { selectedIndex, keyboardActive, expandedIds, selectNext, selectPrev, clearSelection, deactivateKeyboard, collapseAll, toggleExpanded }
}
