set check_function_bodies = off;

CREATE OR REPLACE FUNCTION player.commit_pet_fusion(p_character_id bigint, p_main_id bigint, p_material_ids bigint[], p_main jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 SET search_path TO ''
AS $function$
declare n integer;
begin
 if cardinality(p_material_ids) <> 4 or p_main_id = any(p_material_ids) or (select count(distinct x) from unnest(p_material_ids) x) <> 4 then
  raise exception 'Invalid fusion pet set';
 end if;
 perform id from player.character_pets where character_id=p_character_id and (id=p_main_id or id=any(p_material_ids)) order by id for update;
 get diagnostics n = row_count;
 if n <> 5 then raise exception 'Fusion pets changed or not owned'; end if;
 update player.character_pets set grow_rate=(p_main->>'GrowRate')::double precision,
 grow_rate_add=(p_main->>'GrowRateAdd')::double precision,
 property=p_main->'Property', current_hp=(p_main->>'CurrentHP')::integer,
 current_mp=(p_main->>'CurrentMP')::integer,max_hp=(p_main->>'MaxHP')::integer,
 max_mp=(p_main->>'MaxMP')::integer,updated_at=now()
 where id=p_main_id and character_id=p_character_id;
 delete from player.character_pets where id=any(p_material_ids) and character_id=p_character_id;
 get diagnostics n = row_count;
 if n <> 4 then raise exception 'Fusion material deletion incomplete'; end if;
 return true;
end $function$
;
