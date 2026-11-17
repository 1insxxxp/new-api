export const PORTAL_LOGIN_URL = 'https://api.passionapi.com/'

export function isPortalAuthPath(pathname: string): boolean {
  return /^\/(?:sign-in|sign-up|login|register|forgot-password|reset|otp|oauth|user\/reset)(?:\/|$)/.test(
    pathname
  )
}
