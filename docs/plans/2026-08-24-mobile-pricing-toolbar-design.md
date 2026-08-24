# Mobile Pricing Toolbar Design

## Problem

The price display and token unit segmented controls are wrapped in
`hidden sm:flex`, so both controls have a zero-size layout below the `sm`
breakpoint. Mobile users cannot switch between standard and recharge prices or
between `/1M` and `/1K` units.

## Design

Keep the existing desktop toolbar unchanged. Below the `sm` breakpoint, render
the two pricing controls in their own full-width row between the model summary
and the sort/view controls. The row uses the existing `SegmentedControl`
component and wraps if translated labels ever exceed the available width.

Do not add horizontal scrolling or replace the controls with a dropdown. The
controls remain directly visible and retain their current state and callbacks.

## Verification

- A component test verifies both pricing control groups are present in the
  mobile-only row and that the row is hidden at `sm` and above.
- Existing pricing tests remain green.
- Production build and type checking pass.
- At a 390px viewport, both controls have non-zero bounds, fit within the
  toolbar, and switch their existing values.
- At a desktop viewport, only the existing desktop control row is visible.
