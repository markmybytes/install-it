import type { storage } from '@/wailsjs/go/models'
import * as libsysi from '@/wailsjs/go/sysinfo/SysInfo'

/**
 * Retrieves detailed hardware information from the system.
 */
export async function getHardware() {
  return libsysi.ResolvedHardware()
}

/**
 * Retrieves OS info (caption, display version, activation status).
 */
export async function getOSInfo() {
  return libsysi.OSInfo()
}

/**
 * Tests whether the given input string satisfies the specified match rule.
 */
export function testMatchRule(rule: storage.Rule, input: string) {
  input = rule.is_case_sensitive ? input : input.toLowerCase()
  const values = rule.is_case_sensitive ? rule.values : rule.values.map(v => v.toLowerCase())
  const hits = values.map((v: string): boolean => {
    switch (rule.operator) {
      case 'contain':
        return input.includes(v)
      case 'notContain':
        return !input.includes(v)
      case 'equal':
        return input === v
      case 'notEqual':
        return input !== v
      case 'regex': {
        try {
          return new RegExp(v, rule.is_case_sensitive ? '' : 'i').test(input)
        } catch {
          return false
        }
      }
      default:
        return false
    }
  })

  return rule.should_hit_all ? hits.every(Boolean) : hits.some(Boolean)
}

/**
 * Resolve a backend error (or plain string during migration) into a localized message.
 * The Wails rejection shape is `{code: string, params?: Record<string, unknown>}`;
 * anything else (string, null, undefined, native Error) falls back to the raw form.
 *
 * Pure function — no Vue reactivity, callable anywhere (setup, watchers, tests).
 * `t` is the caller's vue-i18n translation function.
 */
export function decodeError(
  err: unknown,
  t: (key: string, params?: Record<string, unknown>) => string
): string {
  const { code, params } = extractCode(err)
  return params ? t(code, params) : t(code)
}

function extractCode(err: unknown): { code: string; params?: Record<string, unknown> } {
  if (err && typeof err === 'object' && 'code' in err) {
    return err as { code: string; params?: Record<string, unknown> }
  }
  return { code: String(err) }
}
