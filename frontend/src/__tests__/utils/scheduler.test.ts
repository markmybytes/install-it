import { describe, it, expect } from 'vitest'
import type { Command } from '@/types/execute'
import { schedule, expandIncompat, firstBlocker, type CommandId } from '@/utils/scheduler'
import { status, storage } from '@/wailsjs/go/models'

const cmd = (id: number | string, incompatibles: number[] = []): Command => ({
  id,
  groupName: 'g',
  config: { program: 'p', options: [], minExeTime: 0, allowRtCodes: [], incompatibles }
})

const statuses = (...pairs: [CommandId, status.Status][]): Map<CommandId, status.Status> =>
  new Map(pairs)

const hasPair = (wave: ReadonlyArray<Command>, a: number, b: number) =>
  wave.some(c => c.id === a) && wave.some(c => c.id === b)

const MUTEX_PAIRS: Array<[number, number]> = [
  [1, 2],
  [1, 3],
  [2, 3]
]

describe('schedule', () => {
  it('overlap invariant: mutually-exclusive drivers never share a wave across snapshots', () => {
    const group = new storage.DriverGroup({
      id: 1,
      mutuallyExclusive: true,
      drivers: [{ id: 1 }, { id: 2 }, { id: 3 }]
    })
    const commands = expandIncompat([cmd(1), cmd(2), cmd(3), cmd(10), cmd(11)], [group])

    const snapshots: Array<Map<CommandId, status.Status>> = [
      statuses(
        [1, status.Status.PENDING],
        [2, status.Status.PENDING],
        [3, status.Status.PENDING],
        [10, status.Status.PENDING],
        [11, status.Status.PENDING]
      ),
      statuses(
        [1, status.Status.RUNNING],
        [2, status.Status.PENDING],
        [3, status.Status.PENDING],
        [10, status.Status.PENDING],
        [11, status.Status.PENDING]
      ),
      statuses(
        [1, status.Status.COMPLETED],
        [2, status.Status.RUNNING],
        [3, status.Status.PENDING],
        [10, status.Status.PENDING],
        [11, status.Status.PENDING]
      )
    ]

    for (const snapshot of snapshots) {
      const { wave } = schedule(commands, snapshot, Infinity)
      for (const [a, b] of MUTEX_PAIRS) expect(hasPair(wave, a, b)).toBe(false)
    }
  })

  it('parallelLimit=1 yields one wave slot; conflicting pendings land in blockers', () => {
    const commands = [cmd(1), cmd(2, [1]), cmd(3)]
    const allPending = statuses(
      [1, status.Status.PENDING],
      [2, status.Status.PENDING],
      [3, status.Status.PENDING]
    )
    const r = schedule(commands, allPending, 1)
    expect(r.wave).toHaveLength(1)
    expect(r.wave[0].id).toBe(1)
    expect(r.blockers.get(2)).toBe(1)
    expect(r.blockers.has(3)).toBe(false)

    const second = schedule(
      commands,
      statuses([1, status.Status.RUNNING], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      1
    )
    expect(second.wave.map(c => c.id)).toEqual([3])
    expect(second.blockers.get(2)).toBe(1)
  })

  it('symmetric one-directional edge: B blocks A after expandIncompat', () => {
    const c1 = cmd(1, [2])
    const c2 = cmd(2)
    const expanded = expandIncompat([c1, c2], [])
    expect(expanded[0].config.incompatibles).toEqual([2])
    expect(expanded[1].config.incompatibles).toEqual([1])

    const r = schedule(
      expanded,
      statuses([1, status.Status.PENDING], [2, status.Status.PENDING]),
      Infinity
    )
    expect(r.wave.map(c => c.id)).toEqual([1])
    expect(r.blockers.get(2)).toBe(1)
    expect(hasPair(r.wave, 1, 2)).toBe(false)
  })

  it('aborting occupies a slot and blocks pending dependents', () => {
    const r = schedule(
      [cmd(1, [2]), cmd(2)],
      statuses([1, status.Status.PENDING], [2, status.Status.ABORTING]),
      Infinity
    )
    expect(r.wave).toHaveLength(0)
    expect(r.blockers.get(1)).toBe(2)
  })

  it('errored does not occupy: subsequent cycle skips it and dispatches the rest', () => {
    const commands = expandIncompat([cmd(1, [2]), cmd(2), cmd(3)], [])
    const first = schedule(
      commands,
      statuses([1, status.Status.PENDING], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      Infinity
    )
    expect(first.wave.map(c => c.id)).toEqual([1, 3])
    expect(first.blockers.get(2)).toBe(1)

    const second = schedule(
      commands,
      statuses([1, status.Status.ERRORED], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      Infinity
    )
    expect(second.wave.map(c => c.id)).toEqual([2, 3])
    expect(second.blockers.size).toBe(0)
  })

  it('empty input returns empty wave and blockers', () => {
    const r = schedule([], new Map(), Infinity)
    expect(r.wave).toEqual([])
    expect(r.blockers.size).toBe(0)
  })

  it('self-reference in incompatibles is not a self-block', () => {
    const r = schedule([cmd(1, [1])], statuses([1, status.Status.PENDING]), Infinity)
    expect(r.wave.map(c => c.id)).toEqual([1])
    expect(r.blockers.size).toBe(0)
  })

  it('wave order preserves input order', () => {
    const commands = [cmd('a'), cmd('b'), cmd('c')]
    const r = schedule(
      commands,
      statuses(
        ['a', status.Status.PENDING],
        ['b', status.Status.PENDING],
        ['c', status.Status.PENDING]
      ),
      Infinity
    )
    expect(r.wave.map(c => c.id)).toEqual(['a', 'b', 'c'])
  })

  it('parallelLimit=0 throws RangeError', () => {
    expect(() => schedule([cmd(1)], new Map(), 0)).toThrow(RangeError)
  })
})

describe('expandIncompat', () => {
  it('mutually-exclusive DriverGroup expands each driver against the others', () => {
    const group = new storage.DriverGroup({
      id: 1,
      name: 'gpu',
      type: storage.DriverType.DISPLAY,
      mutuallyExclusive: true,
      drivers: [
        { id: 1, name: 'a' },
        { id: 2, name: 'b' },
        { id: 3, name: 'c' }
      ]
    })
    const out = expandIncompat([cmd(1), cmd(2), cmd(3)], [group])
    expect(out[0].config.incompatibles).toEqual([2, 3])
    expect(out[1].config.incompatibles).toEqual([1, 3])
    expect(out[2].config.incompatibles).toEqual([1, 2])
  })

  it('does not mutate input commands or groups', () => {
    const group = new storage.DriverGroup({
      id: 1,
      mutuallyExclusive: true,
      drivers: [{ id: 1 }, { id: 2 }]
    })
    const commands = [cmd(1, [9]), cmd(2)]
    const commandsSnapshot = JSON.parse(JSON.stringify(commands))
    const groupsSnapshot = JSON.parse(JSON.stringify([group]))

    expandIncompat(commands, [group])

    expect(JSON.parse(JSON.stringify(commands))).toEqual(commandsSnapshot)
    expect(JSON.parse(JSON.stringify([group]))).toEqual(groupsSnapshot)
  })

  it('firstBlocker returns the first occupied incompatible id, else null', () => {
    expect(firstBlocker(cmd(1, [2, 3]), new Set([3, 4]))).toBe(3)
    expect(firstBlocker(cmd(1, [2, 3]), new Set([9]))).toBeNull()
  })
})
