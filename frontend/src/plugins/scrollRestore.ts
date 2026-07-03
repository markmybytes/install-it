import type { App, Directive } from 'vue'
import type { Router } from 'vue-router'

const SCROLL_STATE_KEY = '__scrollPositions'

type ScrollPositions = Record<string, { top: number; left: number }>

// Tracks active containers to avoid expensive document.querySelectorAll scans on route changes.
const activeElements = new Set<HTMLElement>()

function tryRestore(el: HTMLElement, id: string) {
  if (el.dataset.restored === 'true') return

  const state = history.state as Record<string, unknown> | null
  const positions = state?.[SCROLL_STATE_KEY] as ScrollPositions | undefined
  const saved = positions?.[id]

  if (!saved) {
    el.dataset.restored = 'true'
    return
  }

  const isScrollable = el.scrollHeight > el.clientHeight || el.scrollWidth > el.clientWidth
  const isTargetZero = saved.top === 0 && saved.left === 0

  // Apply immediately if layout heights are computed, or if target is the top baseline.
  if (isScrollable || isTargetZero) {
    el.scrollTop = saved.top
    el.scrollLeft = saved.left
    el.dataset.restored = 'true'
  }
}

export const vScrollRestore: Directive<HTMLElement, string> = {
  mounted(el, binding) {
    const id = binding.value
    if (!id) {
      console.warn('[v-scroll-restore] requires a unique string ID as its value.')
      return
    }

    el.setAttribute('data-scroll-id', id)
    el.dataset.restored = 'false'
    activeElements.add(el)

    tryRestore(el, id)
  },

  updated(el, binding) {
    tryRestore(el, binding.value)
  },

  unmounted(el) {
    activeElements.delete(el)
  }
}

let installed = false

export default {
  install(app: App, options: { router: Router }): void {
    if (installed) return
    installed = true

    app.directive('scroll-restore', vScrollRestore)

    options.router.beforeResolve((to, from) => {
      if (!from || to.fullPath === from.fullPath) return

      // Unlock tracking states across active elements so components reused on query
      // changes can re-evaluate layout heights and apply new scroll constraints.
      activeElements.forEach(el => {
        el.dataset.restored = 'false'
      })

      // Skip snapshot on popstate (back/forward) to preserve destination coordinate maps.
      const state = history.state as Record<string, unknown> | null
      const currentPath = state?.['current']
      if (typeof currentPath !== 'string' || currentPath !== from.fullPath) return

      const positions: ScrollPositions = {
        ...(state?.[SCROLL_STATE_KEY] as ScrollPositions | undefined)
      }

      activeElements.forEach(el => {
        const id = el.getAttribute('data-scroll-id')
        if (id) {
          positions[id] = { top: el.scrollTop, left: el.scrollLeft }
        }
      })

      history.replaceState({ ...history.state, [SCROLL_STATE_KEY]: positions }, '')
    })
  }
}

declare module 'vue' {
  export interface GlobalDirectives {
    vScrollRestore: typeof vScrollRestore
  }
}
