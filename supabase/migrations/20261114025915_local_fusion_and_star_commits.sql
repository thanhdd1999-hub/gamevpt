set check_function_bodies = off;

CREATE OR REPLACE FUNCTION player.commit_equipment_star(p_char bigint, p_equipment jsonb, p_stone jsonb, p_count integer)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare eid bigint:=(p_equipment->>'ID')::bigint; sid bigint:=(p_stone->>'ID')::bigint; total integer:=(p_stone->>'StackCount')::integer; tid integer:=(p_stone->>'TemplateID')::integer; star integer:=(p_equipment->>'StarLevel')::integer; n integer;
begin
 if eid=sid or p_count<1 or p_count>5 or total<p_count or star<0 or star>10 or tid not in(5,2906) then raise exception 'Invalid star attempt';end if;
 perform id from player.character_items where character_id=p_char and id in(eid,sid) order by id for update;
 get diagnostics n=row_count;if n<>2 then raise exception 'Star input changed';end if;
 if not exists(select 1 from player.character_items where id=eid and item_type=2) or not exists(select 1 from player.character_items where id=sid and template_id=tid and slot_type in(0,3,5)) then raise exception 'Invalid star inputs';end if;
 update player.character_items set star_level=star,properties=p_equipment->'Properties',updated_at=now() where id=eid;
 if total=p_count then delete from player.character_items where id=sid;else update player.character_items set stack_count=total-p_count,updated_at=now() where id=sid;end if;
 return true;
end $function$
;

CREATE OR REPLACE FUNCTION player.commit_senior_pet_fusion(p_character_id bigint, p_main_id bigint, p_material_ids bigint[], p_cost integer, p_main jsonb, p_items jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare n integer; needed integer := p_cost; row_data jsonb; item_id bigint; amount integer; take integer;
begin
 if cardinality(p_material_ids) < 4 or cardinality(p_material_ids) > 16 or p_main_id=any(p_material_ids) or (select count(distinct x) from unnest(p_material_ids) x)<>cardinality(p_material_ids) or (p_cost<>1 and p_cost<>2*cardinality(p_material_ids)) then raise exception 'Invalid senior fusion input'; end if;
 perform id from player.character_pets where character_id=p_character_id and (id=p_main_id or id=any(p_material_ids)) order by id for update;
 get diagnostics n = row_count;
 if n<>cardinality(p_material_ids)+1 then raise exception 'Fusion pets changed or not owned'; end if;
 if jsonb_array_length(p_items) < 1 or (select count(distinct (v->>'ID')::bigint) from jsonb_array_elements(p_items) v) <> jsonb_array_length(p_items) then raise exception 'Invalid fusion item set'; end if;
 for row_data in select v from jsonb_array_elements(p_items) v order by (v->>'ID')::bigint loop
  item_id := (row_data->>'ID')::bigint;
  amount := (row_data->>'StackCount')::integer;
  if amount<=0 or needed<=0 then raise exception 'Invalid fusion item count'; end if;
  perform id from player.character_items where id=item_id and character_id=p_character_id and template_id=3014 and slot_type in(0,5) for update;
  if not found then raise exception 'Fusion item changed or not owned'; end if;
  take := least(needed,amount);
  if take=amount then delete from player.character_items where id=item_id;
  else update player.character_items set stack_count=amount-take,updated_at=now() where id=item_id;
  end if;
  needed := needed-take;
 end loop;
 if needed<>0 then raise exception 'Not enough fusion items'; end if;
 update player.character_pets set grow_rate=(p_main->>'GrowRate')::double precision,grow_rate_add=(p_main->>'GrowRateAdd')::double precision,property=p_main->'Property',current_hp=(p_main->>'CurrentHP')::integer,current_mp=(p_main->>'CurrentMP')::integer,max_hp=(p_main->>'MaxHP')::integer,max_mp=(p_main->>'MaxMP')::integer,updated_at=now() where id=p_main_id and character_id=p_character_id;
 delete from player.character_pets where id=any(p_material_ids) and character_id=p_character_id;
 get diagnostics n = row_count;
 if n<>cardinality(p_material_ids) then raise exception 'Fusion material deletion incomplete'; end if;
 return true;
end $function$
;

CREATE OR REPLACE FUNCTION player.commit_wing_fusion(p_character_id bigint, p_main_id bigint, p_material_ids bigint[], p_main jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare n integer; done boolean;
begin
 if p_character_id is null or p_main_id is null or p_material_ids is null or cardinality(p_material_ids)<4 or cardinality(p_material_ids)>16 or p_main_id=any(p_material_ids) or (select count(distinct x) from unnest(p_material_ids) x)<>cardinality(p_material_ids) then raise exception 'Invalid wing fusion pet set'; end if;
 perform id from player.character_items where character_id=p_character_id and item_type=2 and slot_type=0 and stack_count=1 and (id=p_main_id or id=any(p_material_ids)) order by id for update;
 get diagnostics n=row_count;
 if n<>cardinality(p_material_ids)+1 then raise exception 'Wing items changed, equipped, or not owned'; end if;
 done:=player.update_character_item(p_main_id,(p_main->>'TemplateID')::integer,(p_main->>'ItemType')::integer,(p_main->>'SlotType')::integer,(p_main->>'SlotIndex')::integer,(p_main->>'StackCount')::integer,(p_main->>'IsBound')::boolean,(p_main->>'Durability')::integer,(p_main->>'MaxDurability')::integer,(p_main->>'EnchantLevel')::integer,(p_main->>'StarLevel')::integer,(p_main->>'ColorCode')::integer,p_main->'Properties',now());
 if not done then raise exception 'Main wing changed'; end if;
 delete from player.character_items where id=any(p_material_ids) and character_id=p_character_id;
 get diagnostics n=row_count;
 if n<>cardinality(p_material_ids) then raise exception 'Wing material deletion incomplete'; end if;
 return true;
end $function$
;
