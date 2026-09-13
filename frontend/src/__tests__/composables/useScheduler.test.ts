import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useScheduler } from '@/composables/useScheduler'
import type { Command, Process } from '@/types/execute'
import { status, storage } from '@/wailsjs/go/models'

const { runMock, abortMock, exitedHandler } = vi.hoisted(() => ({
  runMock: vi.fn(),
  abortMock: vi.fn(),
  exitedHandler: {
    current: undefined as undefined | ((id: string, result: unknown) => void)
  }
}))

vi.mock('@/wailsjs/go/execute/CommandExecutor', () => ({
  Run: runMock,
  Abort: abortMock
}))

vi.mock('@/wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((event: string, handler: (id: string, result: unknown) => void) => {
    if (event === 'execute:exited') exitedHandler.current = handler
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const cmd = (id: number, incompatibles: number[] = []): Command => ({
  id,
  groupName: 'g',
  config: { program: `p${id}`, options: [], minExeTime: 0, allowRtCodes: [], incompatibles }
})

const mutexGroup = () =>
  new storage.DriverGroup({
    id: 1,
    mutuallyExclusive: true,
    drivers: [{ id: 1 }, { id: 2 }, { id: 3 }]
  })

const exited = (procId: string, overrides: Partial<NonNullable<Process['result']>> = {}) => {
  exitedHandler.current?.(procId, {
    lapse: 5,
    exitCode: 0,
    stdout: '',
    stderr: '',
    error: '',
    aborted: false,
    ...overrides
  } as NonNullable<Process['result']>)
}

beforeEach(() => {
  vi.clearAllMocks()
  exitedHandler.current = undefined
})

describe('useScheduler', () => {
  it('start expands mutex groups and dispatches only the first wave', async () => {
    runMock.mockResolvedValue('proc-1')
    const scheduler = useScheduler()
    scheduler.start(true, [cmd(1), cmd(2), cmd(3)], [mutexGroup()])
    await flushPromises()

    expect(runMock).toHaveBeenCalledTimes(1)
    expect(scheduler.processes.value.map(p => p.status)).toEqual([
      status.Status.RUNNING,
      status.Status.PENDING,
      status.Status.PENDING
    ])
    expect(scheduler.processes.value[0].procId).toBe('proc-1')
    expect(scheduler.processes.value[0].command.config.incompatibles).toEqual([2, 3])
  })

  it('execute:exited with exitCode 0 completes the process and dispatches the next wave', async () => {
    runMock.mockResolvedValueOnce('proc-1').mockResolvedValueOnce('proc-2')
    const scheduler = useScheduler()
    scheduler.start(true, [cmd(1), cmd(2), cmd(3)], [mutexGroup()])
    await flushPromises()

    exited('proc-1', { exitCode: 0 })
    await flushPromises()

    expect(scheduler.processes.value[0].status).toBe(status.Status.COMPLETED)
    expect(runMock).toHaveBeenCalledTimes(2)
    expect(scheduler.processes.value[1].status).toBe(status.Status.RUNNING)
    expect(scheduler.processes.value[1].procId).toBe('proc-2')
  })

  it('allCompleted flips true once every process completes', async () => {
    runMock
      .mockResolvedValueOnce('proc-1')
      .mockResolvedValueOnce('proc-2')
      .mockResolvedValueOnce('proc-3')
    const scheduler = useScheduler()
    scheduler.start(true, [cmd(1), cmd(2), cmd(3)], [mutexGroup()])
    await flushPromises()
    expect(scheduler.allCompleted.value).toBe(false)

    exited('proc-1', { exitCode: 0 })
    await flushPromises()
    exited('proc-2', { exitCode: 0 })
    await flushPromises()
    exited('proc-3', { exitCode: 0 })
    await flushPromises()

    expect(scheduler.allCompleted.value).toBe(true)
    expect(scheduler.processes.value.every(p => p.status === status.Status.COMPLETED)).toBe(true)
  })

  it('abort on a pending process without procId marks it ABORTED immediately', async () => {
    runMock.mockResolvedValue('proc-1')
    const scheduler = useScheduler()
    scheduler.start(false, [cmd(1), cmd(2)], [])
    await flushPromises()

    expect(scheduler.processes.value[1].status).toBe(status.Status.PENDING)

    const outcome = await scheduler.abort(scheduler.processes.value[1])
    expect(outcome).toEqual({ ok: true })
    expect(scheduler.processes.value[1].status).toBe(status.Status.ABORTED)
    expect(abortMock).not.toHaveBeenCalled()
  })

  it('abort on a running process: Abort rejection marks ERRORED and returns ok:false', async () => {
    runMock.mockResolvedValue('proc-1')
    abortMock.mockRejectedValue({ code: 'errX' })
    const scheduler = useScheduler()
    scheduler.start(true, [cmd(1)], [])
    await flushPromises()

    expect(scheduler.processes.value[0].status).toBe(status.Status.RUNNING)

    const outcome = await scheduler.abort(scheduler.processes.value[0])
    expect(outcome.ok).toBe(false)
    expect(scheduler.processes.value[0].status).toBe(status.Status.ERRORED)
    expect(scheduler.processes.value[0].result).toMatchObject({
      lapse: -1,
      exitCode: -1,
      aborted: false
    })
    expect(scheduler.processes.value[0].result?.error).toBe('errX')
  })

  it('abort with errExecuteIdNotFound leaves status unchanged and returns the code', async () => {
    runMock.mockResolvedValue('proc-1')
    abortMock.mockRejectedValue({ code: 'errExecuteIdNotFound' })
    const scheduler = useScheduler()
    scheduler.start(true, [cmd(1)], [])
    await flushPromises()

    const outcome = await scheduler.abort(scheduler.processes.value[0])
    expect(outcome).toEqual({
      ok: false,
      code: 'errExecuteIdNotFound',
      message: 'errExecuteIdNotFound'
    })
    // The errExecuteIdNotFound branch performs no status write — status stays at
    // the ABORTING value set before `Abort` was called.
    expect(scheduler.processes.value[0].status).toBe(status.Status.ABORTING)
  })

  it('lock is released before Abort completes: a queued abort proceeds while Abort is in flight', async () => {
    runMock.mockResolvedValue('proc-1')
    // Abort never resolves — if the lock were held across it, the second abort would hang.
    abortMock.mockImplementation(() => new Promise(() => {}))
    const scheduler = useScheduler()
    scheduler.start(false, [cmd(1), cmd(2)], [])
    await flushPromises()

    expect(scheduler.processes.value[0].status).toBe(status.Status.RUNNING)
    expect(scheduler.processes.value[1].status).toBe(status.Status.PENDING)

    // Never settles; if the lock were held across Abort, the next abort would hang too.
    scheduler.abort(scheduler.processes.value[0])
    const second = await scheduler.abort(scheduler.processes.value[1])

    expect(second).toEqual({ ok: true })
    expect(scheduler.processes.value[1].status).toBe(status.Status.ABORTED)
    expect(abortMock).toHaveBeenCalledTimes(1)
  })
})
