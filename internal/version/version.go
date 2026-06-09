//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

// Package version exposes herdr-plus's release version. Keep this file tiny —
// it is the single source of truth that release CI bumps on every merge to
// main, so the one-line diff stays trivial to review.
package version

// Version is the herdr-plus release version, printed by `herdr-plus version`.
// Release automation bumps the patch number on every merge to main; edit the
// major or minor by hand to cut a larger release.
const Version = "0.0.0"
