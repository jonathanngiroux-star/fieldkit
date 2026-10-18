# GOVERNANCE.md — Fieldkit

## License

MIT (whole repository). See LICENSE.

## Contributions

- **DCO only.** Every commit carries `Signed-off-by:` (`git commit -s`).
- **No CLA.** Contributors keep their copyright; the project never asks
  for a transfer or a license grant beyond the MIT license the commit is
  made under.
- Contributions that would add billing, payment processing, telemetry, or
  feature gates will be declined.

## Maintainers

Current maintainer: Jonathann Giroux (@jonathanngiroux-star).
Consensus rule: changes to **license** or **donation wallet addresses**
require 4/4 maintainer consensus *and* 30 days of public notice in this
repository before they take effect. While there are fewer than four
maintainers, the rule is unanimous consent of all listed maintainers plus
the 30-day notice.

## Donation addresses (binding)

- Ethereum / USDC (ERC-20): `0x85ee7E71f762d772599cbF1EC20E651B30657521`
- Bitcoin: `bc1qxe2zx5tv3hdreaej6s2x4p7han85uey828rrhg`

No other wallets may be added to any Fieldkit surface without the
consensus + notice process above. No feature is gated on donations.

## No-billing rule

Fieldkit will never carry a subscription, an invoice, a paid tier, a
payment processor, or "contact sales" language. The repository ships a
walk test that fails the build if billing surface language appears in
source or shipped docs. This rule is changed only by the same 4/4 + 30-day
process as the license.
