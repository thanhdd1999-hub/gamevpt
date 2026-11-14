package combat

import "testing"

func TestAttackRetargetDeadEnemy(t *testing.T) {
	battle := NewBattle(BattleTypePVE, 20)
	actor := NewParticipant("player", "player", SidePlayer, false)
	actor.Position = 10
	actor.Attack = 120
	actor.Hit = 999
	battle.AddParticipant(actor)
	dead := NewParticipant("dead", "dead", SideEnemy, true)
	dead.Position = 2
	dead.IsAlive = false
	dead.CurrentHP = 0
	battle.AddParticipant(dead)
	live := NewParticipant("live", "live", SideEnemy, true)
	live.Position = 0
	live.CurrentHP = 500
	live.MaxHP = 500
	live.IsAlive = true
	battle.AddParticipant(live)
	tp := NewTurnProcessor(battle, nil)
	for _, target := range []string{dead.ID, "2"} {
		if got := tp.resolveAttackTarget(actor, target); got != live {
			t.Fatalf("target %s: got %v, want live enemy", target, got)
		}
	}
	if tp.resolveAttackTarget(actor, actor.ID) != nil || tp.resolveAttackTarget(actor, "missing") != nil {
		t.Fatal("invalid or allied target must not be retargeted")
	}
	if tp.resolveTarget(actor, dead.ID) != nil {
		t.Fatal("shared target resolver must retain dead-target rejection")
	}
	actions := tp.executeAttack(actor, &BattleCommand{ActorID: actor.ID, TargetID: dead.ID, ActionType: ClientActionAttack})
	if len(actions) == 0 || live.CurrentHP >= 500 {
		t.Fatal("attack did not damage replacement enemy")
	}
	live.IsAlive = false
	if tp.resolveAttackTarget(actor, dead.ID) != nil {
		t.Fatal("must not retarget when no enemy remains alive")
	}
}
