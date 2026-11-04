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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelCard } from '../model-card'
import { ModelDetailsContent } from '../model-details'
import { PricingTable } from '../pricing-table'

vi.mock('@/lib/lobe-icon', () => ({ getLobeIcon: () => null }))
vi.mock('@visactor/react-vchart', () => ({ VChart: () => null }))
vi.mock('@/features/performance-metrics/api', () => ({
  getPerfMetrics: async () => ({ success: true, data: { groups: [] } }),
}))

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
  test('the all-groups card chooses the cheaper ordinary dynamic tier over an inherited quote', () => {
    const mixed: PricingModel = {
      ...model,
      billing_mode: 'tiered_expr',
      billing_expr: 'tier("base", p * 3 + c * 9)',
      model_ratio: 7,
      group_prices: {},
      group_ratio: { 'token-group': 2, 'request-group': 1 },
      group_pricing: {
        'request-group': {
          billing_mode: 'token',
          input_price: 0.000008,
          output_price: 0.000024,
        },
      },
    }
    render(<ModelCard model={mixed} onClick={vi.fn()} />)
    expect(screen.getByText('token-group')).toBeInTheDocument()
    expect(screen.queryByText('request-group')).not.toBeInTheDocument()
    expect(screen.getByText('$6')).toBeInTheDocument()
  })

  test('group comparison retains ordinary dynamic tiers alongside inherited quotes', () => {
    const mixed: PricingModel = {
      ...model,
      billing_mode: 'tiered_expr',
      billing_expr: 'tier("base", p * 3 + c * 9)',
      model_ratio: 7,
      group_prices: {},
      group_ratio: { 'token-group': 2, 'request-group': 1 },
      group_pricing: {
        'request-group': {
          billing_mode: 'token',
          input_price: 0.000004,
          output_price: 0.000012,
        },
      },
    }
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const details = render(
      <QueryClientProvider client={client}>
        <ModelDetailsContent
          model={mixed}
          selectedGroup='token-group'
          groupRatio={{ 'token-group': 2, 'request-group': 99 }}
          usableGroup={{
            'token-group': { desc: '', ratio: 2 },
            'request-group': { desc: '', ratio: 1 },
          }}
          endpointMap={{}}
          autoGroups={[]}
          priceRate={1}
          usdExchangeRate={1}
          tokenUnit='M'
        />
      </QueryClientProvider>
    )
    const section = screen.getByText('Pricing by Group').closest('section')
    expect(section).not.toBeNull()
    if (!section) throw new Error('missing group comparison')
    const comparison = within(section)
    const tier = comparison.getByText('base').closest('tr')
    expect(tier).toHaveTextContent('$6')
    expect(tier).toHaveTextContent('$18')
    const inherited = comparison.getByText('request-group').closest('tr')
    expect(inherited).toHaveTextContent('$4')
    expect(inherited).toHaveTextContent('$12')
    expect(inherited).toHaveTextContent('1x')
    expect(comparison.queryByText('$28')).not.toBeInTheDocument()
    details.unmount()
    client.clear()
  })

  test('table and details compare inherited prices independently from ordinary prices', () => {
    const inherited: PricingModel = {
      ...model,
      model_ratio: 7,
      completion_ratio: 3,
      group_ratio: { 'token-group': 2, 'request-group': 1 },
      group_prices: {},
      group_pricing: {
        'request-group': {
          billing_mode: 'token',
          input_price: 0.000006,
          output_price: 0.000018,
          image_input_price: 0,
          image_output_price: 0.000009,
        },
      },
    }
    const table = render(
      <PricingTable models={[inherited]} selectedGroup='request-group' />
    )
    expect(screen.getByRole('table')).toHaveTextContent('$6')
    expect(screen.getByRole('table')).toHaveTextContent('$18')
    table.unmount()

    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const details = render(
      <QueryClientProvider client={client}>
        <ModelDetailsContent
          model={inherited}
          selectedGroup='request-group'
          groupRatio={{ 'token-group': 2, 'request-group': 99 }}
          usableGroup={{
            'token-group': { desc: '', ratio: 2 },
            'request-group': { desc: '', ratio: 1 },
          }}
          endpointMap={{}}
          autoGroups={[]}
          priceRate={1}
          usdExchangeRate={1}
          tokenUnit='M'
        />
      </QueryClientProvider>
    )
    const rows = screen.getAllByRole('row')
    const category = rows.find((row) =>
      row.textContent?.includes('request-group')
    )
    const ordinary = rows.find((row) =>
      row.textContent?.includes('token-group')
    )
    expect(category).toHaveTextContent('$6')
    expect(category).toHaveTextContent('$18')
    expect(category).toHaveTextContent('$0')
    expect(category).toHaveTextContent('$9')
    expect(category).toHaveTextContent('1x')
    expect(ordinary).toHaveTextContent('$28')
    expect(ordinary).toHaveTextContent('$84')
    details.unmount()
    client.clear()
  })
  test('an inherited image group overrides the badge without a duplicate model identity', () => {
    const inherited = {
      ...model,
      group_pricing: {
        'request-group': { billing_mode: 'image', per_request_price: 0.4 },
      },
    }
    render(
      <ModelCard
        model={inherited}
        selectedGroup='request-group'
        onClick={vi.fn()}
      />
    )
    expect(screen.getByText('shared-model')).toBeInTheDocument()
    expect(screen.getByText('Image')).toBeInTheDocument()
    expect(screen.getByText('/ Image')).toBeInTheDocument()
    expect(screen.queryByText('Token-based')).not.toBeInTheDocument()
  })
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
