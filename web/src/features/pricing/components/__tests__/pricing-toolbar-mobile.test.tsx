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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test, vi } from 'vitest'

import { VIEW_MODES } from '../../constants'
import { PricingToolbar } from '../pricing-toolbar'

vi.mock('../pricing-sidebar', () => ({
  PricingSidebar: () => null,
}))

function renderToolbar() {
  const onRechargePriceChange = vi.fn()

  render(
    <PricingToolbar
      filteredCount={3}
      totalCount={3}
      sortBy='name'
      onSortChange={vi.fn()}
      tokenUnit='M'
      onTokenUnitChange={vi.fn()}
      showRechargePrice={false}
      onRechargePriceChange={onRechargePriceChange}
      viewMode={VIEW_MODES.CARD}
      onViewModeChange={vi.fn()}
      quotaTypeFilter='all'
      endpointTypeFilter='all'
      vendorFilter='all'
      groupFilter='all'
      tagFilter='all'
      onQuotaTypeChange={vi.fn()}
      onEndpointTypeChange={vi.fn()}
      onVendorChange={vi.fn()}
      onGroupChange={vi.fn()}
      onTagChange={vi.fn()}
      vendors={[]}
      groups={[]}
      tags={[]}
      models={[]}
      hasActiveFilters={false}
      activeFilterCount={0}
      onClearFilters={vi.fn()}
    />
  )

  return { onRechargePriceChange }
}

describe('pricing toolbar mobile controls', () => {
  test('renders a mobile control row wired to the pricing callbacks', async () => {
    const user = userEvent.setup()
    const { onRechargePriceChange } = renderToolbar()
    const mobileControls = screen.getByTestId('mobile-pricing-controls')

    expect(mobileControls).toHaveClass('sm:hidden')
    expect(
      within(mobileControls).getByRole('group', {
        name: 'Price display mode',
      })
    ).toBeInTheDocument()
    expect(
      within(mobileControls).getByRole('group', { name: 'Token unit' })
    ).toBeInTheDocument()
    expect(
      screen.getAllByRole('group', { name: 'Price display mode' })
    ).toHaveLength(2)
    expect(screen.getAllByRole('group', { name: 'Token unit' })).toHaveLength(2)

    await user.click(
      within(mobileControls).getByRole('button', { name: 'Recharge' })
    )

    expect(onRechargePriceChange).toHaveBeenCalledWith(true)
  })
})
