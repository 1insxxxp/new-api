# Pricing Mode Currency Symbol Design

## Goal

On the public model pricing page, keep all existing price calculations unchanged while making the selected display mode control the visible currency symbol:

- Standard mode displays `$`.
- Recharge mode displays `¥`.

## Scope

The change applies only to model pricing displays on `/pricing`, including model cards, the table view, dynamic pricing entries, and the model details drawer. It does not change billing calculations, group ratios, recharge rates, balances, payment pages, or the global currency configuration.

## Design

Extend the shared currency formatter with an optional display-symbol override that affects only presentation. The formatter must still execute the existing conversion and rounding path before replacing the symbol.

Pricing formatters will pass `$` when `showRechargePrice` is false and `¥` when it is true. Centralizing this decision in the pricing formatting helpers keeps cards, tables, and details consistent without coupling the toolbar to every price component.

## Verification

Add focused tests covering standard and recharge modes for token-based, fixed-request, and dynamic prices. Verify that the numeric portion remains identical to the current behavior and that only the symbol changes. Run the relevant frontend test suite and a production build before deployment.
