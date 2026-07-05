import { vi } from 'vitest'

// Mock all Wails bindings before tests run.
// Source files import from '@/wailsjs/go/...' — the alias is resolved in vitest.config.ts.
// models.ts is NOT mocked here because it only defines TypeScript classes/enums with no
// window.go.* calls and must be importable as real values in tests.

vi.mock('@/wailsjs/go/main/App', () => ({
  AppVersion: vi.fn().mockResolvedValue('1.0.0'),
  AppBinaryType: vi.fn().mockResolvedValue('windows-x64'),
  AppConfigPath: vi.fn().mockResolvedValue('C:/conf'),
  AppDriverPath: vi.fn().mockResolvedValue('C:/drivers'),
  Cwd: vi.fn().mockResolvedValue('C:/'),
  PathExists: vi.fn().mockResolvedValue(true),
  ExecutableExists: vi.fn().mockResolvedValue(true),
  SelectFolder: vi.fn().mockResolvedValue('C:/selected'),
  SelectFile: vi.fn().mockResolvedValue('C:/selected/file.exe'),
  SetContext: vi.fn().mockResolvedValue(undefined),
  Update: vi.fn().mockResolvedValue(undefined),
  WebView2Version: vi.fn().mockResolvedValue('110.0.1'),
  WebView2Path: vi.fn().mockResolvedValue('C:/bin/WebView2')
}))

vi.mock('@/wailsjs/go/storage/AppSettingStorage', () => ({
  All: vi.fn(),
  Update: vi.fn()
}))

vi.mock('@/wailsjs/go/storage/DriverGroupStorage', () => ({
  All: vi.fn(),
  Get: vi.fn(),
  Add: vi.fn(),
  Update: vi.fn(),
  Remove: vi.fn(),
  MoveBehind: vi.fn(),
  IndexOf: vi.fn()
}))

vi.mock('@/wailsjs/go/storage/RuleSetStorage', () => ({
  All: vi.fn(),
  Get: vi.fn(),
  Add: vi.fn(),
  Update: vi.fn(),
  Remove: vi.fn()
}))

vi.mock('@/wailsjs/go/matching/Matcher', () => ({
  MatchedGroupIds: vi.fn()
}))

vi.mock('@/wailsjs/go/execute/CommandExecutor', () => ({
  Run: vi.fn(),
  RunAndOutput: vi.fn(),
  Abort: vi.fn()
}))

vi.mock('@/wailsjs/go/porter/Porter', () => ({
  Status: vi.fn(),
  Abort: vi.fn(),
  Progress: vi.fn(),
  Export: vi.fn(),
  ImportFromFile: vi.fn(),
  ImportFromURL: vi.fn()
}))

vi.mock('@/wailsjs/go/sysinfo/SysInfo', () => ({
  ResolvedHardware: vi.fn().mockResolvedValue({
    cpu: [],
    gpu: [],
    memory: [],
    motherboard: [],
    nic: [],
    storage: []
  })
}))

vi.mock('@/wailsjs/runtime/runtime', () => ({
  LogPrint: vi.fn(),
  LogTrace: vi.fn(),
  LogDebug: vi.fn(),
  LogInfo: vi.fn(),
  LogWarning: vi.fn(),
  LogError: vi.fn(),
  EventsOnMultiple: vi.fn(),
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
  EventsOffAll: vi.fn(),
  EventsOnce: vi.fn(),
  EventsEmit: vi.fn(),
  BrowserOpenURL: vi.fn()
}))
