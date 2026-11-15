<script setup lang="ts">
import { sysinfo } from '@/wailsjs/go/models'

type Size = 'sm' | 'lg'

withDefaults(
  defineProps<{
    hw: sysinfo.ResolvedHardware
    os: sysinfo.OSInfo
    /** Tailwind-style size token; 'lg' is the fullscreen photo mode */
    size?: Size
    cpuTemp?: number | null
    cpuTempError?: string | null
  }>(),
  { size: 'sm' }
)

// Full literal class strings so Tailwind JIT can see them
const heading: Record<Size, string> = { sm: 'text-sm', lg: 'text-2xl' }
const value: Record<Size, string> = { sm: 'text-sm', lg: 'mt-1 text-3xl' }
const block: Record<Size, string> = { sm: '', lg: 'mb-8' }
const badge: Record<Size, string> = {
  sm: 'px-1.5 py-0 text-xs',
  lg: 'px-3 py-1 text-2xl'
}

function hwKey(part: string): string {
  return `hw${part.charAt(0).toUpperCase() + part.slice(1)}`
}
</script>

<template>
  <div v-for="[part, names] in Object.entries(hw)" :key="part" :class="block[size]">
    <h2 :class="heading[size]" class="font-bold">
      {{ $t(hwKey(part)) }}
      <UBadge
        v-if="part === 'cpu' && (cpuTemp != null || cpuTempError != null)"
        size="sm"
        class="ms-1 rounded font-medium"
        :class="[
          badge[size],
          cpuTempError != null
            ? 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
            : cpuTemp! < 45
              ? 'bg-apple-green-100 text-apple-green-700'
              : cpuTemp! >= 85
                ? 'bg-red-100 text-red-700'
                : 'bg-yellow-100 text-yellow-700'
        ]"
      >
        {{ cpuTempError != null ? '--' : cpuTemp }}°C
      </UBadge>
    </h2>

    <p v-for="(name, i) in names" :key="i" :class="value[size]">
      {{ name }}
    </p>
  </div>

  <div>
    <h2 :class="heading[size]" class="font-bold">{{ $t('hwOs') }}</h2>

    <p :class="value[size]">
      {{ [os.caption, os.displayVersion].filter(Boolean).join(' ') }}
      —
      {{ $t(os.activated ? 'osActivationActivated' : 'osActivationNotActivated') }}
    </p>
  </div>
</template>
