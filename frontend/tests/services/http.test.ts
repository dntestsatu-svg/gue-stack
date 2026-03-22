import { describe, expect, it } from 'vitest'
import { resolveApiBaseURL } from '@/services/http'

describe('resolveApiBaseURL', () => {
  it('keeps configured base URL when host already matches', () => {
    const result = resolveApiBaseURL('http://localhost:8080', {
      hostname: 'localhost',
      protocol: 'http:',
    })
    expect(result).toBe('http://localhost:8080')
  })

  it('normalizes localhost API host to 127 when app runs on 127', () => {
    const result = resolveApiBaseURL('http://localhost:8080', {
      hostname: '127.0.0.1',
      protocol: 'http:',
    })
    expect(result).toBe('http://127.0.0.1:8080')
  })

  it('uses browser host fallback when base URL is missing', () => {
    const result = resolveApiBaseURL(undefined, {
      hostname: '127.0.0.1',
      protocol: 'http:',
    })
    expect(result).toBe('http://127.0.0.1:8080')
  })
})

