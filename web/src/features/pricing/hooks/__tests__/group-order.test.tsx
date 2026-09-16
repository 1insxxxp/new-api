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
import { renderHook, waitFor } from '@testing-library/react'
import type { PropsWithChildren } from 'react'
import { expect, test, vi } from 'vitest'

import { getPricing } from '../../api'
import { getAvailableGroups } from '../../lib/model-helpers'
import { usePricingData } from '../use-pricing-data'

vi.mock('../../api', () => ({ getPricing: vi.fn() }))
vi.mock('@/hooks/use-status', () => ({ useStatus: () => ({ status: {} }) }))

test('sync order controls filters and model group lists while preserving local groups', async () => {
  const data = {
    success: true,
    data: [
      {
        id: 1,
        model_name: 'shared',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: ['alpha', 'zeta', 'beta'],
      },
    ],
    vendors: [],
    group_ratio: {},
    supported_endpoint: {},
    auto_groups: [],
    usable_group: {
      alpha: { desc: '', ratio: 1 },
      beta: { desc: '', ratio: 1 },
      zeta: { desc: '', ratio: 1 },
      local: { desc: '', ratio: 1 },
    },
    group_order: ['zeta', 'beta', 'missing', 'zeta', 'alpha'],
  }
  vi.mocked(getPricing).mockResolvedValue(data)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const wrapper = (props: PropsWithChildren) => (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
  const { result, unmount } = renderHook(() => usePricingData(), { wrapper })
  await waitFor(() => expect(result.current.models).toHaveLength(1))
  expect(Object.keys(result.current.usableGroup)).toEqual([
    'zeta',
    'beta',
    'alpha',
    'local',
  ])
  expect(result.current.models[0].enable_groups).toEqual([
    'zeta',
    'beta',
    'alpha',
  ])
  expect(
    getAvailableGroups(result.current.models[0], result.current.usableGroup)
  ).toEqual(['zeta', 'beta', 'alpha'])

  vi.mocked(getPricing).mockResolvedValue({
    ...data,
    group_order: ['alpha', 'zeta', 'beta'],
  })
  await result.current.refetch()
  await waitFor(() =>
    expect(Object.keys(result.current.usableGroup)).toEqual([
      'alpha',
      'zeta',
      'beta',
      'local',
    ])
  )
  unmount()
  client.clear()
})
