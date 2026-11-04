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
import { filterAndSortModels, sortModels } from '../filters'
import { getDisplayPricingGroup, isTokenBasedModel } from '../model-helpers'
import {
  formatFixedPrice,
  formatGroupPrice,
  formatPrice,
  formatRequestPrice,
} from '../price'

const initialState = useSystemConfigStore.getInitialState()
const model: PricingModel = {
  id: 1,
  model_name: 'shared',
  quota_type: 0,
  model_ratio: 7,
  completion_ratio: 3,
  cache_ratio: 0.2,
  enable_groups: ['normal', 'category', 'fixed', 'zero-input'],
  group_ratio: { normal: 2, category: 1, fixed: 1, 'zero-input': 1 },
  group_pricing: {
    category: {
      billing_mode: 'token',
      input_price: 0.000006,
      output_price: 0.000018,
    },
    fixed: { billing_mode: 'per_request', per_request_price: 0.6 },
    'zero-input': {
      billing_mode: 'token',
      input_price: 0,
      output_price: 0.000003,
      cache_read_price: 0,
      image_input_price: 0,
      image_output_price: 0.000009,
    },
  },
}

const dynamicModel: PricingModel = {
  ...model,
  billing_mode: 'tiered_expr',
  billing_expr:
    'len <= 200000 ? tier("base", p * 3 + c * 9) : tier("long", p * 6 + c * 18)',
  enable_groups: ['normal', 'category'],
  group_pricing: {
    category: {
      billing_mode: 'token',
      input_price: 0.000008,
      output_price: 0.000024,
    },
  },
}

beforeEach(() => {
  useSystemConfigStore.setState(initialState, true)
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'USD',
      usdExchangeRate: 1,
    },
  })
})
afterEach(() => {
  useSystemConfigStore.setState(initialState, true)
  useSystemConfigStore.persist.clearStorage()
})

describe('inherited group display pricing', () => {
  test('selects the ordinary group when its dynamic first tier costs less than the inherited quote', () => {
    expect(getDisplayPricingGroup(dynamicModel)).toBe('normal')
    expect(getDisplayPricingGroup(dynamicModel, 'category')).toBe('category')
  })

  test('sorts dynamic first-tier prices with ordinary group multipliers', () => {
    const inherited = {
      ...dynamicModel,
      model_name: 'inherited',
      enable_groups: ['category'],
    }
    const ordinary = {
      ...dynamicModel,
      model_name: 'dynamic',
      enable_groups: ['normal'],
      group_pricing: undefined,
    }
    expect(
      sortModels([inherited, ordinary], 'price-low').map(
        (row) => row.model_name
      )
    ).toEqual(['dynamic', 'inherited'])
    expect(
      sortModels([inherited, ordinary], 'price-high').map(
        (row) => row.model_name
      )
    ).toEqual(['inherited', 'dynamic'])
  })

  test.each(['', 'p * p', 'tier("overflow", p * 1e309)'])(
    'ignores unknown or non-finite dynamic pricing when selecting a group: %s',
    (billing_expr) => {
      expect(
        getDisplayPricingGroup({
          ...dynamicModel,
          model_ratio: 0,
          billing_expr,
        })
      ).toBe('category')
    }
  )

  test.each(['price-low', 'price-high'])(
    'puts unknown expressions and non-finite prices last for %s',
    (sortBy) => {
      const unknown = {
        ...dynamicModel,
        model_name: 'unknown',
        billing_expr: 'p * p',
        model_ratio: 0,
        group_pricing: undefined,
        enable_groups: ['normal'],
      }
      const ordinary = {
        ...model,
        group_pricing: undefined,
        enable_groups: ['normal'],
      }
      const invalid = [
        Number.NaN,
        Number.POSITIVE_INFINITY,
        Number.NEGATIVE_INFINITY,
      ].map((model_ratio, index) => ({
        ...ordinary,
        model_ratio,
        model_name: `invalid-${index}`,
      }))
      const cheaper = { ...ordinary, model_name: 'cheaper', model_ratio: 1 }
      const expensive = { ...ordinary, model_name: 'expensive', model_ratio: 2 }
      const rows = sortModels(
        [unknown, invalid[0], expensive, invalid[1], cheaper, invalid[2]],
        sortBy
      )
      expect(rows.slice(0, 2).map((row) => row.model_name)).toEqual(
        sortBy === 'price-low'
          ? ['cheaper', 'expensive']
          : ['expensive', 'cheaper']
      )
      expect(rows.slice(2)).toEqual([unknown, ...invalid])
    }
  )

  test.each([
    { billing_expr: 'tier("free-input", p * 0 + c * 9)', rate: 2 },
    { billing_expr: 'tier("base", p * 3 + c * 9)', rate: 0 },
  ])(
    'preserves zero dynamic input or group multiplier: %j',
    ({ billing_expr, rate }) => {
      expect(
        getDisplayPricingGroup({
          ...dynamicModel,
          billing_expr,
          group_ratio: { normal: rate, category: 1 },
        })
      ).toBe('normal')
    }
  )

  test('sorts ordinary and inherited input prices in the same effective units', () => {
    const ordinary = {
      ...model,
      model_name: 'ordinary',
      group_pricing: undefined,
      model_ratio: 2,
      enable_groups: ['category'],
      group_ratio: { category: 2 },
    }
    const inherited = { ...model, enable_groups: ['category'] }
    // Ordinary input is $8/M (ratio 2 * $2/M * group rate 2), inherited is $6/M.
    expect(sortModels([ordinary, inherited], 'price-low', 'category')).toEqual([
      inherited,
      ordinary,
    ])
    expect(sortModels([ordinary, inherited], 'price-high')).toEqual([
      ordinary,
      inherited,
    ])
    expect(
      sortModels(
        [{ ...ordinary, group_ratio: { category: 0 } }, inherited],
        'price-low'
      )[0].model_name
    ).toBe('ordinary')
  })

  test('selects source prices without changing ordinary group pricing or input', () => {
    expect(formatPrice(model, 'input', 'M', false, 1, 1, 'normal')).toBe('$28')
    expect(formatPrice(model, 'input', 'M', false, 1, 1, 'category')).toBe('$6')
    expect(
      formatGroupPrice(model, 'category', 'output', 'M', false, 1, 1, {
        category: 99,
      })
    ).toBe('$18')
    expect(
      formatGroupPrice(
        model,
        'category',
        'cache',
        'M',
        false,
        1,
        1,
        model.group_ratio ?? {}
      )
    ).toBe('-')
    expect(model.model_ratio).toBe(7)
    expect(model.cache_ratio).toBe(0.2)
  })

  test('preserves free input and nonzero output without ratio division', () => {
    expect(formatPrice(model, 'input', 'M', false, 1, 1, 'zero-input')).toBe(
      '$0'
    )
    expect(formatPrice(model, 'output', 'M', false, 1, 1, 'zero-input')).toBe(
      '$3'
    )
    expect(formatPrice(model, 'cache', 'M', false, 1, 1, 'zero-input')).toBe(
      '$0'
    )
    expect(formatPrice(model, 'image', 'M', false, 1, 1, 'zero-input')).toBe(
      '$0'
    )
    expect(
      formatPrice(model, 'image_output', 'M', false, 1, 1, 'zero-input')
    ).toBe('$9')
  })

  test('resolves inherited fixed mode and ignores New group multipliers', () => {
    expect(isTokenBasedModel(model, 'fixed')).toBe(false)
    expect(formatFixedPrice(model, 'fixed', false, 1, 1, { fixed: 50 })).toBe(
      '$0.6'
    )
    expect(formatRequestPrice(model, false, 1, 1, 'fixed')).toBe('$0.6')
  })

  test('shows category-only models once using their inherited price', () => {
    const only = { ...model, enable_groups: ['category'] }
    expect(formatPrice(only, 'input', 'M')).toBe('$6')
    const rows = filterAndSortModels([only], {
      search: '',
      vendor: 'all',
      group: 'all',
      quotaType: 'all',
      endpointType: 'all',
      tag: 'all',
      sortBy: 'name',
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].model_name).toBe('shared')
  })
})
