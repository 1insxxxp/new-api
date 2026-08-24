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

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import type { PricingModel } from '../../types'
import { formatDynamicUnitPrice } from '../dynamic-price'
import {
  formatFixedPrice,
  formatGroupPrice,
  formatPrice,
  formatRequestPrice,
} from '../price'

const initialState = useSystemConfigStore.getInitialState()

const tokenModel: PricingModel = {
  id: 1,
  model_name: 'token-model',
  quota_type: 0,
  model_ratio: 1.5,
  completion_ratio: 5,
  enable_groups: ['default'],
  group_ratio: { default: 1 },
}

const requestModel: PricingModel = {
  id: 2,
  model_name: 'request-model',
  quota_type: 1,
  model_ratio: 0,
  completion_ratio: 0,
  model_price: 2,
  enable_groups: ['default'],
  group_ratio: { default: 1 },
}

function resetSystemCurrency(): void {
  useSystemConfigStore.setState(initialState, true)
  useSystemConfigStore.persist.clearStorage()
  localStorage.clear()
}

beforeEach(() => {
  resetSystemCurrency()
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'CNY',
      usdExchangeRate: 7,
    },
  })
})

afterEach(resetSystemCurrency)

describe('pricing display mode symbols', () => {
  test('formatPrice uses dollars for standard values and yuan for recharge values', () => {
    expect(formatPrice(tokenModel, 'input', 'M', false, 4, 7)).toBe('$21')
    expect(formatPrice(tokenModel, 'input', 'M', true, 4, 7)).toBe('¥12')
  })

  test('formatGroupPrice uses the symbol selected by display mode', () => {
    const groupRatio = { premium: 2 }

    expect(
      formatGroupPrice(
        tokenModel,
        'premium',
        'input',
        'M',
        false,
        4,
        7,
        groupRatio
      )
    ).toBe('$42')
    expect(
      formatGroupPrice(
        tokenModel,
        'premium',
        'input',
        'M',
        true,
        4,
        7,
        groupRatio
      )
    ).toBe('¥24')
  })

  test('formatRequestPrice uses dollars for standard values and yuan for recharge values', () => {
    expect(formatRequestPrice(requestModel, false, 4, 7)).toBe('$14')
    expect(formatRequestPrice(requestModel, true, 4, 7)).toBe('¥8')
  })

  test('formatFixedPrice uses the symbol selected by display mode', () => {
    const groupRatio = { premium: 1.5 }

    expect(
      formatFixedPrice(requestModel, 'premium', false, 4, 7, groupRatio)
    ).toBe('$21')
    expect(
      formatFixedPrice(requestModel, 'premium', true, 4, 7, groupRatio)
    ).toBe('¥12')
  })

  test('formatDynamicUnitPrice uses dollars for standard values and yuan for recharge values', () => {
    expect(
      formatDynamicUnitPrice(3, {
        tokenUnit: 'M',
        showRechargePrice: false,
        priceRate: 4,
        usdExchangeRate: 7,
      })
    ).toBe('$21')
    expect(
      formatDynamicUnitPrice(3, {
        tokenUnit: 'M',
        showRechargePrice: true,
        priceRate: 4,
        usdExchangeRate: 7,
      })
    ).toBe('¥12')
  })
})
