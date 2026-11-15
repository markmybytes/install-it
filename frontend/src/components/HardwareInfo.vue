<script setup lang="ts">
import { sysinfo } from '@/wailsjs/go/models'
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** null renders nothing; callers may bind the page's hw+os ref directly */
    info?: { hw: sysinfo.ResolvedHardware; os: sysinfo.OSInfo } | null
    size?: 'normal' | 'large'
    /** null/undefined hides the badge */
    cpuTemp?: { value: number | null; error: string | null } | null
  }>(),
  { size: 'normal', info: null, cpuTemp: null }
)

// Full literal class strings so Tailwind JIT can see them
const STYLES: Record<
  typeof props.size,
  { heading: string; value: string; block: string; badge: string }
> = {
  normal: { heading: 'text-sm', value: 'text-sm break-words', block: '', badge: 'text-xs' },
  large: { heading: 'text-lg', value: 'mt-1 text-lg break-words', block: 'mb-2', badge: 'text-base' }
}

function hwKey(part: string): string {
  return `hw${part.charAt(0).toUpperCase() + part.slice(1)}`
}

const tempClass = computed(() =>
  props.cpuTemp?.error != null
    ? 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
    : props.cpuTemp!.value! < 45
      ? 'bg-apple-green-100 text-apple-green-700'
      : props.cpuTemp!.value! >= 85
        ? 'bg-red-100 text-red-700'
        : 'bg-yellow-100 text-yellow-700'
)

const tempText = computed(() => `${props.cpuTemp?.error != null ? '--' : props.cpuTemp!.value}°C`)
</script>

<template>
  <template v-if="info">
    <div v-for="[part, names] in Object.entries(info.hw)" :key="part" :class="STYLES[size].block">
      <h2 :class="STYLES[size].heading" class="font-bold">
        {{ $t(hwKey(part)) }}
        <UBadge
          v-if="part === 'cpu' && cpuTemp && (cpuTemp.value != null || cpuTemp.error != null)"
          size="sm"
          class="ms-1 rounded px-1.5 py-0 font-medium"
          :class="[STYLES[size].badge, tempClass]"
        >
          {{ tempText }}
        </UBadge>
      </h2>

      <p v-for="(name, i) in names" :key="i" :class="STYLES[size].value">
        {{ name }}
      </p>
    </div>

    <div>
      <h2 :class="STYLES[size].heading" class="font-bold">{{ $t('hwOs') }}</h2>

      <p :class="STYLES[size].value">
        {{ [info.os.caption, info.os.displayVersion].filter(Boolean).join(' ') }}
        —
        {{ $t(info.os.activated ? 'osActivationActivated' : 'osActivationNotActivated') }}
      </p>
    </div>
  </template>
</template>
