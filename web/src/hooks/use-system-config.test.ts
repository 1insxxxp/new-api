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
import { describe, expect, test } from 'vitest'

import { resolveSystemLogo } from '@/hooks/use-system-config'
import {
  DEFAULT_LOGO,
  DEFAULT_LOGO_DARK,
  DEFAULT_LOGO_LIGHT,
} from '@/lib/constants'

describe('resolveSystemLogo', () => {
  test('uses the light bundled logo for the default logo in light theme', () => {
    expect(resolveSystemLogo(DEFAULT_LOGO, 'light')).toBe(DEFAULT_LOGO_LIGHT)
  })

  test('uses the dark bundled logo for the default logo in dark theme', () => {
    expect(resolveSystemLogo(DEFAULT_LOGO, 'dark')).toBe(DEFAULT_LOGO_DARK)
  })

  test('keeps an administrator-configured logo in every theme', () => {
    const customLogo = 'https://cdn.example.com/custom.png'

    expect(resolveSystemLogo(customLogo, 'light')).toBe(customLogo)
    expect(resolveSystemLogo(customLogo, 'dark')).toBe(customLogo)
  })
})
