<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  steps: string[]
  currentStep: string
  jobStatus: string | undefined
  progress: number
}>()

function stepKey(name: string): string {
  const map: Record<string, string> = {
    initialisation: 'stepInit',
    compression: 'stepCompression',
    complete: 'complete',
    download: 'stepDownload',
    compare: 'stepCompare',
    backup: 'stepBackup',
    extract: 'stepExtract',
    cleanup: 'stepCleanup'
  }
  return map[name] ?? name
}

const processedSteps = computed(() => {
  const idx = props.steps.indexOf(props.currentStep)
  const st = props.jobStatus

  return props.steps.map((name, i) => {
    let state: 'pending' | 'running' | 'completed' | 'failed' | 'aborted' = 'pending'

    if (name === 'initialisation') {
      state = !st || st === 'pending' ? 'pending' : idx === -1 ? 'running' : 'completed'
    } else if (name === 'complete') {
      state = st === 'completed' || st === 'failed' || st === 'aborted' ? st : 'pending'
    } else if (st === 'completed') {
      state = 'completed'
    } else if (st === 'failed' || st === 'aborted') {
      state = idx === -1 || i === idx ? st : i < idx ? 'completed' : 'pending'
    } else {
      state = idx !== -1 && i < idx ? 'completed' : i === idx ? 'running' : 'pending'
    }

    const icons: Record<string, string> = {
      running: 'mdi:loading',
      completed: 'mdi:check',
      failed: 'mdi:alert',
      aborted: 'mdi:alert'
    }
    const icon =
      name === 'complete' && state === 'completed'
        ? 'mdi:flag-checkered'
        : name === 'initialisation' && state === 'running'
          ? 'mdi:play'
          : state === 'pending'
            ? 'mdi:circle-outline'
            : icons[state]

    return { name, state, icon, color: `var(--color-${state})` }
  })
})
</script>

<template>
  <ol
    class="grid w-full items-start"
    :style="{
      gridTemplateColumns: processedSteps
        .map((_, i) => (i === 0 ? 'minmax(3rem, max-content)' : '1fr minmax(3rem, max-content)'))
        .join(' ')
    }"
  >
    <template v-for="(step, i) in processedSteps" :key="step.name">
      <div v-if="i > 0" class="mx-1 mt-3 h-1 rounded bg-gray-200 lg:mt-4">
        <div
          class="h-full rounded transition-all duration-300"
          :class="{ 'animate-pulse': processedSteps[i - 1]?.state === 'running' }"
          :style="{
            width: ['completed', 'failed', 'aborted'].includes(processedSteps[i - 1]?.state || '')
              ? '100%'
              : processedSteps[i - 1]?.state === 'running'
                ? `${Math.floor(progress * 100)}%`
                : '0%',
            backgroundColor:
              processedSteps[i - 1]?.state !== 'pending'
                ? processedSteps[i - 1]?.color
                : 'transparent'
          }"
        ></div>
      </div>

      <div class="flex flex-col items-center text-center">
        <span
          class="flex size-6 shrink-0 items-center justify-center rounded-full lg:size-8"
          :style="{
            backgroundColor: step.state !== 'pending' ? step.color : 'transparent',
            borderColor: step.color,
            color: step.state !== 'pending' ? 'white' : 'var(--color-gray-300)'
          }"
        >
          <Icon :icon="step.icon" :class="{ 'animate-spin': step.state === 'running' }" />
        </span>

        <span
          class="mt-1.5 w-full text-xs leading-tight wrap-break-word lg:text-sm"
          :class="{ 'text-gray-400': step.state === 'pending' }"
        >
          {{ $t(stepKey(step.name)) }}
        </span>
      </div>
    </template>
  </ol>
</template>
