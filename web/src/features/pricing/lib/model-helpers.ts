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
import { EXCLUDED_GROUPS, FILTER_ALL, QUOTA_TYPE_VALUES } from '../constants'
import type { PricingModel, PriceType } from '../types'
import { getDynamicPricingTiers } from './dynamic-price-tiers'

/** Ordinary token input in USD/M, matching the first tier shown in the plaza. */
export function getOrdinaryInputPricePerMillion(model: PricingModel): number {
  if (model.billing_mode === 'tiered_expr') {
    const input = getDynamicPricingTiers(model)[0]?.inputPrice
    return typeof input === 'number' ? input : Number.NaN
  }
  return model.model_ratio * 2
}

/** Pick one real group for a summary; never combine a price from one source
 * with the multiplier or output price of another, or compare unlike units. */
export function getDisplayPricingGroup(
  model: PricingModel,
  selectedGroup?: string
): string | undefined {
  const groups = model.enable_groups ?? []
  if (selectedGroup && selectedGroup !== FILTER_ALL) {
    return groups.includes(selectedGroup) ? selectedGroup : undefined
  }
  if (!model.group_pricing || Object.keys(model.group_pricing).length === 0) {
    return undefined
  }
  const normal = groups.find((group) => !model.group_pricing?.[group])
  const firstQuote = model.group_pricing[groups[0]]
  const isDynamic = model.billing_mode === 'tiered_expr'
  const ordinaryType = isDynamic ? QUOTA_TYPE_VALUES.TOKEN : model.quota_type
  let targetType = ordinaryType
  if (!normal && firstQuote) {
    targetType = firstQuote.billing_mode === 'token' ? 0 : 1
  }
  let best: string | undefined
  let bestPrice = Number.POSITIVE_INFINITY
  for (const group of groups) {
    const quote = model.group_pricing[group]
    let type = ordinaryType
    let price: number
    if (quote) {
      type = quote.billing_mode === 'token' ? 0 : 1
      price =
        type === 0
          ? (quote.input_price ?? Number.NaN) * 1e6
          : (quote.per_request_price ?? Number.NaN)
    } else {
      if (!isDynamic && Object.hasOwn(model.group_prices ?? {}, group)) {
        type = QUOTA_TYPE_VALUES.REQUEST
      }
      const ratio = getConfiguredGroupRatio(model.group_ratio ?? {}, group)
      price =
        type === QUOTA_TYPE_VALUES.TOKEN
          ? getOrdinaryInputPricePerMillion(model) * ratio
          : (model.group_prices?.[group] ?? model.model_price ?? 0) * ratio
    }
    if (type === targetType && Number.isFinite(price) && price < bestPrice) {
      best = group
      bestPrice = price
    }
  }
  return best ?? normal ?? groups[0]
}

export function getInheritedDisplayPrice(
  model: PricingModel,
  selectedGroup?: string
) {
  const group = getDisplayPricingGroup(model, selectedGroup)
  return group ? model.group_pricing?.[group] : undefined
}

export function getDisplayBillingMode(
  model: PricingModel,
  selectedGroup?: string
): string | undefined {
  return (
    getInheritedDisplayPrice(model, selectedGroup)?.billing_mode ??
    model.billing_mode
  )
}

export function hasDisplayPrice(
  model: PricingModel,
  type: PriceType,
  selectedGroup?: string
): boolean {
  const quote = getInheritedDisplayPrice(model, selectedGroup)
  if (quote) {
    switch (type) {
      case 'input':
        return quote.input_price != null
      case 'output':
        return quote.output_price != null
      case 'cache':
        return quote.cache_read_price != null
      case 'create_cache':
        return quote.cache_write_price != null
      case 'image':
        return quote.image_input_price != null
      case 'image_output':
        return quote.image_output_price != null
      default:
        return false
    }
  }
  switch (type) {
    case 'cache':
      return model.cache_ratio != null
    case 'create_cache':
      return model.create_cache_ratio != null
    case 'image':
      return model.image_ratio != null
    case 'image_output':
      return false
    case 'audio_input':
      return model.audio_ratio != null
    case 'audio_output':
      return model.audio_ratio != null && model.audio_completion_ratio != null
    default:
      return true
  }
}

// ----------------------------------------------------------------------------
// Model Helper Utilities
// ----------------------------------------------------------------------------

/**
 * Get available groups for a model
 */
export function getAvailableGroups(
  model: PricingModel,
  usableGroup: Record<string, { desc: string; ratio: number }>
): string[] {
  const modelEnableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []

  return modelEnableGroups
    .filter((g) => !EXCLUDED_GROUPS.includes(g))
    .filter((g) => Object.hasOwn(usableGroup, g))
}

export function orderGroups(
  groups: string[],
  preferred: string[] = []
): string[] {
  const available = new Set(groups)
  return [
    ...new Set([
      ...preferred.filter((group) => available.has(group)),
      ...groups,
    ]),
  ]
}

/**
 * Read a configured group ratio while preserving valid zero ratios.
 */
export function getConfiguredGroupRatio(
  groupRatio: Record<string, number>,
  group: string
): number {
  const ratio = groupRatio[group]
  return typeof ratio === 'number' && Number.isFinite(ratio) ? ratio : 1
}

/**
 * Resolve the group ratio used by model square summary prices.
 *
 * When no specific group is selected, the model square shows the best price
 * available to the viewer. When a group filter is active, it shows that
 * group's price instead.
 */
export function getDisplayGroupRatio(
  model: PricingModel,
  selectedGroup?: string
): number {
  selectedGroup = getDisplayPricingGroup(model, selectedGroup) ?? selectedGroup
  if (getInheritedDisplayPrice(model, selectedGroup)) return 1
  const modelEnableGroups = Array.isArray(model.enable_groups)
    ? model.enable_groups
    : []
  const groupRatio = model.group_ratio || {}

  if (
    selectedGroup &&
    selectedGroup !== FILTER_ALL &&
    modelEnableGroups.includes(selectedGroup)
  ) {
    return getConfiguredGroupRatio(groupRatio, selectedGroup)
  }

  if (modelEnableGroups.length === 0) {
    return 1
  }

  let minRatio = Number.POSITIVE_INFINITY

  for (const group of modelEnableGroups) {
    const ratio = groupRatio[group]
    if (
      typeof ratio === 'number' &&
      Number.isFinite(ratio) &&
      ratio < minRatio
    ) {
      minRatio = ratio
    }
  }

  return minRatio === Number.POSITIVE_INFINITY ? 1 : minRatio
}

/** Resolve a fixed price for the selected group without changing model_name. */
export function getDisplayModelPrice(
  model: PricingModel,
  selectedGroup?: string
): number {
  selectedGroup = getDisplayPricingGroup(model, selectedGroup) ?? selectedGroup
  const inherited = getInheritedDisplayPrice(model, selectedGroup)
  if (inherited) return inherited.per_request_price ?? Number.NaN
  const prices = model.group_prices || {}
  if (selectedGroup && selectedGroup in prices) {
    return prices[selectedGroup]
  }

  const availablePrices = Object.values(prices).filter(
    (price) => typeof price === 'number' && Number.isFinite(price)
  )
  if (availablePrices.length > 0) {
    return Math.min(...availablePrices)
  }
  return model.model_price || 0
}

export function getResolvedQuotaType(
  model: PricingModel,
  selectedGroup?: string
): number {
  selectedGroup = getDisplayPricingGroup(model, selectedGroup) ?? selectedGroup
  const inherited = getInheritedDisplayPrice(model, selectedGroup)
  if (inherited) {
    return inherited.billing_mode === 'token'
      ? QUOTA_TYPE_VALUES.TOKEN
      : QUOTA_TYPE_VALUES.REQUEST
  }
  if (model.billing_mode === 'tiered_expr') {
    return QUOTA_TYPE_VALUES.TOKEN
  }
  if (!selectedGroup || selectedGroup === FILTER_ALL) {
    return model.quota_type
  }
  const prices = model.group_prices || {}
  if (Object.hasOwn(prices, selectedGroup)) {
    return QUOTA_TYPE_VALUES.REQUEST
  }
  if (Object.keys(prices).length > 0 && Number(model.model_ratio) > 0) {
    return QUOTA_TYPE_VALUES.TOKEN
  }
  return model.quota_type
}

/**
 * Replace model placeholder in endpoint path
 */
export function replaceModelInPath(path: string, modelName: string): string {
  return path.replaceAll('{model}', modelName)
}

/**
 * Check if model is token-based pricing
 */
export function isTokenBasedModel(
  model: PricingModel,
  selectedGroup?: string
): boolean {
  return getResolvedQuotaType(model, selectedGroup) === QUOTA_TYPE_VALUES.TOKEN
}
