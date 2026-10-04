import { describe, it, expect } from 'vitest'
import type { Command } from '@/types/execute'
import { schedule, expandIncompat, firstBlocker, type CommandId } from '@/utils/scheduler'
import { status, storage } from '@/wailsjs/go/models'

const cmd = (
  id: number | string,
  incompatibles: Array<number | string> = [],
  globalExclusive = false
): Command => ({
  id,
  groupName: 'g',
  config: {
    program: 'p',
    options: [],
    minExeTime: 0,
    allowRtCodes: [],
    incompatibles,
    globalExclusive
  }
})

const statuses = (...pairs: [CommandId, status.Status][]): Map<CommandId, status.Status> =>
  new Map(pairs)

const hasPair = (wave: ReadonlyArray<Command>, a: CommandId, b: CommandId) =>
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
      const wave = schedule(commands, snapshot, Infinity)
      for (const [a, b] of MUTEX_PAIRS) expect(hasPair(wave, a, b)).toBe(false)
    }
  })

  it('overlap invariant: global-exclusive driver never shares a wave with any other command', () => {
    const commands = expandIncompat([cmd(1, [], true), cmd(10), cmd(11), cmd('set_password')], [])

    const snapshots: Array<Map<CommandId, status.Status>> = [
      statuses(
        [1, status.Status.PENDING],
        [10, status.Status.PENDING],
        [11, status.Status.PENDING],
        ['set_password', status.Status.PENDING]
      ),
      statuses(
        [1, status.Status.RUNNING],
        [10, status.Status.PENDING],
        [11, status.Status.PENDING],
        ['set_password', status.Status.PENDING]
      ),
      statuses(
        [1, status.Status.COMPLETED],
        [10, status.Status.RUNNING],
        [11, status.Status.PENDING],
        ['set_password', status.Status.PENDING]
      )
    ]

    expect(schedule(commands, snapshots[0], Infinity).map(c => c.id)).toEqual([1])

    for (const snapshot of snapshots) {
      const wave = schedule(commands, snapshot, Infinity)
      expect(hasPair(wave, 1, 10)).toBe(false)
      expect(hasPair(wave, 1, 11)).toBe(false)
      expect(hasPair(wave, 1, 'set_password')).toBe(false)
    }
  })

  it('parallelLimit=1: global-exclusive command dispatches alone first', () => {
    const commands = expandIncompat([cmd(1, [], true), cmd(2)], [])
    const wave = schedule(
      commands,
      statuses([1, status.Status.PENDING], [2, status.Status.PENDING]),
      1
    )
    expect(wave.map(c => c.id)).toEqual([1])
  })

  it('parallelLimit=1: conflicting pendings stay out of the wave', () => {
    const commands = [cmd(1), cmd(2, [1]), cmd(3)]
    const allPending = statuses(
      [1, status.Status.PENDING],
      [2, status.Status.PENDING],
      [3, status.Status.PENDING]
    )
    const r = schedule(commands, allPending, 1)
    expect(r).toHaveLength(1)
    expect(r[0].id).toBe(1)
    expect(r.some(c => c.id === 2)).toBe(false)
    expect(r.some(c => c.id === 3)).toBe(false)

    const second = schedule(
      commands,
      statuses([1, status.Status.RUNNING], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      1
    )
    expect(second.map(c => c.id)).toEqual([3])
    expect(second.some(c => c.id === 2)).toBe(false)
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
    expect(r.map(c => c.id)).toEqual([1])
    expect(hasPair(r, 1, 2)).toBe(false)
  })

  it('aborting occupies a slot and blocks pending dependents', () => {
    const r = schedule(
      [cmd(1, [2]), cmd(2)],
      statuses([1, status.Status.PENDING], [2, status.Status.ABORTING]),
      Infinity
    )
    expect(r).toHaveLength(0)
  })

  it('errored does not occupy: subsequent cycle skips it and dispatches the rest', () => {
    const commands = expandIncompat([cmd(1, [2]), cmd(2), cmd(3)], [])
    const first = schedule(
      commands,
      statuses([1, status.Status.PENDING], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      Infinity
    )
    expect(first.map(c => c.id)).toEqual([1, 3])

    const second = schedule(
      commands,
      statuses([1, status.Status.ERRORED], [2, status.Status.PENDING], [3, status.Status.PENDING]),
      Infinity
    )
    expect(second.map(c => c.id)).toEqual([2, 3])
  })

  it('empty input returns an empty wave', () => {
    expect(schedule([], new Map(), Infinity)).toEqual([])
  })

  it('self-reference in incompatibles is not a self-block', () => {
    const r = schedule([cmd(1, [1])], statuses([1, status.Status.PENDING]), Infinity)
    expect(r.map(c => c.id)).toEqual([1])
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
    expect(r.map(c => c.id)).toEqual(['a', 'b', 'c'])
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

  it('global-exclusive driver in a mutually-exclusive group does not duplicate ME-pushed ids', () => {
    const group = new storage.DriverGroup({
      id: 1,
      name: 'gpu',
      type: storage.DriverType.DISPLAY,
      mutuallyExclusive: true,
      drivers: [{ id: 1 }, { id: 2 }]
    })
    const out = expandIncompat([cmd(1, [], true), cmd(2)], [group])
    expect(out[0].config.incompatibles).toEqual([2])
    expect(out[1].config.incompatibles).toEqual([1])
  })

  it('global-exclusive command lists every other id (incl. string ids) and excludes self', () => {
    const out = expandIncompat([cmd(1, [], true), cmd(2), cmd('set_password')], [])
    expect(out[0].config.incompatibles).toEqual([2, 'set_password'])
    expect(out[0].config.incompatibles).not.toContain(1)
  })

  it('global-exclusive mirror: every other command lists the exclusive id', () => {
    const out = expandIncompat([cmd(1, [], true), cmd(2), cmd('set_password')], [])
    expect(out[1].config.incompatibles).toEqual([1])
    expect(out[2].config.incompatibles).toEqual([1])
  })

  it('global-exclusive expansion skips ids already present (no duplicates)', () => {
    const out = expandIncompat([cmd(1, [2], true), cmd(2)], [])
    expect(out[0].config.incompatibles).toEqual([2])
  })

  it('does not mutate inputs when expanding a global-exclusive command', () => {
    const commands = [cmd(1, [9], true), cmd(2), cmd('set_password')]
    const commandsSnapshot = JSON.parse(JSON.stringify(commands))

    expandIncompat(commands, [])

    expect(JSON.parse(JSON.stringify(commands))).toEqual(commandsSnapshot)
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
