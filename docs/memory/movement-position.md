# Movement & Position

Position update RPC: `udcp(posX, posY, centerX, centerY, dir, moveInfo)`. Speed validation: `distance * 1000 / elapsedTime` compared against `serverSpeedThreshold` (~300 units/sec).

## Unlinked Portal Recovery (2026-11-14)

Live `sceneChange([757])` on map 3 failed because scene-item instance 757 has `from=0`; the request itself was valid. Explicit `from` links remain authoritative. For missing links only, portal template 38 can resolve an exact unique destination map name to that map's configured SafeX/SafeY. This fallback rejects duplicate names, self-targets, copy/instance maps, nonpositive map type, and missing/out-of-bounds safe points. Gate 757 therefore routes to map 1 Xuất Vân Thôn at (1000,1000). This restores a route using the existing safe-spawn convention; it does not claim to reconstruct the original portal landing coordinate. Unresolved gates now show a game notice instead of a misleading invalid-arguments connection error.
