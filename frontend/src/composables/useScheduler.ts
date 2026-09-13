import type { Command, Process } from '@/types/execute'
import * as executor from '@/wailsjs/go/execute/CommandExecutor'
import { status } from '@/wailsjs/go/models'
import type { storage } from '@/wailsjs/go/models'
import * as runtime from '@/wailsjs/runtime/runtime'
import { decodeError } from '@/utils/index'
import { expandIncompat, schedule, type CommandId, type ProcessStatus } from '@/utils/scheduler'
import AsyncLock from 'async-lock'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

export type AbortOutcome = { ok: true } | { ok: false; code?: string; message: string }

/**
 * Owns the whole run lifecycle: expand-then-schedule, dispatch waves under a
 * serial lock, and all process state mutation. Single-run app — this composable
 * is instantiated once (modal setup), so the `execute:exited` subscription
 * below registers exactly one handler.
 */
export function useScheduler() {
  const { t } = useI18n()

  const processes = ref<Process[]>([])
  let isParallel = false
  const lock = new AsyncLock()

  const allCompleted = computed(() => processes.value.every(c => c.status === 'completed'))
  const noActive = computed(() => processes.value.every(c => !c.status.includes('ing')))

  function start(
    parallel: boolean,
    cmds: Array<Command>,
    groups: ReadonlyArray<storage.DriverGroup>
  ) {
    isParallel = parallel
    const expanded = expandIncompat(cmds, groups)
    processes.value = expanded.map(vals => ({
      command: { ...vals },
      status: status.Status.PENDING
    }))
    dispatch()
  }

  function dispatch() {
    return lock.acquire('executor', async () => {
      const statusById = new Map<CommandId, ProcessStatus>(
        processes.value.map(p => [p.command.id, p.status])
      )
      const wave = schedule(
        processes.value.map(p => p.command),
        statusById,
        isParallel ? Number.POSITIVE_INFINITY : 1
      )

      for (const cmd of wave) {
        const process = processes.value.find(p => p.command.id === cmd.id)!
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

  async function abort(process: Process): Promise<AbortOutcome> {
    await lock.acquire('executor', () => {
      if (process.status === status.Status.PENDING || process.status === status.Status.RUNNING) {
        process.status =
          process.procId == undefined || process.procId == ''
            ? status.Status.ABORTED
            : status.Status.ABORTING
      }
    })

    if (process.status !== status.Status.ABORTING) return { ok: true }

    try {
      await executor.Abort(process.procId!)
      return { ok: true }
    } catch (error) {
      const code = (error as { code?: string })?.code ?? ''
      if (code === 'errExecuteIdNotFound') {
        return { ok: false, code: 'errExecuteIdNotFound', message: decodeError(error, t) }
      }

      process.status = status.Status.ERRORED
      process.result = {
        lapse: -1,
        exitCode: -1,
        stdout: '',
        stderr: '',
        error: decodeError(error, t),
        aborted: false
      }
      return { ok: false, message: decodeError(error, t) }
    }
  }

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

    dispatch()
  })

  return { processes, start, abort, allCompleted, noActive }
}
