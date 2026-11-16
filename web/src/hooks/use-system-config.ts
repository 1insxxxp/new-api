/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useCallback } from 'react'

import { useTheme } from '@/context/theme-provider'
import {
  DEFAULT_LOGO,
  DEFAULT_LOGO_DARK,
  DEFAULT_LOGO_LIGHT,
} from '@/lib/constants'
import { applyFaviconToDom } from '@/lib/dom-utils'
import { ensureStatus } from '@/lib/status-query'
import { useSystemConfigStore } from '@/stores/system-config-store'

interface UseSystemConfigOptions {
  /** Automatically fetch config from backend (use only in root component) */
  autoLoad?: boolean
}

export function resolveSystemLogo(
  logo: string,
  resolvedTheme: 'light' | 'dark'
): string {
  if (logo !== DEFAULT_LOGO) return logo
  return resolvedTheme === 'dark' ? DEFAULT_LOGO_DARK : DEFAULT_LOGO_LIGHT
}

/** Preload an image, returning a cleanup that detaches the pending handlers. */
function preloadImage(
  src: string,
  onLoad: () => void,
  onError: () => void
): () => void {
  const img = new Image()
  img.onload = onLoad
  img.onerror = onError
  img.src = src

  return () => {
    img.onload = null
    img.onerror = null
  }
}

/**
 * System configuration hook with auto-loading and logo preloading
 *
 * @example
 * // Root component - auto-load from backend
 * useSystemConfig({ autoLoad: true })
 *
 * @example
 * // Other components - use cached config
 * const { systemName, logo, loading } = useSystemConfig()
 */
export function useSystemConfig(options: UseSystemConfigOptions = {}) {
  const { autoLoad = false } = options
  const { resolvedTheme } = useTheme()
  const queryClient = useQueryClient()
  const { config, loading, loadedLogoUrl, setLoadedLogoUrl, setLoading } =
    useSystemConfigStore()

  // Load config from backend via the shared `/api/status` cache.
  // `ensureStatus` writes the mapped config into this store itself, so there is
  // no second request and no second mapping path here.
  const loadConfig = useCallback(async () => {
    try {
      setLoading(true)
      await ensureStatus(queryClient)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to load system config:', error)
    } finally {
      setLoading(false)
    }
  }, [queryClient, setLoading])

  useEffect(() => {
    if (autoLoad) loadConfig()
  }, [autoLoad, loadConfig])

  // Preload logo image when URL changes
  const resolvedLogo = resolveSystemLogo(config.logo, resolvedTheme)

  useEffect(() => {
    const logo = resolvedLogo

    // Skip if logo is already loaded
    if (!logo || logo === loadedLogoUrl) return

    // Preload new logo
    return preloadImage(
      logo,
      () => {
        setLoadedLogoUrl(logo)
        applyFaviconToDom(logo)
      },
      () => {
        if (logo !== DEFAULT_LOGO_LIGHT && logo !== DEFAULT_LOGO_DARK) {
          // eslint-disable-next-line no-console
          console.error('Failed to load logo:', logo)
        }
        // Mark as loaded even on error to prevent infinite retry
        setLoadedLogoUrl(logo)
      }
    )
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resolvedLogo, loadedLogoUrl, setLoadedLogoUrl])

  return {
    ...config,
    logo: resolvedLogo,
    loading,
    logoLoaded: resolvedLogo === loadedLogoUrl && !!loadedLogoUrl,
  }
}
