# Huyền Minh Linh Vũ skill and gift investigation

## Confirmed local defects

Creature2281 has handbook90 relating to it and listing three skill IDs:7002(U Minh Huyền Vũ),7011(Tứ Phương Khiếp Đảm),7014(Thợ Săn Sát Thủ). Its runtime creature-skill table had only7011 and7014. `ContractPets` seeds defaults exclusively from that table, so the innate7002 was never created.

Applied idempotent data repair from handbook.skill_id1, scoped to2281 and7002, inserting creature-skill row5168. Exported the actual updated table through SupabaseCLI to canonical seed. Regression test `TestHuyenMinhContractIncludesCanonicalInnateSkill` reads this real seed, loads it into game data, contracts the pet and requires all three IDs. No general handbook-to-default fallback was added for unrelated pets. Existing pet records are not modified by this data repair; they require a separate conservative backfill preserving occupied slots.

## Gift6861 and Vạn Linh Bội6859

6861 description promises one blue growth1.5 Huyền Minh Linh Vũ, and a chance of Vạn Linh Bội. The current handler contracts only a pet; no additional-item roll exists. Live data_tbl_item_award has no rows for6861 or6859. Therefore no amount of opening6861 can currently produce6859 through that path. This is missing logic, not ordinary bad luck.

6859 description says it allows the pet to learn Thợ Săn Sát Thủ or Tứ Phương Khiếp Đảm. The exact original bonus-drop percentage is absent from the supplied data and was not found in official searches. It must not be silently replaced with an invented percentage described as original.

## Official source distinctions

- https://vuaphapthuat.go.vn/bai-viet/26-10-update-phien-ban-moi-dau-truong-tranh-ba/10238.html announces Huyền Minh Linh Vũ in the26October2023 update. This pet is not evidence of a Go2020 build.
- https://vuaphapthuat.go.vn/danh-muc-su-kien/sao-lap-lanh/3185.html lists an older bundle of9HMLV boxes plus1guaranteedVạnLinhBội,1pill and7shards.
- https://vuaphapthuat.go.vn/bai-viet/sao-lap-lanh/10822.html (12February2026) lists10boxes,2pills,3soul boxes and6shards, with no guaranteedVạnLinhBội.
- Local send_combine_data.go uses10boxes,1pill,7shards. It is a custom mixture; an old official bundle cannot establish a missing guaranteed reward for every current purchase.

No drop-rate or reward-pool change was made during this investigation. The original probability remains unresolved.

## Owner-selected local rule applied

The owner subsequently chose **one guaranteed Vạn Linh Bội per6861 box**, rather than a probability. This is an explicit local rule, not a recovered official drop percentage.

Added `hmlv_gift.go` and a scoped6861 handler integration. Every successful opening now grants one growth1.5 pet and one6859 pendant with binding matching the source box. The helper grants pendant inventory first, then creates the pet and consumes the source inside the grant batch commit callback. Returned errors compensate the pet and pendant; fully consumed source stacks reserve a temporary slot so a full bag can reuse the old box slot. Compensation failures are joined into the returned error and logged. This is best-effort compensation, not process-crash atomicity across pet and item repositories.

Tests cover three consecutive openings, both bindings, bonus grant failure before pet creation, pet creation failure, and source consumption failure rolling back the created pet/pendant and restoring the source slot. Full item service and item handler suites and item vet pass.

Vạn Linh Bội6859's learning behavior remains unsupported in the current generic550 effect path. The existing use-item handler rejects unsupported effects without consuming the item; no learning effect was fabricated in this change.

## Vạn Linh Bội use implemented

The previously unsupported6859 learning path is now implemented. Flash `Player.useItem` confirms CTRL+double-click targets the active battle pet with `useItem(2, battlePetID, slotID)`; no invented book-choice UI or item IDs were added.

The new scoped pet service validates the owner,6859 source and HMLV template2281/2282, then learns one missing7011/7014 skill family in an empty unlocked slot. Higher-level members of the same family count as already known. Existing skills are never overwritten. If both are known, target is another species, or no open slot remains, it retains the pendant and gives a specific explanation. It saves a cloned pet before consuming exactly one pendant and compensates the pet on a returned consumption error. Choosing uniformly among missing families is a local implementation assumption; an original server distribution was not recovered.

The generic item-effect callbacks now send present skill1..15 fields and a derived-property refresh for updated pets, rather than life alone. Regression test verifies immediate skill3 refresh without adding the pet a second time. Targeted pet/HMLV contract tests, full item service and item handler suites, and vet pass.
