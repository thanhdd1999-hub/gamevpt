set check_function_bodies = off;

CREATE OR REPLACE FUNCTION player.commit_pet_evolution_action(p_owner bigint, p_pet jsonb, p_character jsonb, p_items jsonb, p_np_cost integer, p_gold_cost bigint)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare pid bigint:=(p_pet->>'ID')::bigint;iid bigint;total integer;qty integer;v jsonb;item_data jsonb;done boolean;
begin
 if p_owner is null or pid is null or (p_character->>'ID')::bigint<>p_owner or p_np_cost<0 or p_gold_cost<0 or (p_character->>'Gold')::bigint<0 or (p_character->>'NpPoint')::integer<0 then raise exception 'Invalid pet evolution action';end if;
 perform id from player.character_pets where id=pid and character_id=p_owner for update;
 if not found then raise exception 'Pet no longer owned';end if;
 perform character_id from player.character_wallet where character_id=p_owner for update;
 if not found then raise exception 'Character wallet missing';end if;
 perform character_id from player.character_feature_states where character_id=p_owner for update;
 if not found then raise exception 'Character feature state missing';end if;
 if (select count(distinct(x->'Item'->>'ID')::bigint) from jsonb_array_elements(p_items)x)<>jsonb_array_length(p_items) then raise exception 'Duplicate evolution protection item';end if;
 for v in select value from jsonb_array_elements(p_items) loop
  item_data:=v->'Item';iid:=(item_data->>'ID')::bigint;total:=(item_data->>'StackCount')::integer;qty:=(v->>'Count')::integer;
  if qty<1 or qty>total or (item_data->>'TemplateID')::integer<>3911 then raise exception 'Invalid feather protection count';end if;
  perform id from player.character_items where id=iid and character_id=p_owner and template_id=3911 and slot_type in(0,5) for update;
  if not found then raise exception 'Feather protection changed or not owned';end if;
  if qty=total then delete from player.character_items where id=iid;else update player.character_items set stack_count=total-qty,updated_at=now() where id=iid;end if;
 end loop;
 done:=player.update_character_pet(pid,(p_pet->>'TemplateID')::integer,p_pet->>'Name',(p_pet->>'Level')::integer,(p_pet->>'Experience')::bigint,(p_pet->>'AptStrength')::integer,(p_pet->>'AptAgility')::integer,(p_pet->>'AptStamina')::integer,(p_pet->>'AptIntelligence')::integer,(p_pet->>'AptEnergy')::integer,(p_pet->>'AptStrengthEx')::integer,(p_pet->>'AptAgilityEx')::integer,(p_pet->>'AptStaminaEx')::integer,(p_pet->>'AptIntelligenceEx')::integer,(p_pet->>'AptEnergyEx')::integer,(p_pet->>'GrowRate')::double precision,(p_pet->>'GrowRateAdd')::double precision,(p_pet->>'UpgradeNum')::integer,(p_pet->>'EvolutionLv')::integer,(p_pet->>'Element')::integer,(p_pet->>'CurrentHP')::integer,(p_pet->>'CurrentMP')::integer,(p_pet->>'MaxHP')::integer,(p_pet->>'MaxMP')::integer,(p_pet->>'IsFollowing')::boolean,(p_pet->>'IsMounting')::boolean,p_pet->'Property',now(),(p_pet->>'AttrPoints')::integer,(p_pet->>'MaxAttrPoints')::integer,(p_pet->>'DistributedAttrPoints')::integer,(p_pet->>'Life')::integer);
 if not done then raise exception 'Pet evolution update failed';end if;
 update player.character_wallet set gold=(p_character->>'Gold')::bigint where character_id=p_owner;
 update player.character_feature_states set pm_process_data=p_character->'PMProcessData',pm_exp=(p_character->>'PMExp')::bigint where character_id=p_owner;
 return true;
end $function$
;

CREATE OR REPLACE FUNCTION player.commit_pet_illusion_attempt(p_char bigint, p_pet jsonb, p_sub bigint, p_item jsonb, p_count integer)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare pid bigint:=(p_pet->>'ID')::bigint; iid bigint;total integer;done boolean;
begin
 if p_char is null or pid is null or p_sub is null or pid=p_sub or p_count<0 or p_count>5 then raise exception 'Invalid illusion';end if;
 perform id from player.character_pets where id in(pid,p_sub) and character_id=p_char order by id for update;
 if (select count(*) from player.character_pets where id in(pid,p_sub) and character_id=p_char)<>2 then raise exception 'Pets changed';end if;
 if p_count>0 then
 iid:=(p_item->>'ID')::bigint;total:=(p_item->>'StackCount')::integer;
 perform id from player.character_items where id=iid and character_id=p_char and template_id=(p_item->>'TemplateID')::integer and stack_count=total and total>=p_count and slot_type in(0,5) for update;
 if not found then raise exception 'Luck item changed';end if;
 end if;
 done:=player.update_character_pet(pid,(p_pet->>'TemplateID')::integer,p_pet->>'Name',(p_pet->>'Level')::integer,(p_pet->>'Experience')::bigint,(p_pet->>'AptStrength')::integer,(p_pet->>'AptAgility')::integer,(p_pet->>'AptStamina')::integer,(p_pet->>'AptIntelligence')::integer,(p_pet->>'AptEnergy')::integer,(p_pet->>'AptStrengthEx')::integer,(p_pet->>'AptAgilityEx')::integer,(p_pet->>'AptStaminaEx')::integer,(p_pet->>'AptIntelligenceEx')::integer,(p_pet->>'AptEnergyEx')::integer,(p_pet->>'GrowRate')::double precision,(p_pet->>'GrowRateAdd')::double precision,(p_pet->>'UpgradeNum')::integer,(p_pet->>'EvolutionLv')::integer,(p_pet->>'Element')::integer,(p_pet->>'CurrentHP')::integer,(p_pet->>'CurrentMP')::integer,(p_pet->>'MaxHP')::integer,(p_pet->>'MaxMP')::integer,(p_pet->>'IsFollowing')::boolean,(p_pet->>'IsMounting')::boolean,p_pet->'Property',now(),(p_pet->>'AttrPoints')::integer,(p_pet->>'MaxAttrPoints')::integer,(p_pet->>'DistributedAttrPoints')::integer,(p_pet->>'Life')::integer);
 if not done then raise exception 'Illusion pet update failed';end if;
 delete from player.character_pets where id=p_sub and character_id=p_char;
 if p_count>0 then
 if total=p_count then delete from player.character_items where id=iid;else update player.character_items set stack_count=total-p_count,updated_at=now() where id=iid;end if;
 end if;
 return true;
end $function$
;

CREATE OR REPLACE FUNCTION player.commit_pet_item_action(p_char bigint, p_pet jsonb, p_item jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare pid bigint:=(p_pet->>'ID')::bigint;iid bigint:=(p_item->>'ID')::bigint;total integer:=(p_item->>'StackCount')::integer;done boolean;
begin
 if p_char is null or pid is null or iid is null or total<1 then raise exception 'Invalid pet item action';end if;
 perform id from player.character_pets where id=pid and character_id=p_char for update;
 if not found then raise exception 'Pet no longer owned';end if;
 perform i.id from player.character_items i join data.data_tbl_item_template t on t.id=i.template_id where i.id=iid and i.character_id=p_char and i.template_id=(p_item->>'TemplateID')::integer and i.slot_type in(0,5) and t.type=507 for update of i;
 if not found then raise exception 'Pet action item changed or not owned';end if;
 done:=player.update_character_pet(pid,(p_pet->>'TemplateID')::integer,p_pet->>'Name',(p_pet->>'Level')::integer,(p_pet->>'Experience')::bigint,(p_pet->>'AptStrength')::integer,(p_pet->>'AptAgility')::integer,(p_pet->>'AptStamina')::integer,(p_pet->>'AptIntelligence')::integer,(p_pet->>'AptEnergy')::integer,(p_pet->>'AptStrengthEx')::integer,(p_pet->>'AptAgilityEx')::integer,(p_pet->>'AptStaminaEx')::integer,(p_pet->>'AptIntelligenceEx')::integer,(p_pet->>'AptEnergyEx')::integer,(p_pet->>'GrowRate')::double precision,(p_pet->>'GrowRateAdd')::double precision,(p_pet->>'UpgradeNum')::integer,(p_pet->>'EvolutionLv')::integer,(p_pet->>'Element')::integer,(p_pet->>'CurrentHP')::integer,(p_pet->>'CurrentMP')::integer,(p_pet->>'MaxHP')::integer,(p_pet->>'MaxMP')::integer,(p_pet->>'IsFollowing')::boolean,(p_pet->>'IsMounting')::boolean,p_pet->'Property',now(),(p_pet->>'AttrPoints')::integer,(p_pet->>'MaxAttrPoints')::integer,(p_pet->>'DistributedAttrPoints')::integer,(p_pet->>'Life')::integer);
 if not done then raise exception 'Pet item action failed';end if;
 if total=1 then delete from player.character_items where id=iid;else update player.character_items set stack_count=total-1,updated_at=now() where id=iid;end if;
 return true;
end $function$
;

CREATE OR REPLACE FUNCTION player.commit_pet_star_attempt(p_char bigint, p_pet jsonb, p_items jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare pid bigint:=(p_pet->>'ID')::bigint; iid bigint; total integer; v jsonb; n integer; idx integer:=0;
begin
 if p_char is null or pid is null or jsonb_array_length(p_items) not in(1,2) or (p_pet->>'UpgradeNum')::integer not between 0 and 12 then raise exception 'Invalid pet star input';end if;
 perform id from player.character_pets where id=pid and character_id=p_char for update;
 if not found then raise exception 'Pet no longer owned';end if;
 if (select count(distinct(x->>'ID')::bigint) from jsonb_array_elements(p_items)x)<>jsonb_array_length(p_items) then raise exception 'Duplicate pet star item';end if;
 for v in select value from jsonb_array_elements(p_items) loop
  iid:=(v->>'ID')::bigint;total:=(v->>'StackCount')::integer;idx:=idx+1;
  if total<1 then raise exception 'Invalid pet star item count';end if;
  perform i.id from player.character_items i join data.data_tbl_item_template t on t.id=i.template_id where i.id=iid and i.character_id=p_char and i.template_id=(v->>'TemplateID')::integer and i.slot_type in(0,5) and t.type=507 and t.prop_type=(case when idx=1 then 3 else 6 end) for update of i;
  if not found then raise exception 'Pet star item changed or not owned';end if;
  if total=1 then delete from player.character_items where id=iid;else update player.character_items set stack_count=total-1,updated_at=now() where id=iid;end if;
 end loop;
 update player.character_pets set upgrade_num=(p_pet->>'UpgradeNum')::integer,grow_rate_add=(p_pet->>'GrowRateAdd')::double precision,property=p_pet->'Property',current_hp=(p_pet->>'CurrentHP')::integer,current_mp=(p_pet->>'CurrentMP')::integer,max_hp=(p_pet->>'MaxHP')::integer,max_mp=(p_pet->>'MaxMP')::integer,updated_at=now() where id=pid;
 return true;
end $function$
;
