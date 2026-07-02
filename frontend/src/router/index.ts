import { createRouter, createWebHashHistory } from 'vue-router'
import { handleHotUpdate, routes } from 'vue-router/auto-routes'

const SCROLL_STATE_KEY = '__scrollPositions'
const SCROLL_SELECTOR = '.scrollable'

type ScrollPositions = Record<string, { top: number; left: number }>

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes
})

router.beforeResolve(to => {
  if (history.state?.current === to.fullPath) {
    return
  }

  const positions: ScrollPositions = {}
  document.querySelectorAll(SCROLL_SELECTOR).forEach((el, i) => {
    positions[i] = { top: el.scrollTop, left: el.scrollLeft }
  })
  history.replaceState({ ...history.state, [SCROLL_STATE_KEY]: positions }, '')
})

router.afterEach(() => {
  const positions = history.state?.[SCROLL_STATE_KEY] as ScrollPositions | undefined
  const isHistory = positions !== undefined

  requestAnimationFrame(() => {
    document.querySelectorAll(SCROLL_SELECTOR).forEach((el, i) => {
      if (isHistory && positions[i]) {
        el.scrollTop = positions[i].top
        el.scrollLeft = positions[i].left
      } else {
        el.scrollTop = 0
        el.scrollLeft = 0
      }
    })
  })
})

if (import.meta.hot) {
  handleHotUpdate(router)
}

export default router
