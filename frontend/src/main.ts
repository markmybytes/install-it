import './assets/main.css'

import { i18n } from '@/i18n'
import { icons as lucideIcons } from '@iconify-json/lucide'
import { icons as mdiIcons } from '@iconify-json/mdi'
import { Icon, addCollection } from '@iconify/vue'
import ui from '@nuxt/ui/vue-plugin'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import scrollRestore from './plugins/scrollRestore'

// Register icon collections for offline support
// See: https://iconify.design/news/2025.html#why-was-it-removed
addCollection(mdiIcons)
addCollection(lucideIcons) // nuxt/ui uses lucide icons

const app = createApp(App)
  .use(router)
  .use(createPinia())
  .use(i18n)
  .use(ui)
  .use(scrollRestore, { router })
  .component('Icon', Icon)

app.config.globalProperties.$window = window

app.mount('#app')
