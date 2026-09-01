import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { readdirSync, statSync } from 'node:fs'
import { join, sep } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(here, '..', '..', '..')

const enPath = resolve(here, '../i18n/en.json')
const zhPath = resolve(here, '../i18n/zh_Hant_HK.json')
const en = JSON.parse(readFileSync(enPath, 'utf-8')) as Record<string, string>
const zh = JSON.parse(readFileSync(zhPath, 'utf-8')) as Record<string, string>

/**
 * Walk a directory recursively and return all files matching the predicate.
 */
function walkGoFiles(root: string): string[] {
  const results: string[] = []
  for (const entry of readdirSync(root)) {
    const full = join(root, entry)
    const st = statSync(full)
    if (st.isDirectory()) {
      results.push(...walkGoFiles(full))
    } else if (st.isFile() && entry.endsWith('.go') && !entry.endsWith('_test.go')) {
      results.push(full)
    }
  }
  return results
}

/**
 * Extract every `errcode.New("...")` / `errcode.Newf("...", ...)` first-arg
 * literal across pkg/ + app.go + main.go.
 */
function collectErrcodeCodes(): Set<string> {
  const goFiles = [
    resolve(repoRoot, 'app.go'),
    resolve(repoRoot, 'main.go'),
    ...walkGoFiles(resolve(repoRoot, 'pkg')).filter(p => !p.includes(`pkg${sep}errcode${sep}`))
  ]
  const codes = new Set<string>()
  const re = /errcode\.Newf?\("([^"]+)"/g
  for (const file of goFiles) {
    try {
      const content = readFileSync(file, 'utf-8')
      let m: RegExpExecArray | null
      while ((m = re.exec(content)) !== null) {
        codes.add(m[1])
      }
    } catch {
      // skip unreadable files
    }
  }
  return codes
}

describe('i18n completeness', () => {
  it('zh keys are a subset of en keys (no missing en entries)', () => {
    const missing = Object.keys(zh).filter(k => !(k in en))
    expect(missing).toEqual([])
  })

  it('en keys are a subset of zh keys (1:1 mirror)', () => {
    const missing = Object.keys(en).filter(k => !(k in zh))
    expect(missing).toEqual([])
  })

  it('all keys have non-empty values', () => {
    for (const [key, value] of Object.entries(en)) {
      expect(value, `en[${key}]`).toBeTruthy()
    }
    for (const [key, value] of Object.entries(zh)) {
      expect(value, `zh[${key}]`).toBeTruthy()
    }
  })

  describe('errcode.New/Newf coverage', () => {
    const codes = collectErrcodeCodes()

    it('discovers errcode.New/Newf codes in backend', () => {
      // sanity: at least one code must be discovered
      expect(codes.size).toBeGreaterThan(0)
    })

    it('every errcode.New/Newf code has an en.json entry', () => {
      const missing = [...codes].filter(c => !(c in en))
      expect(missing, `missing en.json keys: ${missing.join(', ')}`).toEqual([])
    })

    it('every errcode.New/Newf code has a zh_Hant_HK.json entry', () => {
      const missing = [...codes].filter(c => !(c in zh))
      expect(missing, `missing zh.json keys: ${missing.join(', ')}`).toEqual([])
    })
  })
})
