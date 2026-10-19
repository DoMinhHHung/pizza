import { beforeEach, describe, expect, it } from 'vitest'
import { clearTokens, getAccessToken, getRefreshToken, isLoggedIn, setTokens } from './auth-storage'

describe('auth-storage', () => {
  beforeEach(() => {
    clearTokens()
  })

  it('stores and reads tokens', () => {
    expect(isLoggedIn()).toBe(false)
    setTokens('access-1', 'refresh-1')
    expect(getAccessToken()).toBe('access-1')
    expect(getRefreshToken()).toBe('refresh-1')
    expect(isLoggedIn()).toBe(true)
  })

  it('clears tokens', () => {
    setTokens('a', 'r')
    clearTokens()
    expect(getAccessToken()).toBeNull()
    expect(getRefreshToken()).toBeNull()
    expect(isLoggedIn()).toBe(false)
  })
})
