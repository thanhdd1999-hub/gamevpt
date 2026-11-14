# Local server fixes and validation

This repository is an unofficial server implementation. These changes do not establish equivalence with the original GoPlay server.

## Behavior

- Pet progression RPCs now match the Flash argument and response contracts: skill upgrade, star upgrade, aptitude/growth reroll, point reset, normal/advanced fusion, two-phase illusion, evolution and feather progression.
- Fusion, star, reroll, illusion and evolution persistence functions commit the pet and their associated costs together. Failed validation leaves resources unchanged; valid unsuccessful rolls consume the specified resources.
- Pet DTO wrappers retain pi and soulInfo because Flash deliberately omits these fields while flattening data. Missing wrapper fields prevented advanced fusion from sending its RPC.
- Huyen Minh bag 6861 grants one pet and exactly one Van Linh Boi 6859, per the local owner's request. Gift compensation is best-effort across repositories, not process-crash atomic.
- Huyen Minh creature skill data includes skill 7002 alongside 7011 and 7014. The pendant learns one missing 7011/7014 family without overwriting an occupied skill slot.
- Limit shop exposes book gift bags 3012 and 3755 at 3000 gold each through the actual getLimitTimeShop RPC. Book reward data is included in static seeds.
- Inventory callbacks include changed stacks, deleted slots and pet skills so successful actions can refresh immediately.
- Reward-point purchases use the reward-point balance; gold spending follows the VIP experience path.

## Local choices and limitations

The advanced purple-to-orange fusion floor is 60%, requested by the owner. Random reroll distributions, illusion gain distributions, the equal-weight book pool, and some feather checkpoint transitions are local choices where original server behavior is unpublished. Flash client files are read-only and are not patched.

The changed item, shop, pet-handler, cache, persistence and domain-pet suites pass. Targeted pet progression tests pass. The full application/pet suite retains four pre-existing failures concerning expected combat attributes in book/equipment/delete-skill fixtures; these are not hidden by changing their assertions. A full live Flash interaction audit remains necessary.

Runtime account snapshots, credentials, launcher binaries and local audit logs are intentionally excluded from this source commit. Use docker/.env.example and the existing Docker setup to configure a separate environment; do not copy a live user's credentials.
