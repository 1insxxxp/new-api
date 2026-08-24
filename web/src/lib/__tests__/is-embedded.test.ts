/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { describe, expect, test } from 'vitest'

import { isEmbeddedWindow } from '../is-embedded'

describe('isEmbeddedWindow', () => {
  test('returns false when the current window is the top-level window', () => {
    const topWindow = {}

    expect(
      isEmbeddedWindow({
        self: topWindow,
        top: topWindow,
      })
    ).toBe(false)
  })

  test('returns true when the current window differs from the top-level window', () => {
    expect(
      isEmbeddedWindow({
        self: {},
        top: {},
      })
    ).toBe(true)
  })

  test('returns false when no window context is available', () => {
    expect(isEmbeddedWindow(undefined)).toBe(false)
  })
})
