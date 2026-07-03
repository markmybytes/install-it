import type { App, Directive } from 'vue'
import type { Router } from 'vue-router'

/**
 * Scroll restoration plugin.
 *
 * A self-contained Vue plugin that:
 *   1. Registers the `v-scroll-restore` directive (tag a scrollable element
 *      with a unique app-wide ID).
 *   2. Hooks the router's `beforeResolve` to snapshot each tagged element's
 *      scroll position into `history.state` on forward navigations only.
 *   3. The directive restores from `history.state` on mount.
 *
 * Restore semantics: on a back/forward navigation (`popstate`), the snapshot
 * is skipped so the destination's saved positions survive in `history.state`.
 * On a forward navigation, the new page's elements start at scrollTop=0 unless
 * a position was previously saved under their ID (e.g. after a round-trip
 * through another route).
 *
 * Tag IDs MUST be unique across the app — positions are keyed by the string
 * passed to `v-scroll-restore="'…'"`.
 *
 * Usage:
 *   // main.ts
 *   app.use(scrollRestore, { router })
 *
 *   // component
 *   <div v-scroll-restore="'app-sidebar'" class="overflow-y-auto h-screen">
 */

const SCROLL_STATE_KEY = '__scrollPositions'

type ScrollPositions = Record<string, { top: number; left: number }>

// Exported so the directive can be unit-tested in isolation.
export const vScrollRestore: Directive<HTMLElement, string> = {
  mounted(el, binding) {
    const id = binding.value
    if (!id) {
      console.warn('[v-scroll-restore] requires a unique string ID as its value.')
      return
    }

    // Tag the element so the snapshot sweep in `install` can find it
    // without depending on a CSS class (which would tie identity to styling).
    el.setAttribute('data-scroll-id', id)

    const state = history.state as Record<string, unknown> | null
    const positions = state?.[SCROLL_STATE_KEY] as ScrollPositions | undefined

    if (positions?.[id]) {
      el.scrollTop = positions[id].top
      el.scrollLeft = positions[id].left
    }
    // No saved position for this id → leave at default. A forward navigation
    // starts the new page at scrollTop=0, which is what we want.
  }
}

let installed = false

export default {
  install(app: App, options: { router: Router }): void {
    if (installed) return
    installed = true

    app.directive('scroll-restore', vScrollRestore)

    // `popstate` fires on back/forward navigation. Vue Router's `beforeResolve`
    // also fires for those, but we must NOT overwrite the destination route's
    // saved scroll positions when the user is popping the history stack.
    let isBackNav = false
    window.addEventListener('popstate', () => {
      isBackNav = true
    })

    options.router.beforeResolve((to, from) => {
      if (isBackNav) {
        isBackNav = false
        return
      }

      // Initial nav has no `from`. Same-fullPath navs (e.g. query-only changes)
      // shouldn't snapshot — they re-fire `beforeResolve` without an actual leave.
      if (!from || to.fullPath === from.fullPath) return

      // Carry forward positions for elements not currently in the DOM (e.g.
      // off-screen panes, modal content) so a later mount can still restore them.
      const positions: ScrollPositions = {
        ...((history.state as Record<string, unknown> | null)?.[SCROLL_STATE_KEY] as
          | ScrollPositions
          | undefined)
      }

      document.querySelectorAll('[data-scroll-id]').forEach(el => {
        const id = el.getAttribute('data-scroll-id')
        if (id) positions[id] = { top: el.scrollTop, left: el.scrollLeft }
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
