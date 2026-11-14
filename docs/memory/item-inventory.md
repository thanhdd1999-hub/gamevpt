# Item & Inventory System

## Local Expansion Book Fix (2026-11-14)

### Gift-Type Context Guard

The choose-one pet handler previously accepted any selectable gift description and searched creature names without requiring a pet context. Wing gift 3376 mentions Ác Ma and accidentally matched creature 504, granting a pet. CanHandle now requires the same explicit pet context as other pet bags. Unimplemented non-pet choice gifts are rejected without consuming the package. This prevents an incorrect pet grant; it does not implement the wing reward selection UI.

The follow-up purchase audit also closed no-op consumption in the default and generic buff fallbacks. Skill/formula books require a successful presentation-layer preparation step (`UseContext.BookPrepared`) before consumption; actual life-skill and recipe learning still runs through the existing prepare/finalize/rollback handlers. Unknown templates and unsupported effects fail without changing the stack. These guards preserve inventory rather than pretending to implement missing gameplay.

Type-500 scrolls previously returned a success message and consumed items without applying any effect. Explicit handling now expands regular bag pages with template 325 (max 7), bank pages with 324 (max 5), and carried-pet slots with 326 (runtime `pet.MaxPets`, currently 180). Page setters synchronize the derived 30-slot capacities; pet capacity keeps PetSlots synchronized. The existing character update path persists cached or offline state. Effects carry a minimal `character_updates` map, emitted as `onUPP`; unchanged capacity keys are omitted to avoid false notices. At capacity the book is retained and an explicit message is shown. Other type-500 scrolls are rejected without consumption until their effect is implemented, including pet-bank expansion book 2092.

Bulk use now rejects nonpositive counts and permits only supported HP/MP potion and food types (501/502), because the bulk response only carries recovery effects. The RTMP handler validates argument types and bounds instead of using panic-prone type assertions. Regression tests cover consumption, unchanged-capacity limits, unsupported scroll preservation, bulk expansion rejection, and malformed bulk arguments.

The surrounding audit also found broader preexisting incompleteness: generic skill-book handling (type 505) only generates a message outside the separate life-skill pre-handler, and the default effect fallback can consume an unsupported item without an effect. These are separate implementation gaps, not covered by the expansion patch. Item removal and character persistence are separate writes; database-failure atomicity is not guaranteed by this patch.

## Item-Use Target Gate

The Flash item-use contract uses `useType` only as a target gate (`1` character, `2` pet, `3` either, `4` not directly usable) before branching by item `type` and special item IDs. The choose-one package panel contract is `useItem` -> server callback `itemMultitoSel({packItemList:{idx:{ti,ii,n,q}}, packGetNum}, sid)` -> client RPC `selMultiItemByIdx(packItemSid, selectedIdx)`; the responder returns `0` to close the panel or a positive remaining count to keep it open. Current server support matches this choose-one flow, but generic item use still misses full server-side `useType` target enforcement, general pet-target consumable routing, paid package `usePackageByMoney` / `onUsePackageByMoney`, and `onItemUseCallBack` confirmation wiring. See `docs/research/2026-04-25_02_ITEM_USE_PANEL_CONTRACT_RESEARCH.md`.

## Core Data Tables

TBL_CHARACTOR=2, TBL_CHARACTOR_SLOT=9, TBL_CREATURE=12, TBL_EQUIPT_INSTANCE=18, TBL_EQUIPT_TEMPLATE=19, TBL_ITEM_INSTANCE=28, TBL_ITEM_TEMPLATE=29, TBL_MAP=33, TBL_NPC=35, TBL_PET=39, TBL_QUEST=45, TBL_SKILL=52, TBL_BUFF=66.

Slot object: `{id, sid, type, itemId, stackNum, tid}`.

Client item category constants used by the Flash client:

- Kinds: `1=ITEM_KIND_MAINHAND`, `2=ITEM_KIND_SUBHAND`, `3=ITEM_KIND_DEFENCE`, `4=ITEM_KIND_JEWELRY`, `5=ITEM_KIND_ITEM`, `6=ITEM_KIND_MATERIAL`, `7=ITEM_KIND_PET`, `8=ITEM_KIND_MAGICWEAPON`, `9=ITEM_KIND_PETEQU`, `10=ITEM_KIND_FLYER`, `11=ITEM_KIND_DRESS`, `12=ITEM_KIND_GATHER`, `13=ITEM_KIND_WING`, `14=ITEM_KIND_FEATHER`.
- Kind `1` / main hand: `100=HAMMER`, `101=STICK`, `102=GUN`, `103=SWORDONE`, `104=GUITAR`, `105=BAT`.
- Kind `2` / sub hand: `200=SHIELD`, `201=BOOK`, `202=CUFF`, `203=KNIFE`, `204=PICK`, `205=GLOVE`.
- Kind `3` / defence: `300=HAT`, `301=CLOTHES`, `302=TROUSERS`, `303=BELT`, `304=SHOE`, `305=SHOULDER`.
- Kind `4` / jewelry: `400=NECKLACE`, `401=RING`, `402=JEWELRY1`, `403=JEWELRY2`.
- Kind `5` / item: `500=REEL`, `501=MEDICINE`, `502=FOOD`, `503=JEWEL`, `504=STAR`, `505=SKILLBOOK`, `506=CREBOOK`, `507=PETFUNC`, `508=QUEST`, `509=KEY`, `510=MW_REPAIR`, `511=MW_SKILL_RESET`, `512=PETEQU_LEVELUP`, `513=PETEQU_MODCOLOR`, `514=MW_TRANS`, `515=FISHING_TOOL`, `516=FORMULA`, `517=TEMP_BAG`, `518=WING_ENHANCE`, `521=FAIRY_SKILL_ITEM`. Extra client-only item-related constants: `519=MW_PROP_RESET`, `520=TARGET_ITEM`, `522=MW_STAGE_EIGHT`, `523=SUBLIME`, `524=RESTRAIN`, `527=QILINGSTORE`, `551=STAR_ADD`, `552=STAR_SPEED`.
- Kind `6` / material: `600=DIAMOND`, `601=METAL`, `602=WOOD`, `603=JADE`, `604=CLOTH`, `605=FUR`, `610=FISH`, `611=PLANT`, `612=HERB`.
- Kind `7` / pet: `700=PET_HUMAN`, `701=PET_MONSTER`, `702=PET_PLANT`, `703=PET_MACHINE`, `704=PET_DEVIL`, `705=PET_DRAGON`.
- Kind `8` / magic weapon: `800=MAIN_MAGICWEAPON`, `801=SUB_MAGICWEAPON`.
- Kind `9` / pet equip: `900=PETEQU_SPUR`, `901=PETEQU_NECK`, `902=PETEQU_BELL`, `903=PETEQU_WING`, `904=PETEQU_ARMOR`, `905=PETEQU_CUFF`, `906=PETEQU_DRAGON1`, `907=PETEQU_DRAGON2`.
- Kind `10` / flyer: `1000=FLYER_SUBTYPE`.
- Kind `11` / dress: `1100=DRESS_SUBTYPE`.
- Kind `12` / gather: `1200=GATHER_GLOVE`, `1201=GATHER_ROD`.
- Kind `13` / wing: `1300=WING`.
- Kind `14` / feather: `1400=FEATHER_MIX`, `1401=FEATHER_A`, `1402=FEATHER_B`, `1403=FEATHER_C`, `1404=FEATHER_D`.

## Item Instance Field Mapping

Key mappings (Go -> Flash client):

- `TemplateID` -> `tid`, `cid`, `giid`, `tplId`
- `ID` -> `id`, `itemId`
- `StackCount` -> `stackNum`, `stackCount`
- `IsBound` -> `binded` (NOT "isBound")
- `EnchantLevel` -> `enchantLv`
- `StarLevel` -> `starLv`, `upgradeNum` (client tooltip reads this for star display)
- `ColorCode` -> `color`, `colorCode` (instance overrides template)
- `Durability` -> `endure` (NOT "durability")
- `MaxDurability` -> `maxEndure`
- `SlotType` -> `slotType`, `SlotIndex` -> `slotIndex`
- `CalculateSID()` -> `sid` (computed global slot ID)
- `ItemType` -> `itemType`, `type` (table type 28 or 18)

Item Template fields all use camelCase JSON tags: `itemLevel`, `color`, `colorCode`, `iconCode`, `reqLevel`, `reqClass`, `bindType`, `propType`, `proplNum`, `useType`, etc.

Combined DTO structure: `{"type": 28, "inst": {...}, "temp": {...}}`.

Field priority: Instance `color` overrides template `color`. Instance `binded` for binding checks. Template `itemLevel` for display.

Missing fields fix: `f` (field data JSON string, `"{}"` when empty), `t` (type value from template, defaults to `-1`).

## SID Boundaries

`CalculateSID`/`ParseSID`: Bag=2101..2310, Quest bag=2311..2340, Pet item bag=2341..2370.

## Item RPC Argument Hardening

Flexible integer parser for mixed types (`float`, `int`, `int64`, unsigned ints, numeric strings). Applied to `UseItem`, `DropItem`, `EquipOn`.

## Inventory Bag Fix

Warm inventory logic aligned with slot enums. Slot capacities from domain constants (`DefaultBagSlots`, `DefaultQuestBagSlots`, `DefaultPetItemBagSlots`, `EquipSlotMax`).

## Equipment Property Name Mapping

Equipment properties (numeric ID -> Vietnamese name): 1=Gioi han HP, 2=Gioi han MP, 3=Nang luong, 4=Cong vat ly, 5=Cong ma phap, 6=Phong vat ly, 7=Phong ma phap, 8=Chinh xac, 9=Ne tranh, 10=Phan kich, 11=Toc do, 12=Lien kich, 13=Bao kich, 14=Xuyen phong ngu, 20=Luc (STR), 21=Nhan (AGI), 22=Tri (INT), 23=Than (SPI), 24=Gioi han HP (%), 25=Gioi han MP (%), 27=Khang bao kich, 28=Khang debuff, 29=Khang xuyen.

Active soul properties: 1=Cong vat ly, 2=Cong ma phap, 3=Bao kich, 4=Xuyen phong ngu, 5=Giam sat thuong vat ly, 6=Giam sat thuong ma phap, 7=Giam sat thuong, 8=Tang HP.

## Equipment Flag Fields

`flag`: For standard/pet equipment: `sublimeId`, `sublimeElement`, `sublimeAdd`. For spirit weapons (slots 15-20): `propType` and `propVal`. Example: `{sublimeElement:1,sublimeAdd:0,sublimeId:50}`.

`flag2`: Ky Linh (spirit companion) data. 3 lines (indexes 0-2), each with `t` (property code), `v` (current value), `max` (maximum cap). Example: `{0:{t:60,v:0.025,max:0.025},1:{t:5,v:2500,max:2500},2:{t:59,v:0.011,max:0.025}}`.

`flag3`: Pet equipment stone slots. Keys `1..6` represent pet stone slots. Example: `{1:[1,1,12345],2:[1],3:[1]}`.

## Material Mixing

RPCs: `materialMixOne(num, itemID, tempBagFlag)` and `materialMixAll(num, itemID, tempBagFlag)`. `num` = quantity consumed per attempt (2-5).

Quality system (ColorCode): 0=White, 1=Green, 2=Blue, 3=Purple, 4=Orange, 5=Red (max). Client display: `color = Math.ceil(quality / 5)`.

Success rate: Normal materials (colorCode != 4): `num * 20%`. Rare materials (colorCode == 4, orange): `num * 6%`.

Response format: `{"slotId": <id>, "num": <remaining stack>, "flag": <success bool>, "finalNum": <quantity produced>}`.

## Equipment Quality & Prefix System

Equipment items have a `ColorCode` (0-5) determining display color and Vietnamese quality prefix (`preNameType`).

ColorCode to Prefix & Color: 0=(none), 1=Binh Thuong/White, 2=Cuong Hoa/Green, 3=Tinh Xao/Blue, 4=Hoan My/Purple, 5=Trac Viet/Orange-Yellow.

Quality -> ColorCode: `(quality + 4) / 5` clamped to [0, 5]. Quality 1-5->1, 6-10->2, 11-15->3, 16-20->4, 21-25->5.
Quality -> PrefixType: `quality % 5` (where 0 becomes 5), cycles 1-5 within each tier.
ColorCode -> Representative Quality: `((colorCode - 1) * 6) + 1`.

Equipment creation now derives `preNameType` from `ColorCode` instead of hardcoding `0`. Falls back to template color data when caller doesn't provide color code.

## Equipment Active List Sync

`equipActiveList` built from currently worn equipment. Key = `sid`, value = `true` when template's `activeEquipId > 0` AND required template currently equipped; otherwise `false`.

Emitted at: `onChooseCharactor` (login), `onEquipActiveList` in `equipOn`, `equipOff`, and auto-equip flow of `useItem`.

## Check Equip Edit Compatibility

Registered no-op handler for `checkEquipEdit` RPC. Client probes this when opening equipment/wing panels. Returns no pending state. If server later persists interrupted equipment-edit workflows, should load saved object and trigger `onGetEquipEditObj`.

## Equipment Display Contract

Implemented a shared application-layer equipment display contract in `internal/application/item/display_contract.go`.

Primary entrypoints:

- `BuildClientItemDTO(it *domainitem.Item)`
- `BuildClientItemDTOList(items []*domainitem.Item)`
- `ApplyEquipmentDisplayContract(recordData, templateData)`

Purpose: keep all runtime equipment payloads aligned with the Flash client tooltip contract used by `TipEquip`, `EquipFunc`, inventory callbacks, `equiptList`, and gamedata hydration.

Centralized behavior now includes:

- color-tier fallback resolution in this order: instance `colorCode` -> instance `q` / `quality` -> instance display `color` -> template `colorCode` -> template `color`
- display alias normalization for `color`, `colorCode`, and `preNameType`
- bind normalization to client string flags `binded = "0" | "1"`
- template fallback injection for `mainProp1/2`, `mainPropNum1/2`, `prop1/2`, `propNum1/2`, `activeProp`, `activePropNum`, `bindMainPropNum1/2`, `holeNum`, `endureMax`
- default compatibility fields for `element`, `maker`, `flag`, `flag2`, `flag3`, `t`, and `t1..t10`
- durability alias population for `endureLeft` and `endureMax`

Current runtime coverage:

- item RTMP handlers now route equipment callbacks through the shared builder instead of using raw `ToDTO()` for bag, equip, temp bag, craft, prefix-change, jewel, maker, mix, bind, and use-item flows
- `sendInventoryUpdate` and `getInitSlot` use the shared builder
- appearance sync now uses the shared builder for `equiptList`
- gamedata normalization delegates to `ApplyEquipmentDisplayContract` instead of maintaining separate handler-local color/prefix logic

This was intentionally implemented as application-layer composition, not repository formatting, so domain rules stay in `internal/domain/item/display.go` while presentation only sends normalized payloads.

Focused tests cover:

- equipment DTO normalization from template fallbacks
- quality-driven fallback for `colorCode`, display `color`, and `preNameType`
- default compatibility values for `flag` and `t1..t10`
- durability alias output and bind normalization

Quest reward equipment now randomizes core prop-line assignment from the template stat pool (`mainProp1/2`, `prop1/2`) before rolling the numeric values from the client `EQUIPT_QUALITY[q-1..q]` range. This keeps reward equipment template-derived while no longer forcing a fixed prop layout.

Crafted equipment produced by `newMake` now uses the same template-pool prop shuffle and multiplier-based value rolls when a preview-selected quality is applied. The plain `UpdateEquipmentQuality` path remains non-rerolling so systems like change-prefix do not overwrite existing rolled stats.

## Item Handler RPCs (40+ methods)

Equipment: `equipOn`, `equipOff`, `getInitSlot`, `isEquSid`. Inventory: `moveItem`, `moveItemNum`, `dropItem`, `bagSort`, `moveItemToPage`, `itemToTarget`. Item Use: `useItem`, `useItemGold`, `useMultiItem`, `useTransport`, `useTransportGroup`. Recovery: `fullHpRecoverByItem`, `fullMpRecoverByItem`. Repairs: `repair`, `repairAll`. Appearance: `setDressHide`, `checkEquipEdit`, `checkLimitWingEffect`, `checkSameDay`. Temp Bags: `getTBag`, `openTempSlot`, `openmxTempSlot`, `mxBagSort`, `tempBagSort`, `oneKeyOpenAllMXTemp`. Equipment Modifications: `changePrefix`, `sureChangePrefix`, `changeSoul`, `sureChangeSoul`, `changeElement`, `sureChangeElement`, `changeBind`, `sureChangeBind`, `changeLevel`. Equipment Operations: `equResolve`, `starOne`, `starAll`, `getStarNum`. Sockets: `getHoleNum`, `holeDig`, `getJewelData`, `jewelSet`, `jewelDel`, `jewelUpdateOne`, `jewelUpdateAll`. Crafting: `newMake`, `getMakeColor`, `materialMixOne`, `materialMixAll`. Item Management: `bindItem`, `changeName`. Social: `useSeek`, `useTrack`, `useFootle`, `useAntiFootle`, `useSnowBall`, `useHalloweenCard`, `useFestFootle`, `useRadar`.

### Transport Scrolls (Track A)

`useTransport(mapID)` consumes a regular transport scroll (priority: 1036, 1363, 2946, 1515, then senior IDs as fallback) and teleports the caller. `useTransportGroup(mapID)` validates group leadership via `groupService.IsGroupLeader`; non-leaders are charged the regular cost. `useItemGold({tid, mid})` is the client's "I have no scroll, pay gold" path — server still tries the matching item priority list defensively before charging gold. All three share `chargeAndTransport` -> `performTransport` in `internal/presentation/rtmp/handlers/item/transport.go`. If the player has no item, gold is deducted from `Money` (when `SelectedMoneyType == 2`) or `MoneyBind` (otherwise) — costs: regular `transportRegularGoldCost = 1000`, senior/group `transportSeniorGoldCost = 5000`. When neither item nor gold is available, the client receives `onSystemMidMsgOrNote` and the teleport is aborted. Item path emits `onDelCharactorSlot`/`onUpChaSlotStackNum`; gold path emits `onUPP` with the updated money field. Map validation rejects `mapID` with `MapTemplate.T <= 0`.

### Social Prank Pattern (Track C)

`useFootle`/`useAntiFootle`/`useSnowBall`/`useHalloweenCard`/`useFestFootle` all share `applySocialPrank(ctx, args, rpcName, templateIDs, buildMessage)` in `friend_items.go`. The helper resolves the target by name, validates same-channel + same-map, consumes one item from `templateIDs`, sends `onDelCharactorSlot`/`onUpChaSlotStackNum`, and broadcasts `onExcludeRedMsg` to the scene. `buildMessage(src, target *Character) string` builds the per-item message body.

### Radar / NPC Locator (Track B)

`useRadar(npcName)` looks up NPCs via `gameData.GetNPCsByName` (filters `Fd > 0`), prefers same-map candidates over the first match, consumes one of `[1397, 1484, 1615]`, and replies with `onNpcPos([{name, posMapId, posX, posY}])`. Returns `onNpcPos(nil)` without consuming when no NPC matches.

### Item Type Stub Coverage

`internal/application/item/service.go` `phase4Stubs` covers types 510, 511, 514, 515, 517, 518, 519, 521, 522, 523, 524, 525, 527, 551, 552 — these emit `"loại vật phẩm <name> chưa được hỗ trợ"` and do **not** consume the item. Types 514, 522, 525 were added in this pass to prevent silent item destruction when the type has no real handler.

## Potion Healing System

Healing potion templates have `I1=I2=0` in the database. Heal amounts are hardcoded server-side in `internal/application/item/potion.go` to match the Flash client (`PortraitCanvas.as`). HP potions: template IDs 1, 2, 4, 110, 113, 321, 371, 1975 (300-9000 HP). MP potions: template IDs 158, 125, 126, 127, 322, 370, 1974 (300-7000 MP). The `applyItemEffect` function uses `GetPotionHealAmount()` to resolve heal amounts when I1/I2 are zero. The `fullHpRecoverByItem`/`fullMpRecoverByItem` RPCs consume multiple items from lowest to highest heal amount until HP/MP is full. The client sends item arrays with `{sid: itemDatabaseID, priority: healAmount - boundStatus}`.

## UseItem Effect Registry (Hybrid Architecture)

Item use effects are dispatched through a two-tier system in `internal/application/item/`. Each handler implements `CanHandle(tpl)` and `Apply(ctx, uctx, char, item, tpl)` returning an `ItemEffectResult`. The `UseContext` carries RPC parameters (targetType, targetID, petID, quantity).

Dispatch tiers:

1. **Type registry** (`typeHandlers[tpl.Type]`) — deterministic lookup by item template Type field
2. **Award table** (`awardEffectHandler`) — items with `data_tbl_item_award` entries
3. **Fallback chain** — `boxEffectHandler` (UseType 1), `walletEffectHandler`, `buffEffectHandler`, `defaultEffectHandler`

### ItemTemplate `use_type` semantics

The Flash client uses `useType` first as a target-eligibility gate, not as the full item-effect dispatcher:

- character target accepts only `useType = 1` or `3`
- pet target accepts only `useType = 2` or `3`
- actual behavior after that still branches mostly by template `type` and a few special item IDs / RPC paths

High-confidence meanings:

- `1` = character-target usable
- `2` = pet-target usable
- `3` = usable on either character or pet
- `4` = not directly usable / material-style item

Observed examples only, not the authoritative meaning:

- `useType = 1` commonly appears on skill books, bags, gift boxes, and other direct character-use items
- `useType = 2` commonly appears on pet buffs and pet skill-book style items
- `useType = 3` commonly appears on consumables that can target either the player or a pet, including many healing/battle items

Server-side rule of thumb: treat `useType` as a coarse access gate, then let the shared UseItem registry decide the real effect from `type`, award rows, and any special-case handlers. Do not treat older documentation examples of `use_type = 10` as confirmed live client behavior.

Type registry (27 types mapped):

- 500=scroll, 501=potion, 502=food, 503=gem socket, 504=star stone, 505=skill book, 506=creature book
- 507=pet func, 508/509/550=quest item (delegates to award→giftBox→petBag), 512/513=pet equip
- 516=formula, 520=target item
- Phase 4 stubs (510,511,515,517,518,519,521,523,524,527,551,552): return "chưa được hỗ trợ"

Award table types: 12=pet, 19=equipment, 29=item, 30=currency (silver/gold/honor), 31=exp, 32=buff.

Gift box flow: `useItem` -> `itemMultitoSel` callback -> client opens MultiItemPanel -> user selects -> `selMultiItemByIdx` RPC -> server grants reward -> returns remaining count (0 = done).

Special item RPCs: `changeNameByCard`, `changeCharSex`, `useGoodCard`, `useWeddingBag`, `useRedBag`, `useIntimacyItem`.

Temp bag rewards: effect handlers can set `TempBagItems` in result, dispatched as `warnTemporaryBag` callback.

Research: `docs/research/2026-04-08_01_USEITEM_HYBRID_ARCHITECTURE_RESEARCH.md`

## Local consumable repair 2026-11-14

Confirmed useItem(1,-1,67) at08:20:54 consumed template2846 with no effect. This is an MP reserve refill, not a direct MP potion; template I1/I2 are zero. Added explicit reserve mappings:2846 MP50000,2847 HP50000,2037 MP1000000,2038 HP1000000. Stored pipe-string in Character.PMProcessData.localBloodBag through existing persistence; login ep now reflects stored reserve rather than constant zeros. Capacity10million per reserve, reject overflow without consumption.

useBloodBag previously granted fixed500HP for free. Now debit actual character HP/MP reserves against missing effective vitals outside active battle, emit currentHp/currentMp and reserve update. Does not resurrect. Pet reserve recovery and buyBloodBag purchasing remain outside this patch.

Direct HP/MP potion and food handlers now retain items when no HP/MP is restored. Bulk recovery consumes only effective doses; reserve refills are single-use only to preserve callback semantics. Unsupported consumables remain explicitly rejected, never fake success.

Tests: application/item, domain/character, item/combat RTMP handlers pass; targeted vet and server build pass. Auth suite has daily VIP experience test failure14900 vs15400, unrelated to ep payload change. Live Flash reserve refresh still needs validation. Refunded exactly one bound2846 from confirmed log, receipt in outputs/VPT-Local/HOAN-TUI-MP-2026-11-14.json (workspace root).

Wire verification: main SWF onUPP method97056 reads payload.ep and calls Core.setBloodBag (multinames73446/73445); onUPP {ep:pipeString} is confirmed, not inferred. UI walkthrough remains pending.


## Pet fusion incident 2026-11-14
RPCpetJoin(20,21,24,18,10) at08:43:49 consumedpet21throughlegacy2petAbsorbPet; UIalwaysfailsbecausehandlerreturns success/pet insteadofflag/pid/grow. Verifiedpet21absent, otherfourremain. Temporarilydisablednormalandseniorfusionentrypointswith explicitnotice; no mutations through theseRPCs. Full5petfusionnotyetimplemented. Do not silentlyrestore guessedpetdata or labelguardasfullfix. Official Go guide /bai-viet/dung-hop/4287.html plusclient confirms4subpetsconsumedbothoutcomes, nextcolourrandomgrowth, rates100/80/50/30. Roll distributionnotpublished.


## Live item cache repair 2026-11-14
Craft,mix,star,gem,resolve and equipmentprefix/level/element/bind/soul handlers now send onCreateItemInstance(dto) before onAddCharactorSlot(dto) through sharedsendSlotUpdate. This hydrates changedremainingstacks/newproducts before slotpresentation. No businesslogic changes, no fusionunlock. Handler tests pass includingorderingtest; liveFlashverification pending. Fullyconsumedstacks retain existingonDelCharactorSlot pathway pendingcontractaudit.


Equipment distinction: DTOtype18 uses onChangeEquiptInst, ordinaryDTOtype28 uses onCreateItemInstance, bothbeforeonAddCharactorSlot. Regressiontestcoversboth. ABCauditconfirmedonDelCharactorSlot(itemID,SID) currentargumentorderisvalid; SID=-2 lookupalternativeexistsbutnotintroduced.


## Local fusion and star repair — 2026-11-14

Normal pet fusion now accepts all five instance IDs, validates ownership/uniqueness/species/element/colour/rest state and secondary equipment/souls before rolling. Rates100/80/50/30 from Flash and official Go guide. Success moves growth into next colour; failure preserves the main snapshot; both valid outcomes consume all four secondary pets. Senior accepts one ID array, exact blue4..10/purple4..16 tables and item3014 cost (blue1, purple2 per secondary). Both use atomic Postgres functions and update cache only after commit. onDelPet + onUpdatePet + derived property refresh and exact responder flag/pid/grow keep Flash current. Legacy two-pet handlers removed from RPC flow.

Material/jewel upgrades share validated exchange and return every touched product stack. Valid random failures consume inputs; grant/consume errors compensate. Materialrate=count*20, orangematerial=count*6+activeVIPright8; jewel=count*20 and nextJewelTid. Fully consumed bag source temporarily reserves a tempbag slot so full bags can use freed space. Limitation: compensation is not DB crash-atomic, and requires free temp slot on this success path.

Equipment star rate index corrected to target star. Production cache/Postgres path atomically updates equipment and consumes stones. Wing star accepts feather2906, returns equSlotId/itemSlotId/num/upgradeNum/flag, uses official failure penalty table (0,0,1,2,3,4,4,5,0,0). Random failures consume stones; technical commit errors do not commit either change.

WingJoin and WingSeniorJoin added with exact Flash request/result structures. Validate bag equipment kind13/position22, unique inputs, same colour; normal allows different wing templates with5point penalty each; senior requires same template. Normal purple VIPright9 supported. Atomic wing function commits main and all secondary deletions. Main ID retained. Success uses next colour and local equipment property generator, preserves main stars. Failure preserves main.

Compatibility limits: original Go server random distribution is unpublished. Pet growth uses uniform hundredth steps within documented colour ranges; wing stats use this repo equipment quality generator. Birthday event overrides/special seasonal rate buffs are not enabled by this patch. Wing experience growth/recast/bind/hole/feather operations beyond fusion and star have not been certified here. Do not claim whole game or exact original server compatibility.

Validation: focused normal/senior pet tests pass; full application/item, item/pet RTMP handlers, cache, Postgres packages pass; affected vet and server build pass. SQL rollback fixtures validate pet, seniorpet with split3014 stacks, wing and star success/technical rollback; no fixture data persists. Full pet/auth suites retain unrelated baseline failures, not represented as green. Live Flash walkthrough pending after deployment.

Sources: https://vuaphapthuat.go.vn/bai-viet/dung-hop/4287.html ; https://vuaphapthuat.go.vn/bai-viet/cuong-hoa-canh/4018.html ; read-only Flash PetFuncPanel/PetAdvancedPanel/WingFunc/EquipTFuncPanel and GamePredef.

## Advanced fusion stale material slot 2026-11-14
Screenshotshows3014stack22with13secondarypetsrequiring26, butserverinventoryinstance118has32. No seniorPetJoinRPCreceived; Flash advancedJoin early-returns when petItem.stackNum<itemNeedNum. Added onUpChaSlotStackNum(id,stackNum,sid) aftercacheprime/slotpushinitemhelperandshoppurchaseupdates to refreshfunctionslotquantities. Handler tests pass. Livevalidationpending. Wishbookcandidates3012and3755bothlackItemAwardrows; do not publishunusableboxes. Userprice/currencyquestionpending.


## Advanced pet button actual blocker 2026-11-14
Secondreporthad32visible3014vs26requiredandstillnoRPC. ClientadvancedJoin dereferencesselectedpet.petData.soulInfo.data. DBgenericgdc39replywasminimalandmissedsoulInfo; addedpetservicewiringtogamedatahandlerandownedlivegdc39singleDTOpathusingGetPetDetail.ToDetailDTO. RegressiontestexercisesRPCpathandrequiresnon-null soulInfo.data,binded,property. Earlierstackcallbackfixalonewasinsufficient. Buildandtests passed; liveFlashvalidationpending.
