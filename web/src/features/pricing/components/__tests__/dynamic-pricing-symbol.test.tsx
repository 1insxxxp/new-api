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
import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, test } from 'vitest'

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { DynamicPricingBreakdown } from '../dynamic-pricing-breakdown'

const initialState = useSystemConfigStore.getInitialState()
const billingExpr = 'tier("base", p * 3)'

function resetSystemCurrency(): void {
  useSystemConfigStore.setState(initialState, true)
  useSystemConfigStore.persist.clearStorage()
  localStorage.clear()
}

function expectRenderedPrice(expected: string, unexpected: string): void {
  expect(screen.getAllByText(expected).length).toBeGreaterThan(0)
  expect(screen.queryAllByText(unexpected)).toHaveLength(0)
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

describe('dynamic pricing breakdown symbols', () => {
  test('uses dollars for standard mode without changing the converted price', () => {
    render(
      <DynamicPricingBreakdown
        billingExpr={billingExpr}
        showRechargePrice={false}
      />
    )

    expectRenderedPrice('$21.0000', '¥21.0000')
  })

  test('uses yuan for recharge mode without changing the converted price', () => {
    render(
      <DynamicPricingBreakdown billingExpr={billingExpr} showRechargePrice />
    )

    expectRenderedPrice('¥21.0000', '$21.0000')
  })

  test('keeps the global currency symbol when pricing mode is omitted', () => {
    render(<DynamicPricingBreakdown billingExpr={billingExpr} />)

    expectRenderedPrice('¥21.0000', '$21.0000')
  })
})
