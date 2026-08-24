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
import { afterEach, beforeEach, describe, expect, test } from 'vitest'

import { formatCurrencyFromUSD } from '@/lib/currency'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

const initialState = useSystemConfigStore.getInitialState()

function resetSystemCurrency(): void {
  useSystemConfigStore.setState(initialState, true)
  useSystemConfigStore.persist.clearStorage()
  localStorage.clear()
}

beforeEach(resetSystemCurrency)
afterEach(resetSystemCurrency)

describe('currency symbol override', () => {
  test('uses a dollar override after converting USD to configured CNY', () => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 7,
      },
    })

    expect(
      formatCurrencyFromUSD(3, {
        locale: 'en-US',
        abbreviate: false,
        symbolOverride: '$',
      })
    ).toBe('$21')
  })

  test('uses a yuan override without changing the configured USD value', () => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'USD',
        usdExchangeRate: 1,
      },
    })

    expect(
      formatCurrencyFromUSD(3, {
        locale: 'en-US',
        abbreviate: false,
        symbolOverride: '¥',
      })
    ).toBe('¥3')
  })
})
