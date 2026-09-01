<script setup lang="ts">
import { decodeError } from '@/utils/index'
import { TriggerNativeUpdate } from '@/wailsjs/go/update/Updater'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{ currentVersion: string }>()

const isOpen = ref(false)
const updateResult = ref<{
  hasUpdate: boolean
  latestVersion: string
  releaseNotes: string
  releaseAt: string
}>()
const parsedNotes = ref('')
const webviewVersion = ref(false)
const isUpdating = ref(false)

const releaseAt = ref('')

defineExpose({
  show: (result: typeof updateResult.value) => {
    updateResult.value = result
    isUpdating.value = false
    isOpen.value = true
  },
  hide: () => {
    isOpen.value = false
  }
})

const toast = useToast()
const { t } = useI18n()

const handleUpdate = () => {
  if (!updateResult.value?.latestVersion) {
    toast.add({ title: t('warnNoAssetUrl'), color: 'error' })
    return
  }

  isUpdating.value = true

  TriggerNativeUpdate(updateResult.value.latestVersion, webviewVersion.value).catch(err => {
    isUpdating.value = false
    toast.add({ title: decodeError(err, t), color: 'error' })
  })
}

watch(
  updateResult,
  async result => {
    if (!result) {
      parsedNotes.value = ''
      releaseAt.value = ''
      return
    }

    releaseAt.value = new Date(result.releaseAt).toLocaleDateString()

    if (result.releaseNotes) {
      const html = await marked.parse(result.releaseNotes)
      parsedNotes.value = DOMPurify.sanitize(html)
    } else {
      parsedNotes.value = ''
    }
  },
  { deep: true }
)
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="$t('titleUpdateInfo')"
    :dismissible="!isUpdating"
    :close="!isUpdating"
  >
    <template #body>
      <div
        v-if="isUpdating"
        class="flex min-h-75 flex-col items-center justify-center gap-y-4 px-4 py-12"
      >
        <UIcon name="i-lucide-loader-2" class="size-10 animate-spin text-primary" />

        <p class="max-w-xs text-center text-sm text-gray-500 dark:text-gray-400">
          {{ $t('msgDownloadingUpdater') }}
        </p>
      </div>

      <div v-else class="flex flex-col gap-y-3">
        <div class="flex grow flex-col gap-y-2">
          <div class="flex">
            <h1 class="min-w-34 font-medium">
              {{ $t('labelCurrentVersion') }}
            </h1>

            <p>{{ currentVersion }}</p>
          </div>

          <div class="flex">
            <h1 class="min-w-34 font-medium">
              {{ $t('labelLatestVersion') }}
            </h1>

            <p>
              {{ `${updateResult?.latestVersion} (${releaseAt})` }}
            </p>
          </div>

          <hr />

          <div class="flex grow flex-col">
            <h1 class="mb-1 min-w-32 font-medium">
              {{ $t('labelUpdateInfo') }}
            </h1>

            <!-- eslint-disable vue/no-v-html -->
            <div
              id="release-notes"
              class="rounded-lg border px-1"
              v-html="parsedNotes || `<i>${$t('msgNoUpdateInfo')}</i>`"
            ></div>
            <!-- eslint-enable vue/no-v-html -->
          </div>

          <hr />

          <div class="flex flex-col">
            <h1 class="font-medium">
              {{ $t('labelUpdateOptions') }}
            </h1>

            <label class="flex w-full cursor-pointer items-center select-none">
              <UCheckbox v-model="webviewVersion" name="webview_version" color="primary" />

              <span class="ms-1.5">{{ $t('labelDownloadWebView2') }}</span>
            </label>
          </div>
        </div>

        <UButton
          color="secondary"
          block
          class="justify-center"
          :loading="isUpdating"
          @click="handleUpdate"
        >
          {{ $t('labelUpdate') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<style scoped>
#release-notes * {
  all: revert;
}
</style>
