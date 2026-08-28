<script setup lang="ts">
import { porter } from '@/wailsjs/go/models'
import * as programPorter from '@/wailsjs/go/porter/Porter'
import * as runtime from '@/wailsjs/runtime'
import { decodeError } from '@/utils/index'
import { computed, nextTick, onUnmounted, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'

const isOpen = ref(false)

const mode = ref<'export' | 'import' | 'download'>('export')

const snapshot = ref<porter.JobSnapshot | null>(null)

const steps = computed(() => {
  switch (mode.value) {
    case 'export':
      return ['initialisation', 'compression', 'complete']
    case 'download':
      return ['initialisation', 'download', 'complete']
    case 'import':
      return ['initialisation', 'backup', 'extract', 'cleanup', 'complete']
    default:
      return []
  }
})

const messages = ref<string[]>([])

const title = ref<string>('')

const messageBox = useTemplateRef('message-box')

const { t } = useI18n()

const toast = useToast()

function statusKey(status: string | undefined): string {
  return status ? `status${status.charAt(0).toUpperCase() + status.slice(1)}` : ''
}

let interval: ReturnType<typeof setInterval> | number = -1

function resetInterval() {
  clearInterval(interval)
  interval = -1
}

onUnmounted(() => resetInterval())

function startPolling() {
  updateProgress()
  interval = setInterval(updateProgress, 300)
}

defineExpose({
  export: (destination: string) => {
    isOpen.value = true
    mode.value = 'export'
    title.value = t('labelExport')
    snapshot.value = null
    messages.value = []

    programPorter
      .Export(destination)
      .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
      .finally(() => {
        resetInterval()
        updateProgress()
      })

    startPolling()
  },
  download: (url: string): Promise<porter.ImportPreview> => {
    return new Promise((resolve, reject) => {
      isOpen.value = true
      mode.value = 'download'
      title.value = t('labelUpdate')
      snapshot.value = null
      messages.value = []

      programPorter
        .DownloadAndValidate(url)
        .then((preview: porter.ImportPreview) => {
          resolve(preview)
          isOpen.value = false
        })
        .catch(err => {
          toast.add({ title: decodeError(err, t), color: 'error' })
          reject(err)
        })
        .finally(() => {
          resetInterval()
          updateProgress()
        })

      startPolling()
    })
  },
  import: (from: 'url' | 'file', source: string, opts: porter.ImportOptions) => {
    isOpen.value = true
    mode.value = 'import'
    title.value = `${t('labelImport')} (${t(from === 'file' ? 'file' : 'url')})`
    snapshot.value = null
    messages.value = []

    if (from === 'url') {
      programPorter
        .ImportFromURL(opts)
        .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
        .finally(() => {
          resetInterval()
          updateProgress()
        })
    } else {
      programPorter
        .ImportFromFile(source, opts)
        .catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
        .finally(() => {
          resetInterval()
          updateProgress()
        })
    }

    startPolling()
  }
})

const progressToasted = ref(false)

function updateProgress() {
  return programPorter
    .Progress()
    .then(p => {
      progressToasted.value = false

      const scroll =
        messageBox.value &&
        messageBox.value.scrollHeight - messageBox.value.scrollTop - messageBox.value.clientHeight <
          15

      snapshot.value = p
      messages.value.push(...(p.messages ?? []).filter(m => m !== ''))

      if (scroll && messageBox.value) {
        nextTick(() => {
          messageBox.value!.scrollTop = messageBox.value!.scrollHeight
        })
      }
    })
    .catch(err => {
      if (!progressToasted.value) {
        toast.add({ title: decodeError(err, t), color: 'error' })
        progressToasted.value = true
      }
    })
}
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :title="t('progress')"
    :dismissible="false"
    :close="
      snapshot?.status.includes('ed') && !(mode == 'import' && snapshot?.status == 'completed')
    "
    :ui="{
      content: 'h-[80vh] max-h-150',
      body: 'flex flex-col overflow-hidden'
    }"
  >
    <template #body>
      <div class="flex min-h-0 flex-1 flex-col gap-y-4">
        <div class="flex items-center gap-x-3">
          <h2 class="text-lg font-bold">{{ title }}</h2>

          <UBadge class="h-6" :style="{ backgroundColor: `var(--color-${snapshot?.status})` }">
            <span class="truncate capitalize">{{ $t(statusKey(snapshot?.status)) }}</span>
          </UBadge>
        </div>

        <ProgressStepper
          :steps="steps"
          :current-step="snapshot?.step ?? ''"
          :job-status="snapshot?.status"
          :progress="snapshot?.progress ?? 0"
        />

        <div
          ref="message-box"
          class="flex flex-1 flex-col gap-y-2 overflow-y-auto rounded-sm border p-2"
        >
          <p v-for="(m, i) in messages" :key="i" class="text-xs break-all text-gray-400">
            {{ m }}
          </p>
        </div>

        <div class="flex justify-end gap-x-2">
          <div v-show="snapshot?.status == 'pending' || snapshot?.status == 'running'">
            <UButton
              type="button"
              color="error"
              @click="
                () => {
                  programPorter.Abort().catch(err => toast.add({ title: decodeError(err, t), color: 'error' }))
                }
              "
            >
              {{ $t('cancel') }}
            </UButton>
          </div>

          <div v-show="mode == 'import' && snapshot?.status == 'completed'">
            <UButton
              type="button"
              color="primary"
              @click="
                () => {
                  isOpen = false
                  runtime.WindowReloadApp()
                }
              "
            >
              {{ $t('refresh') }}
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
