<script setup lang="ts">
import { useExecute } from '@/composables/useExecute'
import type { Command, Process } from '@/types/execute'
import type { storage } from '@/wailsjs/go/models'
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const emit = defineEmits<{ completed: [] }>()

const isOpen = ref(false)

const { processes, start, abort, allCompleted, noActive } = useExecute()

const { t } = useI18n()

const toast = useToast()

defineExpose({
  show: (parallel: boolean, cmds: Array<Command>, groups: ReadonlyArray<storage.DriverGroup>) => {
    isOpen.value = true
    start(parallel, cmds, groups)
  },
  hide: () => {
    isOpen.value = false
  }
})

watch(allCompleted, (now, was) => {
  if (now && !was) {
    emit('completed')
    toast.add({ title: t('msgFinished'), color: 'success' })
  }
})
watch(noActive, (now, was) => {
  if (now && !was && !allCompleted.value) {
    toast.add({ title: t('msgFinished'), color: 'info' })
  }
})

function getProcessName(process: Process) {
  return process.command.name
    ? `${process.command.groupName} - ${process.command.name}`
    : process.command.groupName
}

function handleAbort(process: Process) {
  return abort(process).then(outcome => {
    if (outcome.ok) return
    if (outcome.code === 'errExecuteIdNotFound') {
      toast.add({ title: outcome.message, color: 'warning' })
    } else {
      toast.add({ title: `[${getProcessName(process)}] ${outcome.message}`, color: 'error' })
    }
  })
}
</script>

<template>
  <UModal
    v-model:open="isOpen"
    :dismissible="false"
    :title="$t('titleExecutionStatus')"
    :close="!processes.some(cmd => ['pending', 'running', 'aborting'].includes(cmd.status))"
  >
    <template #body>
      <template v-for="(process, i) in processes" :key="i">
        <TaskStatus :process="process" @abort="handleAbort(process)"></TaskStatus>
      </template>

      <div
        v-show="
          processes.every(p => p.status.includes('ed')) &&
          processes.some(p => p.status != 'completed')
        "
        class="flex justify-end border-t pt-2"
      >
        <UButton
          color="secondary"
          size="sm"
          @click="
            (event: MouseEvent) => {
              $emit('completed')
              toast.add({ title: t('msgFinished'), color: 'success' })

              // @ts-ignore
              event.currentTarget?.remove()
            }
          "
        >
          <Icon icon="mdi:arrow-right" />
          {{ $t('actionForceComplete') }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
