import { describe, expect, it } from 'vitest'
import { isPortalAuthPath, PORTAL_LOGIN_URL } from '../portal-login'

describe('portal authentication entry policy', () => {
  it.each(['/sign-in', '/sign-in/', '/sign-up', '/login', '/register', '/otp', '/oauth/github', '/forgot-password', '/reset', '/user/reset'])('redirects authentication entry %s to the fixed portal', (path) => {
    expect(isPortalAuthPath(path)).toBe(true)
    expect(PORTAL_LOGIN_URL).toBe('https://api.passionapi.com/')
  })
  it.each(['/', '/pricing', '/pricing/model', '/about', '/dashboard', '/sign-in-other'])('preserves non-authentication page %s', (path) => {
    expect(isPortalAuthPath(path)).toBe(false)
  })
})
