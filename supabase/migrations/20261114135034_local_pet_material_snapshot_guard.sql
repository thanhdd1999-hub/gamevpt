set check_function_bodies = off;

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
 perform i.id from player.character_items i join data.data_tbl_item_template t on t.id=i.template_id where i.id=iid and i.character_id=p_char and i.template_id=(p_item->>'TemplateID')::integer and i.stack_count=total and i.slot_type in(0,5) and t.type=507 for update of i;
 if not found then raise exception 'Pet action item changed or not owned';end if;
 done:=player.update_character_pet(pid,(p_pet->>'TemplateID')::integer,p_pet->>'Name',(p_pet->>'Level')::integer,(p_pet->>'Experience')::bigint,(p_pet->>'AptStrength')::integer,(p_pet->>'AptAgility')::integer,(p_pet->>'AptStamina')::integer,(p_pet->>'AptIntelligence')::integer,(p_pet->>'AptEnergy')::integer,(p_pet->>'AptStrengthEx')::integer,(p_pet->>'AptAgilityEx')::integer,(p_pet->>'AptStaminaEx')::integer,(p_pet->>'AptIntelligenceEx')::integer,(p_pet->>'AptEnergyEx')::integer,(p_pet->>'GrowRate')::double precision,(p_pet->>'GrowRateAdd')::double precision,(p_pet->>'UpgradeNum')::integer,(p_pet->>'EvolutionLv')::integer,(p_pet->>'Element')::integer,(p_pet->>'CurrentHP')::integer,(p_pet->>'CurrentMP')::integer,(p_pet->>'MaxHP')::integer,(p_pet->>'MaxMP')::integer,(p_pet->>'IsFollowing')::boolean,(p_pet->>'IsMounting')::boolean,p_pet->'Property',now(),(p_pet->>'AttrPoints')::integer,(p_pet->>'MaxAttrPoints')::integer,(p_pet->>'DistributedAttrPoints')::integer,(p_pet->>'Life')::integer);
 if not done then raise exception 'Pet item action failed';end if;
 if total=1 then delete from player.character_items where id=iid;else update player.character_items set stack_count=total-1,updated_at=now() where id=iid;end if;
 return true;
end $function$
;
