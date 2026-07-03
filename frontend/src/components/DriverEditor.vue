<script setup lang="ts">
import ChipInput from '@/components/ChipInput.vue'
import DriverSelector from '@/components/DriverSelector.vue'
import { ExecutableExists, SelectFile } from '@/wailsjs/go/main/App'
import { storage } from '@/wailsjs/go/models'
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

defineProps<{
  index: number
  isNew?: boolean
  expanded?: boolean
}>()

const emit = defineEmits<{
  remove: [id: number]
  toggle: [id: number]
}>()

const driver = defineModel<storage.Driver>('driver', { required: true })

const { t } = useI18n()

const toast = useToast()

const groupStore = useDriverGroupStore()

const FLAGS = {
  'Intel LAN': ['/s'],
  'Realtek LAN': ['-s'],
  'Nvidia Display': ['-s', '-noreboot', 'Display.Driver'],
  'AMD Display': ['-install'],
  'Intel Display': ['-s', '--noExtras'],
  'Intel Wifi': ['-q'],
  'Intel BT': ['/quiet', '/norestart'],
  'Intel Chipset': ['-s', '-norestart'],
  'AMD Chipset': ['/S']
}

const flagItems = Object.entries(FLAGS).map(([name, flags]) => ({
  label: name,
  onSelect: () => {
    driver.value.flags = [...flags]
  }
}))

const path = ref<{ exists: boolean | null }>({ exists: null })
let pathCheckTimeout: ReturnType<typeof setTimeout> | null = null

watch(
  () => driver.value.path,
  newPath => {
    path.value.exists = null

    if (pathCheckTimeout) {
      clearTimeout(pathCheckTimeout)
    }

    if (!newPath) return

    pathCheckTimeout = setTimeout(() => {
      ExecutableExists(newPath)
        .then(exists => {
          path.value.exists = exists
        })
        .catch(() => {
          path.value.exists = false
        })
    }, 300)
  },
  { immediate: true }
)

const notFound = computed(() => path.value.exists === false)

onUnmounted(() => {
  if (pathCheckTimeout) clearTimeout(pathCheckTimeout)
})

const codeStrings = computed({
  get: () => driver.value.allowRtCodes.map(String),
  set: v => {
    driver.value.allowRtCodes = v.map(s => parseInt(s)).filter(n => !Number.isNaN(n))
  }
})

function handleDone() {
  if (!driver.value.path?.trim()) {
    toast.add({ title: t('toastPathRequired'), color: 'warning' })
    return
  }

  if (typeof driver.value.minExeTime === 'string') {
    driver.value.minExeTime = Number(driver.value.minExeTime) || 5
  }

  if (driver.value.id !== undefined) emit('toggle', driver.value.id)
}

function handleRemove() {
  if (driver.value.id !== undefined) emit('remove', driver.value.id)
}
</script>

<template>
  <div
    class="rounded-lg border bg-white shadow-sm transition-colors"
    :class="
      expanded
        ? 'border-half-baked-500 ring-1 ring-half-baked-500'
        : 'border-gray-200 hover:border-half-baked-300'
    "
  >
    <button
      v-if="!expanded"
      type="button"
      class="flex w-full cursor-pointer items-start gap-3 p-3 text-left"
      @click="driver.id !== undefined && emit('toggle', driver.id)"
    >
      <span
        class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-gray-100 text-xs font-bold text-gray-500 xl:text-sm"
      >
        {{ index + 1 }}
      </span>

      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1.5">
          <span
            class="truncate text-sm font-bold xl:text-base"
            :class="driver.name ? 'text-gray-800' : 'text-gray-400'"
          >
            {{ driver.name || $t('msgUnnamedDriver') }}
          </span>

          <span
            v-if="notFound"
            class="rounded border border-red-200 bg-red-100 px-1.5 py-0.5 text-[10px] font-bold text-red-600 xl:text-xs"
          >
            {{ $t('labelMissingExe') }}
          </span>
        </div>

        <p class="mt-0.5 truncate font-mono text-[11px] text-gray-500 xl:text-xs">
          {{ driver.path || $t('msgNoPath') }}
        </p>

        <div class="mt-1.5 flex flex-wrap gap-1.5">
          <span
            v-if="driver.flags.length"
            class="inline-flex items-center gap-1 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-600 xl:text-xs"
            :title="$t('fieldArgument')"
          >
            <Icon icon="mdi:console" class="h-3 w-3" />

            <span class="font-mono">{{ driver.flags.join(' ') }}</span>
          </span>

          <span
            v-if="driver.allowRtCodes?.length"
            class="inline-flex items-center gap-1 rounded bg-purple-50 px-1.5 py-0.5 text-[10px] font-semibold text-purple-600 xl:text-xs"
            :title="$t('fieldAllowedExitCode')"
          >
            <Icon icon="mdi:alert-circle-success-outline" class="h-3 w-3" />
            {{ driver.allowRtCodes.join(', ') }}
          </span>

          <span
            v-if="driver.minExeTime > 0"
            class="inline-flex items-center gap-1 rounded bg-blue-50 px-1.5 py-0.5 text-[10px] font-semibold text-blue-600 xl:text-xs"
            :title="$t('fieldMinExecuteTime')"
          >
            <Icon icon="mdi:timer-outline" class="h-3 w-3" />
            {{ driver.minExeTime }}s
          </span>

          <span
            v-if="driver.incompatibles.length"
            class="inline-flex items-center gap-1 rounded bg-yellow-100 px-1.5 py-0.5 text-[10px] font-semibold text-yellow-700 xl:text-xs"
            :title="$t('labelIncompatibleWith')"
          >
            <Icon icon="mdi:source-merge" class="h-3 w-3" />
            {{ driver.incompatibles.length }}
          </span>
        </div>
      </div>

      <Icon icon="mdi:chevron-down" class="mt-1 h-5 w-5 shrink-0 text-gray-400" />
    </button>

    <div v-else class="space-y-4 p-3 xl:p-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-gray-100 text-xs font-bold text-gray-500 xl:text-sm"
          >
            {{ index + 1 }}
          </span>

          <span class="text-xs font-bold text-gray-500 uppercase xl:text-sm">
            {{ isNew ? $t('titleCreateDriver') : $t('edit') }}
          </span>
        </div>

        <button
          type="button"
          class="text-gray-400 hover:text-gray-600"
          :title="$t('collapse')"
          @click="handleDone"
        >
          <Icon icon="mdi:chevron-up" class="h-5 w-5" />
        </button>
      </div>

      <fieldset>
        <label class="mb-1.5 block text-xs font-bold text-gray-700 xl:mb-2 xl:text-sm">
          {{ t('name') }}
        </label>

        <input
          :value="driver.name"
          type="text"
          name="name"
          placeholder="e.g. Realtek LAN Driver"
          class="w-full rounded-lg border border-gray-200 bg-gray-50/80 px-3.5 py-2 text-xs text-gray-900 transition-all placeholder:font-sans placeholder:text-gray-400 focus:border-half-baked-500 focus:bg-white focus:ring-2 focus:ring-half-baked-500/30 focus:outline-none xl:text-sm"
          @input="event => (driver.name = (event.target as HTMLInputElement).value)"
        />
      </fieldset>

      <fieldset>
        <label class="mb-1.5 block text-xs font-bold text-gray-700 xl:mb-2 xl:text-sm">
          {{ t('path') }} <span class="text-red-500">*</span>
        </label>

        <div class="flex gap-2">
          <input
            :value="driver.path"
            type="text"
            name="path"
            placeholder="C:\Drivers\..."
            class="flex-1 rounded-lg border border-gray-200 bg-gray-50/80 px-3.5 py-2 font-mono text-xs text-gray-900 transition-all placeholder:font-sans placeholder:text-gray-400 focus:border-half-baked-500 focus:bg-white focus:ring-2 focus:ring-half-baked-500/30 focus:outline-none xl:text-sm"
            @input="event => (driver.path = (event.target as HTMLInputElement).value)"
          />

          <button
            type="button"
            class="inline-flex shrink-0 items-center justify-center gap-1 rounded-lg border border-gray-200 bg-gray-50/80 px-3.5 py-2 text-xs font-bold text-gray-700 transition-colors hover:border-half-baked-300 hover:bg-white xl:text-sm"
            @click="
              SelectFile(true).then(path => {
                if (path) driver.path = path
              })
            "
          >
            <Icon icon="mdi:folder-open" />

            <span class="hidden sm:inline">{{ t('labelSelectFile') }}</span>
          </button>
        </div>

        <p
          v-if="path.exists === false"
          class="mt-1 inline-flex items-center gap-1 text-[10px] text-red-600 xl:text-xs"
        >
          <Icon icon="mdi:alert-circle" />
          {{ $t('labelMissingExe') }}
        </p>
      </fieldset>

      <fieldset>
        <label class="mb-1.5 block text-xs font-bold text-gray-700 xl:mb-2 xl:text-sm">
          {{ t('fieldArgument') }}
        </label>

        <ChipInput v-model="driver.flags" placeholder="e.g. /S (Enter)">
          <UDropdownMenu :items="[flagItems]" :ui="{ content: 'max-h-58 overflow-y-auto' }">
            <button
              type="button"
              class="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-bold text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700 xl:text-sm"
            >
              <Icon icon="mdi:magic-staff" />

              <span class="hidden sm:inline">{{ t('labelPresets') }}</span>
            </button>
          </UDropdownMenu>
        </ChipInput>
      </fieldset>

      <div class="grid grid-cols-2 gap-3">
        <fieldset>
          <label class="mb-1.5 block text-xs font-bold text-gray-700 xl:mb-2 xl:text-sm">
            {{ t('fieldAllowedExitCode') }}
          </label>

          <ChipInput
            v-model="codeStrings"
            placeholder="e.g. 1 (Enter/Space)"
            min-input-width="80px"
            :commit-keys="['enter', ' ']"
            :parse="(raw: string) => raw.trim()"
            :accept="(parsed: string) => /^-?\d+$/.test(parsed)"
            style="
              --chip-bg: #f4f4f5;
              --chip-border: #e4e4e7;
              --chip-text: #3f3f46;
              --chip-close-text: #a1a1aa;
              --chip-close-bg-hover: #e4e4e7;
            "
          />
        </fieldset>

        <fieldset>
          <div class="mb-1.5 flex items-center justify-between xl:mb-2">
            <label class="block text-xs font-bold text-gray-700 xl:text-sm">
              {{ t('fieldMinExecuteTime') }}
            </label>

            <span
              class="rounded border border-half-baked-100 bg-half-baked-50 px-2 py-0.5 font-mono text-[10px] font-extrabold text-half-baked-600 shadow-sm xl:text-xs"
            >
              {{ driver.minExeTime > 0 ? driver.minExeTime + 's' : '—' }}
            </span>
          </div>

          <div class="flex items-center gap-3 pt-1">
            <input
              :value="driver.minExeTime"
              type="range"
              min="0"
              max="120"
              step="5"
              name="minExeTime"
              class="h-1.5 w-full cursor-pointer appearance-none rounded-lg bg-gray-200 focus:ring-2 focus:ring-half-baked-500/50 focus:outline-none"
              @input="
                event => (driver.minExeTime = Number((event.target as HTMLInputElement).value))
              "
            />

            <span
              class="w-8 text-right font-mono text-[10px] font-bold tracking-wider text-gray-400"
            >
              120s
            </span>
          </div>

          <p class="mt-1 text-[10px] text-gray-400 xl:text-xs">
            {{ t('descMinExecuteTime') }}
          </p>
        </fieldset>
      </div>

      <details class="group overflow-hidden rounded-lg border border-gray-200 bg-gray-50 shadow-sm">
        <summary
          class="flex cursor-pointer list-none items-center justify-between p-3 text-xs font-bold text-gray-700 select-none xl:text-sm"
        >
          <div class="flex items-center gap-2">
            <Icon icon="mdi:alert-octagram-outline" class="text-amber-500 xl:text-lg" />

            {{ t('labelIncompatibleWith') }}

            <span
              class="rounded border border-amber-200 px-1.5 py-0.5 text-[10px] xl:text-xs"
              :class="
                driver.incompatibles.length
                  ? 'bg-amber-100 text-amber-700'
                  : 'invisible border-transparent'
              "
            >
              {{ driver.incompatibles.length }} set
            </span>
          </div>

          <Icon icon="mdi:chevron-down" class="transition-transform group-open:rotate-180" />
        </summary>

        <div class="border-t border-gray-200 bg-white p-3">
          <p class="mb-2 text-xs font-medium text-gray-500 xl:text-sm">
            {{ t('descIncompatible') }}
          </p>

          <DriverSelector
            v-model="driver.incompatibles"
            group-by="driver"
            :driver-groups="groupStore.groups"
          />
        </div>
      </details>

      <div class="flex items-center justify-end gap-2 border-t border-gray-100 pt-3">
        <UButton
          v-if="!isNew"
          type="button"
          color="error"
          variant="ghost"
          size="sm"
          @click="handleRemove"
        >
          <Icon icon="mdi:trash-can" class="mr-1" />
          {{ t('delete') }}
        </UButton>

        <UButton type="button" color="primary" size="sm" @click="handleDone">
          <Icon icon="mdi:check" class="mr-1" />
          {{ t('done') }}
        </UButton>
      </div>
    </div>
  </div>
</template>
