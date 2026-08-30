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

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

const model: PricingModel = {
  id: 1,
  model_name: '按次反重力1/gemini-3.1-pro-preview-thinking',
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  model_price: 0.02,
  enable_groups: ['【酒馆】次Gemini-反重力'],
  supported_endpoint_types: ['gemini', 'openai'],
}

describe('model card mobile header', () => {
  test('wraps the complete model name on mobile and truncates it on desktop', () => {
    render(<ModelCard model={model} onClick={vi.fn()} />)

    expect(screen.getByRole('heading', { name: model.model_name })).toHaveClass(
      'break-all',
      'whitespace-normal',
      'sm:truncate'
    )
  })

  test('moves actions below the model information on mobile', () => {
    render(<ModelCard model={model} onClick={vi.fn()} />)

    expect(
      screen.getByRole('button', { name: 'Details' }).parentElement
    ).toHaveClass(
      'col-start-2',
      'row-start-2',
      'sm:col-start-3',
      'sm:row-start-1'
    )
  })
})
