//go:build with_db

package balance

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func TestBalanceMeleeRangeAndCasting(t *testing.T) {
	for _, form := range []struct {
		name    string
		talents string
	}{
		{name: "moonkin", talents: DefaultTalents},
		{name: "humanoid"},
	} {
		for _, tc := range []struct {
			name     string
			distance float64
			spellID  int32
			wantAuto bool
		}{
			{name: "at melee reach", distance: core.MaxMeleeRange, wantAuto: true},
			{name: "outside melee reach", distance: core.MaxMeleeRange + 0.01},
			{name: "hardcasting in melee", distance: core.MaxMeleeRange, spellID: 9912},
			{name: "instant casts in melee", distance: core.MaxMeleeRange, spellID: 9835, wantAuto: true},
		} {
			t.Run(form.name+"/"+tc.name, func(t *testing.T) {
				items := make([]*proto.ItemSpec, 17)
				for i := range items {
					items[i] = &proto.ItemSpec{}
				}
				items[14].Id = 2243
				rotation := &proto.APLRotation{Type: proto.APLRotation_TypeAPL}
				if tc.spellID != 0 {
					rotation.PriorityList = []*proto.APLListItem{{Action: &proto.APLAction{
						Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
							SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: tc.spellID}},
						}},
					}}}
				}
				player := &proto.Player{
					Name: "Balance melee", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf,
					Equipment: &proto.EquipmentSpec{Items: items}, TalentsString: form.talents,
					DistanceFromTarget: tc.distance, Rotation: rotation,
					Spec: &proto.Player_BalanceDruid{BalanceDruid: &proto.BalanceDruid{
						Options: &proto.BalanceDruid_Options{ClassOptions: &proto.DruidOptions{}},
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
						if tc.spellID != 0 && action.Id.GetSpellId() == tc.spellID {
							casts += target.Casts
						}
					}
				}
				if (autos > 0) != tc.wantAuto {
					t.Errorf("auto attacks = %d, want swings present = %t", autos, tc.wantAuto)
				}
				if tc.spellID != 0 && casts == 0 {
					t.Error("no spell completed, so the casting case did not run")
				}
			})
		}
	}
}
