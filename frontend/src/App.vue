<script setup lang="ts">
import * as appSettingStorage from '@/wailsjs/go/storage/AppSettingStorage'
import * as driverGroupStorage from '@/wailsjs/go/storage/DriverGroupStorage'
import * as ruleSetStorage from '@/wailsjs/go/storage/RuleSetStorage'
import { CheckForUpdates } from '@/wailsjs/go/update/Updater'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RouteLocationRaw } from 'vue-router'
import { decodeError } from '@/utils/index'

const { t, locale } = useI18n()

const toast = useToast()

const [settingsStore, groupStore, matchStore, unsavedFormStore] = [
  useAppSettingStore(),
  useDriverGroupStore(),
  useMatchRuleStore(),
  useUnsavedFormStore()
]

const initilisating = ref(true)

const hasUpdate = ref(false)

Promise.all([
  driverGroupStorage
    .All()
    .then(gs => (groupStore.groups = gs))
    .catch(err => toast.add({ title: decodeError(err, t), color: 'error' })),
  appSettingStorage
    .All()
    .then(s => {
      settingsStore.settings = s
      locale.value = s.language
    })
    .catch(err => toast.add({ title: decodeError(err, t), color: 'error' })),
  ruleSetStorage
    .All()
    .then(rs => {
      matchStore.ruleSets = rs
    })
    .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
])
  .then(() => {
    setTimeout(() => {
      if (settingsStore.settings.auto_check_update) {
        return CheckForUpdates(settingsStore.settings.allow_pre_release).then(result => {
          hasUpdate.value = result.hasUpdate
          if (result.hasUpdate) {
            toast.add({ title: t('msgUpdateAvailable'), color: 'info', duration: 1000 })
          }
        })
      }
    }, 1000)
  })
  .finally(() => (initilisating.value = false))

const routes: Array<{ to: RouteLocationRaw; icon: string; title: string }> = [
  { to: '/', icon: 'mdi:home', title: 'titleHome' },
  { to: '/drivers', icon: 'mdi:package-variant-closed', title: 'fieldInstallOption' },
  { to: '/match-rules', icon: 'mdi:filter-variant', title: 'titleMatchRule' },
  { to: '/system-utilities', icon: 'mdi:tools', title: 'titleSetupTools' },
  { to: '/porter', icon: 'mdi:import-export', title: 'titleImportExport' },
  { to: '/settings', icon: 'mdi:cog', title: 'titleSettings' },
  { to: '/app-info', icon: 'mdi:information', title: 'titleAbout' }
]
</script>

<template>
  <UApp
    :toaster="{
      position: 'top-right',
      duration: 1500
    }"
  >
    <Transition name="fade" mode="out-in">
      <template v-if="!initilisating">
        <div class="flex h-screen w-screen">
          <aside class="w-12">
            <div class="flex h-full justify-center bg-gray-50">
              <ul class="mt-6 space-y-3 font-medium">
                <li v-for="link in routes" :key="link.to as string">
                  <RouterLink
                    :to="link.to"
                    class="flex rounded-lg p-2 hover:bg-gray-200 focus-visible:outline-2 focus-visible:outline-apple-green-600"
                    active-class="text-apple-green-800 bg-powder-blue-100"
                    :title="t(link.title)"
                    draggable="false"
                  >
                    <div class="relative">
                      <UBadge
                        v-if="link.to == '/app-info' && hasUpdate"
                        size="xs"
                        color="primary"
                        class="absolute -top-1 -right-1 h-2 w-2 p-0"
                      />

                      <Icon :icon="link.icon" />
                    </div>
                  </RouterLink>
                </li>
              </ul>
            </div>
          </aside>

          <main class="flex-1 overflow-hidden p-3">
            <RouterView></RouterView>
          </main>
        </div>
      </template>

      <template v-else>
        <div class="flex h-screen w-screen items-center justify-center">
          <div class="flex items-center gap-2">
            <Icon icon="mdi:loading" class="animate-spin text-5xl text-powder-blue-300" />
          </div>
        </div>
      </template>
    </Transition>
  </UApp>

  <UnsaveConfirmModal
    v-model:open="unsavedFormStore.show"
    :callback="unsavedFormStore.confirmLeave"
  />

  <LoadingOverlay />
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.5s ease-in;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
