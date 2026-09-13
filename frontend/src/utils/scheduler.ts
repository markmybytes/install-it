import type { Command } from '@/types/execute'
import { status } from '@/wailsjs/go/models'
import type { storage } from '@/wailsjs/go/models'

export type CommandId = string | number
export type ProcessStatus = status.Status

interface ScheduleResult {
  /** Commands to dispatch this cycle, in input order, max `parallelLimit`. */
  wave: ReadonlyArray<Command>
  /** Pending commands that are blocked, mapped to the id of the blocker. */
  blockers: ReadonlyMap<CommandId, CommandId>
}

/**
 * First id in `command.config.incompatibles` that is currently occupied, else null.
 */
export function firstBlocker(command: Command, occupied: ReadonlySet<CommandId>): CommandId | null {
  for (const id of command.config.incompatibles) {
    if (occupied.has(id)) return id
  }
  return null
}

/**
 * Deep-ish copy of `commands` with expanded `config.incompatibles`:
 * 1. mutually-exclusive groups block every sibling driver in the group;
 * 2. symmetric mirror — A lists B ⇒ B lists A (when B is a known command).
 * Input commands and groups are never mutated.
 * Precondition: command ids must be unique — duplicate ids make the mirror map last-wins.
 */
export function expandIncompat(
  commands: ReadonlyArray<Command>,
  groups: ReadonlyArray<storage.DriverGroup>
): Command[] {
  const expanded = commands.map(cmd => ({
    ...cmd,
    config: { ...cmd.config, incompatibles: [...cmd.config.incompatibles] }
  }))
  const byId = new Map<CommandId, Command>(expanded.map(cmd => [cmd.id, cmd]))

  for (const cmd of expanded) {
    const list = cmd.config.incompatibles
    const seen = new Set(list)
    for (const group of groups) {
      if (!group.mutuallyExclusive) continue
      const groupIds = group.drivers.map(d => d.id)
      if (!groupIds.includes(cmd.id as number)) continue
      for (const id of groupIds) {
        if (id !== cmd.id && !seen.has(id)) {
          seen.add(id)
          list.push(id)
        }
      }
    }
  }

  for (const cmd of expanded) {
    for (const b of cmd.config.incompatibles) {
      if (cmd.id === b) continue
      const target = byId.get(b)
      if (!target) continue
      const mirrored = target.config.incompatibles as Array<CommandId>
      if (!mirrored.includes(cmd.id)) mirrored.push(cmd.id)
    }
  }

  return expanded
}

/**
 * Pick the next dispatch wave: pending commands with no occupied incompatible
 * id, up to `parallelLimit`. Commands blocked by an occupied id go to `blockers`.
 * Pure — no input mutation, no side effects.
 * Precondition: command ids must be unique — duplicate ids would co-dispatch in the same wave.
 */
export function schedule(
  commands: ReadonlyArray<Command>,
  statusById: ReadonlyMap<CommandId, ProcessStatus>,
  parallelLimit: number
): ScheduleResult {
  if (parallelLimit < 1) throw new RangeError('parallelLimit must be >= 1')

  const occupied = new Set<CommandId>()
  for (const [id, st] of statusById) {
    if (st === status.Status.RUNNING || st === status.Status.ABORTING) occupied.add(id)
  }

  const wave: Command[] = []
  const blockers = new Map<CommandId, CommandId>()

  for (const cmd of commands) {
    if (statusById.get(cmd.id) !== status.Status.PENDING) continue
    const blocker = firstBlocker(cmd, occupied)
    if (blocker !== null) {
      blockers.set(cmd.id, blocker)
      continue
    }
    if (wave.length < parallelLimit) {
      wave.push(cmd)
      occupied.add(cmd.id)
    }
  }

  return { wave, blockers }
}
