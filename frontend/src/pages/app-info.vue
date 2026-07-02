<script setup lang="ts">
import { RunAndOutput } from '@/wailsjs/go/execute/CommandExecutor'
import {
  AppDriverPath,
  AppVersion,
  PathExists,
  WebView2Path,
  WebView2Version
} from '@/wailsjs/go/main/App'
import { CheckForUpdates } from '@/wailsjs/go/update/Updater'
import { BrowserOpenURL, Environment } from '@/wailsjs/runtime/runtime'
import { Icon } from '@iconify/vue'
import { onBeforeMount, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const toast = useToast()

const $loading = useLoading()

const modal = useTemplateRef('modal')

const info = ref({
  app: {
    version: 'na',
    buildType: 'na',
    pathDriver: 'na'
  },
  webview: {
    version: 'na',
    location: 'na'
  }
})

const onCheck = ref(false)
const preferBundled = ref(false)

const appSettingStore = useAppSettingStore()

onBeforeMount(() => {
  Promise.allSettled([
    AppVersion(),
    Environment(),
    AppDriverPath(),
    WebView2Version(),
    WebView2Path(),
    PathExists('internals')
  ]).then(([ver, env, pdri, vwv2, pwv2, hasInternals]) => {
    if (ver.status !== 'rejected') {
      info.value.app.version = ver.value
    }

    if (env.status !== 'rejected') {
      info.value.app.buildType = env.value.buildType
    }

    if (pdri.status !== 'rejected') {
      info.value.app.pathDriver = pdri.value
    }

    if (vwv2.status !== 'rejected') {
      info.value.webview.version = vwv2.value
    }

    if (pwv2.status !== 'rejected') {
      info.value.webview.location = pwv2.value
    }

    if (hasInternals.status !== 'rejected') {
      preferBundled.value = hasInternals.value
    }
  })
})

function buildTypeKey(suffix: string): string {
  return `buildType${suffix.charAt(0).toUpperCase() + suffix.slice(1)}`
}

function checkUpdate() {
  if (Object.values(info.value.app).some(v => v === 'na')) {
    toast.add({ title: t('toastCheckUpdateFailed'), color: 'error' })
    return
  }

  onCheck.value = true
  $loading.show()

  CheckForUpdates(preferBundled.value, appSettingStore.settings.allow_pre_release)
    .then(result => {
      if (result.hasUpdate) {
        modal.value?.show(result)
      } else {
        toast.add({ title: t('toastNoUpdate'), color: 'info' })
      }
    })
    .catch(reason => {
      toast.add({ title: reason, color: 'error' })
    })
    .finally(() => {
      onCheck.value = false
      $loading.hide()
    })
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-6 overflow-y-auto p-2">
    <PageHeader variant="normal" :title="`${$t('titleAbout')} install-it`" />

    <div class="flex flex-col gap-y-6">
      <div>
        <h2 class="mb-2 font-bold">{{ $t('labelThisSoftware') }}</h2>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('version') }}</div>

          <div class="col-span-5 flex items-center gap-x-5">
            <p>
              {{ info.app.version }}
            </p>

            <UButton
              color="neutral"
              variant="outline"
              size="xs"
              :disabled="onCheck"
              @click="checkUpdate()"
            >
              {{ $t('actionCheckUpdate') }}
            </UButton>
          </div>
        </div>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('labelBuildType') }}</div>

          <div class="col-span-5">
            {{ info.app.buildType === 'na' ? $t('labelNA') : $t(buildTypeKey(info.app.buildType)) }}
          </div>
        </div>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('labelDriverPath') }}</div>

          <div class="col-span-5 break-all">
            {{ info.app.pathDriver }}

            <UButton
              type="button"
              color="neutral"
              variant="link"
              size="md"
              @click="RunAndOutput('cmd', ['/c', `explorer.exe ${info.app.pathDriver}`], true)"
            >
              <Icon icon="mdi:open-in-new" />
            </UButton>
          </div>
        </div>
      </div>

      <div>
        <h2 class="mb-2 font-bold">Microsoft Edge WebView2</h2>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('version') }}</div>

          <div class="col-span-5">{{ info.webview.version }}</div>
        </div>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('path') }}</div>

          <div class="col-span-5">
            {{ info.webview.location || $t('msgUsingBuiltInWebView2') }}
          </div>
        </div>
      </div>

      <div>
        <h2 class="mb-2 font-bold">{{ $t('labelDevelopment') }}</h2>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('labelSourceCode') }}</div>

          <div class="col-span-5">
            <a
              href="https://github.com/install-it/install-it"
              class="text-sky-700 underline"
              @click.prevent="event => BrowserOpenURL((event.target as HTMLAnchorElement).href)"
            >
              https://github.com/install-it/install-it
            </a>
          </div>
        </div>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('labelReportBug') }}</div>

          <div class="col-span-5">
            <a
              href="https://github.com/install-it/install-it/issues"
              class="text-sky-700 underline"
              @click.prevent="event => BrowserOpenURL((event.target as HTMLAnchorElement).href)"
            >
              https://github.com/install-it/install-it/issues
            </a>
          </div>
        </div>

        <div class="grid grid-cols-7 gap-4">
          <div class="col-span-2">{{ $t('license') }}</div>

          <div class="col-span-5">
            <div class="flex">
              <p class="inline font-mono">GNU General Public License v2.0</p>

              <UButton
                type="button"
                color="neutral"
                variant="link"
                size="md"
                @click="
                  BrowserOpenURL('https://github.com/install-it/install-it/blob/main/LICENSE')
                "
              >
                <Icon icon="mdi:open-in-new" />
              </UButton>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <UpdateModal ref="modal" :current-version="info.app.version"></UpdateModal>
</template>
