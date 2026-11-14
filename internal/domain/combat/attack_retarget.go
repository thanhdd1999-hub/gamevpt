package combat

func (tp *TurnProcessor) resolveAttackTarget(actor *Participant, targetID string) *Participant {
	predicate := func(p *Participant) bool {
		return p != nil && p.Side == getOppositeSide(actor.Side)
	}
	byID := tp.battle.GetParticipant(targetID)
	byPosition := participantByPositionString(tp.battle, targetID)
	if chooseTargetCandidate(byID, byPosition, predicate, true) != nil {
		return tp.resolveTarget(actor, targetID)
	}
	previous := chooseTargetCandidate(byID, byPosition, predicate, false)
	if previous == nil || previous.IsAlive {
		return nil
	}
	var replacement *Participant
	for _, candidate := range tp.battle.GetAliveParticipants(getOppositeSide(actor.Side)) {
		if replacement == nil || candidate.Position < replacement.Position ||
			(candidate.Position == replacement.Position && candidate.ID < replacement.ID) {
			replacement = candidate
		}
	}
	if replacement == nil {
		return nil
	}
	return tp.resolveTarget(actor, replacement.ID)
}
