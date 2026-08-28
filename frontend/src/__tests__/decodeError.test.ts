import { describe, it, expect, vi } from 'vitest'
import { decodeError } from '@/utils/index'

describe('decodeError', () => {
  it('extracts {code, params} from a structured rejection object', () => {
    const t = vi.fn((k, p) => p ? `${k}:${JSON.stringify(p)}` : k)
    const result = decodeError({ code: 'foo', params: { a: 1 } }, t)
    expect(result).toBe('foo:{"a":1}')
    expect(t).toHaveBeenCalledWith('foo', { a: 1 })
  })

  it('falls back to {code: String(err)} when err is a plain string', () => {
    const t = vi.fn((k) => k)
    const result = decodeError('boom', t)
    expect(result).toBe('boom')
  })

  it('handles null', () => {
    const t = vi.fn((k) => k)
    expect(decodeError(null, t)).toBe('null')
  })

  it('handles undefined', () => {
    const t = vi.fn((k) => k)
    expect(decodeError(undefined, t)).toBe('undefined')
  })

  it('calls t(code, params) when params present', () => {
    const t = vi.fn((k, p) => p ? `${k}:${JSON.stringify(p)}` : k)
    decodeError({ code: 'errCancelFailed', params: { name: 'foo' } }, t)
    expect(t).toHaveBeenCalledWith('errCancelFailed', { name: 'foo' })
  })

  it('calls t(code) when no params', () => {
    const t = vi.fn((k) => k)
    decodeError({ code: 'errSaveFailed' }, t)
    expect(t).toHaveBeenCalledWith('errSaveFailed')
  })

  it('resolves raw string via fallback to t(String(err))', () => {
    const t = vi.fn((k) => k)
    expect(decodeError('errFileNotFound', t)).toBe('errFileNotFound')
  })
})
