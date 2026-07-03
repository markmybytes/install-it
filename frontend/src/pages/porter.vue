<script setup lang="ts">
import { Cwd, SelectFile, SelectFolder } from '@/wailsjs/go/main/App'
import { porter } from '@/wailsjs/go/models'
import { ValidateZip } from '@/wailsjs/go/porter/Porter'
import * as appStorage from '@/wailsjs/go/storage/AppSettingStorage'
import { onBeforeMount, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'

const toast = useToast()
const { t } = useI18n()

const progressModal = useTemplateRef('progressModal')

const exportDirectory = ref('')

const importInput = ref<{
  from: 'file' | 'url'
  filePath: string
  url: string
}>({
  from: 'file',
  filePath: '',
  url: ''
})

const previewOpen = ref(false)
const preview = ref<porter.ImportPreview | null>(null)
const previewSource = ref({ from: 'file' as 'file' | 'url', source: '' })
const importOpts = ref({ data: true, settings: true })

onBeforeMount(() => {
  appStorage.All().then(s => (importInput.value.url = s.driver_download_url))

  Cwd().then(cwd => {
    exportDirectory.value = cwd
  })
})

function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`
  }

  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let unitIndex = 0

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }

  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unitIndex]}`
}

function openPreview(p: porter.ImportPreview, from: 'file' | 'url', source: string) {
  preview.value = p
  previewSource.value = { from, source }
  importOpts.value = {
    data: p.hasData,
    settings: p.hasSettings
  }
  previewOpen.value = true
}

function handleValidateFile() {
  if (!importInput.value.filePath) {
    toast.add({ title: t('toastPathNotFound'), color: 'warning' })
    return
  }

  ValidateZip(importInput.value.filePath)
    .then(p => openPreview(p, 'file', importInput.value.filePath))
    .catch(err => {
      if (err.includes('manifest.json not found'))
        toast.add({ title: t('toastInvalidZipFile'), color: 'error' })
      else toast.add({ title: err, color: 'error' })
    })
}

async function handleDownloadUrl() {
  if (!importInput.value.url) {
    toast.add({ title: t('toastUnsupportedUrlProtocol'), color: 'warning' })
    return
  }

  try {
    const p = await progressModal.value?.download(importInput.value.url)
    if (p) {
      openPreview(p, 'url', '')
    }
  } catch {
    // error already handled in modal
  }
}

function handleImport() {
  if (!importOpts.value.data && !importOpts.value.settings) {
    toast.add({ title: t('errNoCategoriesSelected'), color: 'warning' })
    return
  }

  previewOpen.value = false
  const opts = new porter.ImportOptions({
    settings: importOpts.value.settings,
    data: importOpts.value.data
  })

  progressModal.value?.import(previewSource.value.from, previewSource.value.source, opts)
}
</script>

<template>
  <div class="flex h-full flex-col gap-y-6 p-2">
    <PageHeader
      variant="normal"
      :title="$t('titleImportExport')"
      :description="$t('descImportExport')"
    />

    <div class="flex flex-col gap-y-3">
      <h2 class="mb-1 font-medium">{{ $t('labelExportToFile') }}</h2>

      <div class="flex items-center gap-x-6">
        <label class="w-24 content-center text-gray-900">
          {{ $t('labelExportDestination') }}
        </label>

        <div class="flex w-full gap-x-2">
          <UInput
            v-model="exportDirectory"
            type="url"
            name="export_directory"
            color="primary"
            class="grow"
          />

          <UButton
            type="button"
            color="primary"
            @click="
              () => {
                SelectFolder(false).then(path => {
                  if (path != '') {
                    exportDirectory = path
                  }
                })
              }
            "
          >
            {{ $t('select') }}
          </UButton>
        </div>
      </div>

      <div class="flex justify-end">
        <UButton
          type="button"
          color="secondary"
          class="mt-3 w-28 justify-center"
          @click="
            () => {
              if (!exportDirectory) {
                toast.add({ title: $t('toastEnterExportPath'), color: 'warning' })
              } else {
                progressModal?.export(exportDirectory)
              }
            }
          "
        >
          {{ $t('labelExport') }}
        </UButton>
      </div>
    </div>

    <div class="flex flex-col gap-y-3">
      <div class="flex gap-x-4">
        <h2 class="mb-1 font-medium">
          {{ $t('labelImport') }}
        </h2>

        <div class="relative inline-flex rounded-3xl border p-0.5">
          <button
            class="z-10 rounded-3xl px-3 text-center text-xs select-none"
            @click="importInput.from = 'file'"
          >
            {{ $t('labelImportFile') }}
          </button>

          <button
            class="z-10 rounded-3xl px-3 text-center text-xs select-none"
            @click="importInput.from = 'url'"
          >
            {{ $t('labelImportUrl') }}
          </button>

          <span
            class="absolute top-1 rounded-3xl bg-gray-300 transition duration-200"
            :class="{ 'translate-x-full': importInput.from == 'url' }"
            style="width: calc(50% - 2px); height: calc(100% - 8px)"
          ></span>
        </div>
      </div>

      <!-- from file -->
      <div v-if="importInput.from == 'file'" class="flex items-center gap-x-6">
        <label class="w-24 content-center text-gray-900">
          {{ $t('file') }}
        </label>

        <div class="flex w-full gap-x-2">
          <UInput
            v-model="importInput.filePath"
            type="text"
            name="driver_download_url"
            placeholder="install-it.zip"
            color="primary"
            class="pointer-events-none grow"
          />

          <UButton
            type="button"
            color="primary"
            @click="
              () => {
                SelectFile(false).then(path => {
                  if (path != '') {
                    importInput.filePath = path
                  }
                })
              }
            "
          >
            {{ $t('select') }}
          </UButton>
        </div>
      </div>

      <!-- from url -->
      <div v-else class="flex gap-x-6">
        <label class="w-24 content-center text-gray-900">
          {{ $t('url') }}
        </label>

        <div class="flex w-full gap-x-2">
          <UInput
            v-model="importInput.url"
            type="url"
            placeholder="https://..."
            color="primary"
            class="grow"
          />
        </div>
      </div>

      <div class="flex justify-end">
        <UButton
          type="button"
          color="secondary"
          class="mt-3 w-28 justify-center"
          @click="importInput.from == 'file' ? handleValidateFile() : handleDownloadUrl()"
        >
          {{ importInput.from == 'file' ? $t('labelValidate') : $t('labelUpdate') }}
        </UButton>
      </div>
    </div>
  </div>

  <!-- Preview Modal -->
  <UModal v-model:open="previewOpen" :title="$t('titleImportPreview')">
    <template #body>
      <div v-if="preview" class="flex flex-col gap-y-4">
        <div>
          <p class="font-medium break-all">
            <Icon icon="mdi:package-variant" class="mr-1 inline-block" />
            {{ previewSource.from == 'file' ? importInput.filePath : importInput.url }}
          </p>

          <p class="text-sm text-gray-500">
            {{ $t('labelCreated') }}:
            {{ new Date(preview.exportedAt).toLocaleDateString() }}
          </p>
        </div>

        <div>
          <p class="mb-1 font-medium">{{ $t('msgPreviewContains') }}</p>

          <ul class="space-y-1 text-sm">
            <li class="flex items-center gap-x-2">
              <Icon
                :icon="preview.hasData ? 'mdi:check' : 'mdi:close'"
                :class="preview.hasData ? 'text-green-500' : 'text-gray-400'"
              />

              <span>
                {{ $t('labelData') }}
                <template v-if="preview.hasData && (preview.hasDatabase || preview.hasDrivers)">
                  —
                  <template v-if="preview.hasDatabase">{{ $t('labelDatabase') }}</template>

                  <template v-if="preview.hasDatabase && preview.hasDrivers">, </template>

                  <template v-if="preview.hasDrivers">
                    {{ $t('labelDrivers') }} ({{ preview.driverCount }} files,
                    {{ formatBytes(preview.driverSize) }})
                  </template>
                </template>
              </span>
            </li>

            <li class="flex items-center gap-x-2">
              <Icon
                :icon="preview.hasSettings ? 'mdi:check' : 'mdi:close'"
                :class="preview.hasSettings ? 'text-green-500' : 'text-gray-400'"
              />

              <span>{{ $t('labelSettings') }}</span>
            </li>
          </ul>
        </div>

        <div
          v-if="preview.hasDrivers && !preview.hasDatabase"
          class="flex gap-x-2 rounded bg-amber-50 p-2 text-sm text-amber-700"
        >
          <Icon icon="mdi:alert" class="shrink-0" />

          <span>{{ $t('warnDriversNoDb') }}</span>
        </div>

        <div
          v-if="preview.hasDatabase && !preview.hasDrivers"
          class="flex gap-x-2 rounded bg-amber-50 p-2 text-sm text-amber-700"
        >
          <Icon icon="mdi:alert" class="shrink-0" />

          <span>{{ $t('warnDbNoDrivers') }}</span>
        </div>

        <div>
          <p class="mb-2 font-medium">{{ $t('msgPreviewSelect') }}</p>

          <div class="flex flex-col gap-y-2">
            <label
              class="flex items-center select-none"
              :class="{ 'cursor-not-allowed opacity-50': !preview.hasData }"
            >
              <UCheckbox
                v-model="importOpts.data"
                :disabled="!preview.hasData"
                color="primary"
                class="me-2"
              />

              <span>{{ $t('labelData') }}</span>

              <span v-if="!preview.hasData" class="ml-1 text-xs text-gray-400">
                ({{ $t('labelNotAvailable') }})
              </span>
            </label>

            <label
              class="flex items-center select-none"
              :class="{ 'cursor-not-allowed opacity-50': !preview.hasSettings }"
            >
              <UCheckbox
                v-model="importOpts.settings"
                :disabled="!preview.hasSettings"
                color="primary"
                class="me-2"
              />

              <span>{{ $t('labelSettings') }}</span>

              <span v-if="!preview.hasSettings" class="ml-1 text-xs text-gray-400">
                ({{ $t('labelNotAvailable') }})
              </span>
            </label>
          </div>
        </div>

        <div class="flex justify-end gap-x-2">
          <UButton color="neutral" variant="outline" @click="previewOpen = false">
            {{ $t('cancel') }}
          </UButton>

          <UButton color="primary" @click="handleImport">
            {{ $t('labelImport') }}
          </UButton>
        </div>
      </div>
    </template>
  </UModal>

  <ProgressModal ref="progressModal"></ProgressModal>
</template>
