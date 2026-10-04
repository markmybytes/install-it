import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { flushPromises } from '@vue/test-utils'
import useDriverGroupStore from '@/stores/useDriverGroupStore'
import { storage } from '@/wailsjs/go/models'
import * as App from '@/wailsjs/go/main/App'

// ─── helpers ─────────────────────────────────────────────────────────────────

function makeDriver(id: string, path: string): storage.Driver {
  return new storage.Driver({
    id,
    name: id,
    type: storage.DriverType.NETWORK,
    path,
    flags: [],
    minExeTime: 0,
    allowRtCodes: [],
    incompatibles: [],
    globalExclusive: false
  })
}

function makeGroup(id: string, drivers: storage.Driver[]): storage.DriverGroup {
  return new storage.DriverGroup({
    id,
    name: id,
    type: storage.DriverType.NETWORK,
    mutuallyExclusive: false,
    drivers
  })
}

// ─── tests ───────────────────────────────────────────────────────────────────

describe('useDriverGroupStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Safe default: all executables exist
    vi.mocked(App.ExecutableExists).mockResolvedValue(true)
  })

  it('groups initializes as an empty array', () => {
    const store = useDriverGroupStore()
    expect(store.groups).toEqual([])
  })

  it('notFoundDrivers initializes as an empty array', async () => {
    const store = useDriverGroupStore()
    await flushPromises()
    expect(store.notFoundDrivers).toEqual([])
  })

  it('notFoundDrivers is populated when ExecutableExists returns false', async () => {
    vi.mocked(App.ExecutableExists).mockResolvedValue(false)
    const store = useDriverGroupStore()
    const driver = makeDriver('driver-1', 'C:/net.exe')
    const group = makeGroup('group-1', [driver])

    store.groups = [group]
    await flushPromises()

    expect(store.notFoundDrivers).toContain('driver-1')
  })

  it('notFoundDrivers does not include drivers whose executable is found', async () => {
    vi.mocked(App.ExecutableExists).mockResolvedValue(true)
    const store = useDriverGroupStore()
    const driver = makeDriver('driver-ok', 'C:/display.exe')
    const group = makeGroup('group-ok', [driver])

    store.groups = [group]
    await flushPromises()

    expect(store.notFoundDrivers).not.toContain('driver-ok')
    expect(store.notFoundDrivers).toHaveLength(0)
  })

  it('notFoundDrivers only lists missing drivers when some exist and some do not', async () => {
    vi.mocked(App.ExecutableExists)
      .mockResolvedValueOnce(true) // driver-exists  → found
      .mockResolvedValueOnce(false) // driver-missing → not found

    const store = useDriverGroupStore()
    const existing = makeDriver('driver-exists', 'C:/exists.exe')
    const missing = makeDriver('driver-missing', 'C:/missing.exe')
    const group = makeGroup('group-mixed', [existing, missing])

    store.groups = [group]
    await flushPromises()

    expect(store.notFoundDrivers).not.toContain('driver-exists')
    expect(store.notFoundDrivers).toContain('driver-missing')
  })

  it('isAllDriversExist returns false when a driver is missing', async () => {
    vi.mocked(App.ExecutableExists).mockResolvedValue(false)
    const store = useDriverGroupStore()
    const driver = makeDriver('driver-missing', 'C:/missing.exe')
    const group = makeGroup('group-missing', [driver])

    store.groups = [group]
    await flushPromises()

    expect(store.isAllDriversExist(group)).toBe(false)
  })

  it('isAllDriversExist returns true when all drivers exist', async () => {
    vi.mocked(App.ExecutableExists).mockResolvedValue(true)
    const store = useDriverGroupStore()
    const driver = makeDriver('driver-ok', 'C:/ok.exe')
    const group = makeGroup('group-all-ok', [driver])

    store.groups = [group]
    await flushPromises()

    expect(store.isAllDriversExist(group)).toBe(true)
  })

  it('isAllDriversExist returns true for a group with no drivers', () => {
    const store = useDriverGroupStore()
    const emptyGroup = makeGroup('group-empty', [])
    // No async work needed — every() on an empty array is vacuously true
    expect(store.isAllDriversExist(emptyGroup)).toBe(true)
  })
})
