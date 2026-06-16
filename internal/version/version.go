//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

// Package version exposes herdr-plus's release version. Keep this file tiny —
// it is the single source of truth a future release pipeline bumps, so the
// one-line diff stays trivial to review.
package version

// Version is the herdr-plus release version, printed by `herdr-plus version`.
// This is the fresh plugin-first rebuild, starting at 0.1.0. Edit the major or
// minor by hand to cut a larger release.
const Version = "0.1.10"
