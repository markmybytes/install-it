<script setup lang="ts">
import type { Process } from '@/types/execute'

const props = defineProps<{ process: Process }>()

defineEmits<{ abort: [] }>()

function statusShortKey(status: string): string {
  return `statusShort${status.charAt(0).toUpperCase() + status.slice(1)}`
}
</script>

<template>
  <div class="flex min-h-9 border-t">
    <div class="w-2/6 content-center truncate pe-1 text-xs">
      <p class="truncate font-medium">{{ props.process.command.groupName }}</p>

      <p v-if="props.process.command.name" class="truncate">
        &nbsp;&nbsp;{{ `⤷ ${props.process.command.name}` }}
      </p>
    </div>

    <div class="flex w-4/6 items-center py-1 ps-1">
      <!-- status badge -->
      <div class="w-[4.1rem] shrink-0">
        <UBadge
          :class="{ 'animate-pulse': ['running', 'aborting'].includes(props.process.status) }"
          class="h-6 max-w-[96%]"
          size="md"
          :style="{ backgroundColor: `var(--color-${props.process.status})` }"
        >
          <span class="truncate text-sm">
            {{ $t(statusShortKey(props.process.status)).toUpperCase() }}
          </span>
        </UBadge>
      </div>

      <!-- messages -->
      <template v-if="props.process.status == 'speeded' || props.process.status == 'failed'">
        <div class="line-clamp-3 text-sm break-all">
          {{ $t('msgExitCode', { code: props.process.result?.exitCode }) }}

          <p v-if="props.process.status == 'speeded'" class="text-xs text-orange-300">
            {{
              $t('msgEarlyExit', {
                second: `${(props.process.result?.lapse ?? -1).toFixed(1)}/${props.process.command.config.minExeTime}`
              })
            }}
          </p>

          <p
            v-else-if="props.process.result && props.process.result.error !== ''"
            class="font-mono text-xs text-red-400"
          >
            {{ $t(props.process.result.error) }}
          </p>

          <p v-else class="font-mono text-xs text-red-400">
            {{ props.process.result?.stderr || props.process.result?.stdout }}
          </p>
        </div>
      </template>

      <template v-else-if="props.process.status == 'errored'">
        <div class="line-clamp-2 font-mono text-sm break-all">
          {{
            props.process.result?.error?.split(':').slice(1).join(':').trim() ??
            $t('errStartFailed')
          }}
        </div>
      </template>

      <template v-else-if="props.process.status == 'completed'">
        <div class="text-xs text-gray-300">
          <p class="truncate">
            {{ $t('msgExitCode', { code: props.process.result?.exitCode }) }}
          </p>

          <p class="truncate">
            {{ $t('msgExecuteTime', { second: Math.round(props.process.result?.lapse ?? -1) }) }}
          </p>
        </div>
      </template>

      <!-- abort button -->
      <div
        v-show="props.process.status == 'pending' || props.process.status == 'running'"
        class="ms-auto font-normal"
      >
        <UButton size="xs" @click="$emit('abort')">
          {{ $t('actionAbort') }}
        </UButton>
      </div>
    </div>
  </div>
</template>
