<script setup lang="ts">
import type { Command, Process } from '@/types/execute'
import * as executor from '@/wailsjs/go/execute/CommandExecutor'
import { status } from '@/wailsjs/go/models'
import * as runtime from '@/wailsjs/runtime/runtime'
import { decodeError } from '@/utils/index'
import AsyncLock from 'async-lock'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const emit = defineEmits<{ completed: [] }>()

const isOpen = ref(false)

defineExpose({
  show: async (parallel: boolean, cmds: Array<Command>) => {
    isOpen.value = true

    isParallel = parallel

    processes.value = cmds.map(vals => ({ command: { ...vals }, status: status.Status.PENDING }))
    dispatchCommand()
  },
  hide: () => {
    isOpen.value = false
  }
})

const { t } = useI18n()

const toast = useToast()

const lock = new AsyncLock()

let isParallel = false

const processes = ref<Array<Process>>([])

runtime.EventsOn('execute:exited', async (id: string, result: NonNullable<Process['result']>) => {
  const process = processes.value.find(c => c.procId === id)!
  // The event payload's Error field is now a stable i18n code (e.g. errExecuteIdNotFound);
  // localize it before storing so downstream display never shows the raw code.
  process.result = {
    ...result,
    error: result.error ? decodeError({ code: result.error }, t) : result.error
  }

  if (result.aborted) {
    process.status = status.Status.ABORTED
  } else if (![0, ...process.command.config.allowRtCodes].includes(result.exitCode)) {
    process.status = status.Status.FAILED
  } else if (result.lapse < process.command.config.minExeTime) {
    process.status = status.Status.SPEEDED
  } else {
    process.status = status.Status.COMPLETED
  }

  dispatchCommand().then(() => {
    if (processes.value.every(c => c.status === 'completed')) {
      emit('completed')
      toast.add({ title: t('msgFinished'), color: 'success' })
    } else if (processes.value.every(c => !c.status.includes('ing'))) {
      toast.add({ title: t('msgFinished'), color: 'info' })
    }
  })
})

function getProcessName(process: Process) {
  return process.command.name
    ? `${process.command.groupName} - ${process.command.name}`
    : process.command.groupName
}

async function dispatchCommand() {
  lock.acquire('executor', async () => {
    const pendings = processes.value
      .filter(c => c.status === 'pending')
      .slice(0, isParallel ? undefined : 1)

    for (const process of pendings) {
      if (
        !process.command.config.incompatibles.every(id =>
          processes.value.filter(p => p.status === 'running').every(p => p.command.id != id)
        )
      ) {
        continue
      }

      await executor
        .Run(process.command.config.program, process.command.config.options)
        .then(processId => {
          process.status = status.Status.RUNNING
          process.procId = processId
        })
        .catch(error => {
          process.status = status.Status.ERRORED
          process.result = {
            lapse: -1,
            exitCode: -1,
            stdout: '',
            stderr: '',
            error: decodeError(error, t),
            aborted: false
          }
        })
    }
  })
}

async function handleAbort(process: Process) {
  return lock
    .acquire('executor', () => {
      if (process.status == 'pending' || process.status == 'running') {
        process.status =
          process.procId == undefined || process.procId == ''
            ? status.Status.ABORTED
            : status.Status.ABORTING
      }
    })
    .then(() => {
      if (process.status != 'aborting') {
        return
      }

      // `aborted` status will be updated at `execute:exited` event handler
      executor.Abort(process.procId!).catch(error => {
        const code = (error as { code?: string })?.code ?? ''
        if (code === 'errExecuteIdNotFound') {
          toast.add({ title: decodeError(error, t), color: 'warning' })
          return
        }

        toast.add({ title: `[${getProcessName(process)}] ${decodeError(error, t)}`, color: 'error' })

        process.status = status.Status.ERRORED
        process.result = {
          lapse: -1,
          exitCode: -1,
          stdout: '',
          stderr: '',
          error: decodeError(error, t),
          aborted: false
        }
      })
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
