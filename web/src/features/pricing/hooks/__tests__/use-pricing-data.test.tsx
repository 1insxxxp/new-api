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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'

import { useStatus } from '@/hooks/use-status'

import { getPricing } from '../../api'
import { usePricingData } from '../use-pricing-data'

vi.mock('@/hooks/use-status', () => ({
  useStatus: vi.fn(),
}))

vi.mock('../../api', () => ({
  getPricing: vi.fn(),
}))

const pricingData = {
  success: true,
  data: [],
  vendors: [],
  group_ratio: {},
  usable_group: {},
  supported_endpoint: {},
  auto_groups: [],
}

describe('usePricingData', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(useStatus).mockReturnValue({
      status: { price: 1, usd_exchange_rate: 1 },
      loading: false,
      error: null,
    } as unknown as ReturnType<typeof useStatus>)
    vi.mocked(getPricing).mockResolvedValue(pricingData)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  test('refreshes pricing while the model plaza remains open', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )

    const rendered = renderHook(() => usePricingData(), { wrapper })

    await Promise.resolve()
    expect(getPricing).toHaveBeenCalled()
    const initialCalls = vi.mocked(getPricing).mock.calls.length
    await vi.advanceTimersByTimeAsync(30_000)

    expect(vi.mocked(getPricing).mock.calls.length).toBeGreaterThan(
      initialCalls
    )

    rendered.unmount()
    queryClient.clear()
  })
})
