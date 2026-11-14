# Shop System

## Purchase Safety Audit (2026-11-14)

Normal NPC/system purchases and carts now calculate charges on a character copy and use the shared item `GrantBagRewards` path. It validates combined bag capacity, compensates partial grants if item creation or wallet persistence fails, and returns every touched stack. Shop purchases are serialized by `purchaseMu`. Single and cart quantities must be positive and bounded; invalid catalog types and undefined system-shop prices are rejected before grant.

Equipment classification follows slot table 19, including armor, outfits and wings; table 29 items remain consumable or material according to kind. Carts respect `Gt=1` regular-gold-only listings per row, and return all updated wallet balances instead of syncing only one field. NPC special-currency purchases return the actual primary and secondary balances. Inventory callbacks now create client instance records before inserting/updating bag slots.

Legacy SellItem and ExchangeItem service methods were mock successes (fixed money 1234567 and a fake item respectively). They now explicitly reject without mutation. Their complete gameplay implementation remains pending. This audit does not make unsupported item effects functional.

Regression checks cover invalid quantities, full bag with a shared cached wallet, payment-save rollback for singles/carts, regular-gold-only restrictions in mixed carts, primary/secondary special currencies, split stacks, and bound/unbound separation. Compensation is best-effort; it is not crash-safe database atomicity across cached runtime state and SQL.

## NPC Direct Shop Callback Routing

Direct NPC template shops (`npc.ShopID > 0`) must always use the standard `onOpenShop` callback path from `internal/presentation/rtmp/handlers/npc/npc_click.go`.

`openShopById` is reserved for explicit special-shop scripts such as the handlers in `npc_script.go`. The Flash client binds those callbacks to different data sources:

- `onOpenShop` -> `TBL_SHOP` / `TBL_SHOP_SLOT`
- `openShopById` -> `TBL_CREDIT` / `ViewManager.NPCSHOP_CONFIG`

Shop IDs overlap between those systems, including low IDs such as `2`, so numeric heuristics like `shopID <= 7` are not valid for direct NPC shop routing. That overlap is what caused normal NPC shops such as `Tiệm Thuốc Ông Vương` to open the courage-shop panel (`NPC_SHOP_PANEL[100]`, "Shop Dũng Khí") instead of the standard shop UI.

## Shop Purchase Bag Slot Sync

Shop purchases can touch more than one bag slot in a single operation. A large buy may both fill an existing stack and create one or more remainder slots.

The Flash client only stays visually correct if the server sends `onAddCharactorSlot` for every changed bag slot, not just the last modified item. `internal/application/shop/purchase_sync.go` now snapshots matching bag items by `templateID`, `itemType`, and `isBound`, then returns `AddedItemDTOs` for every changed slot after `BuyItem`, `BuySystemItem`, and `BuySystemItemMulti`.

This keeps immediate shop updates aligned with later `bagSort` results and preserves the bind-state rule: stack only into slots with the same bound/unbound state, and send separate slot updates when the purchase creates a new remainder stack.

## Reward point wallet repair — 2026-11-14

Confirmed account1 has ExPoint1000 and Honor10; screenshot SystemShopPanel reward-point tab shows1000. Flash SystemShopPanel2819/2820 compares listing.point against player.exPoint. BuySystemItem and cart erroneously charged Character.Honor. Both now charge ExPoint, return exPoint in shopBalances, and currency0 fallback callback maps to exPoint. Honor remains independent. Regression tests cover single/cart, sufficientExPointwithlowHonor, insufficientExPointwithhighHonor and no grant onrejection. Reported listing3982 is item3014 costing68 points. Existing rejection did not deduct ExPoint, so no refund required.

Local tuning 2026-11-14: user requested purple-to-orange pet fusion default60%. Normal base changed30→60 (VIPbonus retained). Senior purpletable now60,60,60,60,60,60,60,60,66,73,81,90,100 for4..16secondarypets. Othercoloursandwingrates unchanged. Flashpanelhardcodedpercent remainsold; serverusesnewrates. Targetedboundarytestsupdated. This is user tuning, not officialGo rate.


Local wishbookshop: userprice3000regularGoldeach; Limitcategory123 slots9003012/9003755 positions7/8. Box3012 localuniformpool40matchingpetbooksTrung/CaoCấp; originalGo poolunavailable. Box3755 chooseonefrom3467,3468,3469,3466,2709 perdescription. SQLsetup tmp/add_wish_book_shop.sql and outputs/VPT-Local/THEM-TUI-SACH-LIMIT.sql. Item/shop/handler tests passed. DBvalidation2listings3000gold,40validrewards,5classicbooks. Dockerimageemptybinarydetectedonrestart; rebuiltwithtest-sguard beforefinalreadiness.


## Correct Limit tab transport 2026-11-14
ScreenshotconfirmedactualLimitrequest isgetLimitTimeShop, previouslystubbednil. Addedrealcatalog responderdictionarykeys1/2 withtid29,itemId,gold3000,q,remain/num-1,Unixsecondstimefields; buyLimitTimeItemrestrictsour2IDsandroutesthroughsafeBuySystemItemClient. Removedstubregistrationtoavoidoverride. Earlieradditiontosid123alonewasinsufficient. Catalogtestsandaffectedhandler/app tests passed. LiveFlashverificationawaitinglogged-inwindow.
