//go:build with_db

package elemental

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestElementalMeleeRangeAndCasting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		distance float64
		casting  bool
		wantAuto bool
	}{
		{name: "at melee reach", distance: core.MaxMeleeRange, wantAuto: true},
		{name: "outside melee reach", distance: core.MaxMeleeRange + 0.01},
		{name: "casting in melee", distance: core.MaxMeleeRange, casting: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]*proto.ItemSpec, 17)
			for i := range items {
				items[i] = &proto.ItemSpec{}
			}
			items[14].Id = 2243
			rotation := &proto.APLRotation{Type: proto.APLRotation_TypeAPL}
			if tc.casting {
				rotation.PriorityList = []*proto.APLListItem{{Action: &proto.APLAction{
					Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
						SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 15208}},
					}},
				}}}
			}
			player := &proto.Player{
				Name: "Elemental melee", Class: proto.Class_ClassShaman, Race: proto.Race_RaceTroll,
				Equipment: &proto.EquipmentSpec{Items: items}, TalentsString: DefaultTalents,
				DistanceFromTarget: tc.distance, Rotation: rotation,
				Spec: &proto.Player_ElementalShaman{ElementalShaman: &proto.ElementalShaman{
					Options: &proto.ElementalShaman_Options{ClassOptions: &proto.ShamanOptions{}},
				}},
			}
			encounter := core.MakeSingleTargetEncounter(0)
			encounter.Duration = 5
			result := core.RunRaidSim(&proto.RaidSimRequest{
				Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
				Encounter: encounter, SimOptions: &proto.SimOptions{Iterations: 1, RandomSeed: 1},
			})
			if result.Error != nil {
				t.Fatal(result.Error.Message)
			}
			var autos, casts int32
			for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
				for _, target := range action.Targets {
					if action.Id.GetOtherId() == proto.OtherAction_OtherActionAttack && action.Id.Tag == 1 {
						autos += target.Casts
					}
					if action.Id.GetSpellId() == 15208 {
						casts += target.Casts
					}
				}
			}
			if (autos > 0) != tc.wantAuto {
				t.Errorf("auto attacks = %d, want swings present = %t", autos, tc.wantAuto)
			}
			if tc.casting && casts == 0 {
				t.Error("no Lightning Bolt completed, so casting did not constrain the swings")
			}
		})
	}
}
