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
import type { PricingModel } from '../types'
import {
  parseTiersFromExpr,
  splitBillingExprAndRequestRules,
  type ParsedTier,
} from './billing-expr'

export function isDynamicPricingModel(model: PricingModel): boolean {
  return model.billing_mode === 'tiered_expr' && Boolean(model.billing_expr)
}

export function getDynamicPricingTiers(model: PricingModel): ParsedTier[] {
  if (!isDynamicPricingModel(model)) return []
  const { billingExpr } = splitBillingExprAndRequestRules(
    model.billing_expr || ''
  )
  return parseTiersFromExpr(billingExpr)
}
