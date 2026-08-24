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
import { render, screen } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import { PublicLayout } from '../public-layout'

vi.mock('../public-header', () => ({
  PublicHeader: () => <header data-testid='public-header' />,
}))

describe('PublicLayout', () => {
  test('shows the public header and standard top spacing by default', () => {
    render(
      <PublicLayout>
        <p>Content</p>
      </PublicLayout>
    )

    expect(screen.getByTestId('public-header')).toBeInTheDocument()
    expect(screen.getByRole('main')).toHaveClass('pt-20')
  })

  test('hides the public header and reduces top spacing when disabled', () => {
    render(
      <PublicLayout showHeader={false}>
        <p>Content</p>
      </PublicLayout>
    )

    expect(screen.queryByTestId('public-header')).not.toBeInTheDocument()
    expect(screen.getByRole('main')).toHaveClass('pt-6')
  })
})
