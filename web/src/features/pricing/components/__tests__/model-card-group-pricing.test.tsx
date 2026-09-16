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
import { describe, expect, test, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelCard } from '../model-card'

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))

const model: PricingModel = {
  id: 1,
  model_name: 'shared-model',
  quota_type: 0,
  model_ratio: 1.5,
  completion_ratio: 3,
  enable_groups: ['request-group', 'token-group'],
  group_ratio: { 'token-group': 1, 'request-group': 1 },
  group_prices: { 'request-group': 0.15 },
}

describe('model card group pricing', () => {
  test('switching groups updates the group label and billing mode together', () => {
    const { rerender } = render(
      <ModelCard model={model} selectedGroup='token-group' onClick={vi.fn()} />
    )

    expect(screen.getByText('token-group')).toBeInTheDocument()
    expect(screen.queryByText('request-group')).not.toBeInTheDocument()
    expect(screen.getByText('Token-based')).toBeInTheDocument()
    expect(screen.queryByText('/ request')).not.toBeInTheDocument()

    rerender(
      <ModelCard
        model={model}
        selectedGroup='request-group'
        onClick={vi.fn()}
      />
    )

    expect(screen.getByText('request-group')).toBeInTheDocument()
    expect(screen.queryByText('token-group')).not.toBeInTheDocument()
    expect(screen.getByText('Per Request')).toBeInTheDocument()
    expect(screen.getByText('/ request')).toBeInTheDocument()
  })

  test('the all-groups view labels the group matching the displayed billing mode', () => {
    render(<ModelCard model={model} onClick={vi.fn()} />)

    expect(screen.getByText('token-group')).toBeInTheDocument()
    expect(screen.queryByText('request-group')).not.toBeInTheDocument()
    expect(screen.getByText('Token-based')).toBeInTheDocument()
  })
})
