package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestDistribution(t *testing.T) {
	var (
		voteYes = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionYes,
			Weight: sdk.NewDec(1),
		}}
		voteAbstain = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionAbstain,
			Weight: sdk.NewDec(1),
		}}
		voteNo = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionNo,
			Weight: sdk.NewDec(1),
		}}
		voteNoWithVeto = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionNoWithVeto,
			Weight: sdk.NewDec(1),
		}}
	)

	tests := []struct {
		name              string
		accounts          []Account
		expectedAddresses func(sdk.Dec) map[string]sdk.Dec
		expectedTotal     int64
		expectedUnstaked  int64
		expectedVotes     map[govtypes.VoteOption]int64
	}{
		{
			name: "direct votes",
			accounts: []Account{
				{
					Address:      "yes",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(2),
					Vote:         voteYes,
				},
				{
					Address:      "abstain",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(2),
					Vote:         voteAbstain,
				},
				{
					Address:      "no",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(2),
					Vote:         voteNo,
				},
				{
					Address:      "noWithVeto",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(2),
					Vote:         voteNoWithVeto,
				},
				{
					Address:      "didntVote",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(2),
					Delegations: []Delegation{{
						Amount: sdk.NewDec(2),
					}},
				},
			},
			expectedAddresses: func(blend sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"yes":        sdk.NewDec(1).Mul(blend.Mul(malus)).Add(sdk.NewDec(2)),
					"abstain":    sdk.NewDec(1).Mul(blend.Mul(malus)).Add(sdk.NewDec(2).Mul(blend)),
					"no":         sdk.NewDec(1).Mul(blend.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier)),
					"noWithVeto": sdk.NewDec(1).Mul(blend.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier).Mul(bonus)),
					"didntVote":  sdk.NewDec(1).Mul(blend.Mul(malus)).Add(sdk.NewDec(2).Mul(blend).Mul(malus)),
				}
			},
			expectedTotal:    45,
			expectedUnstaked: 15,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      6,
				govtypes.OptionYes:        2,
				govtypes.OptionAbstain:    6,
				govtypes.OptionNo:         8,
				govtypes.OptionNoWithVeto: 9,
			},
		},
		{
			name: "direct weighted votes",
			accounts: []Account{
				{
					Address:      "directWeightVote",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(18),
					Vote: govtypes.WeightedVoteOptions{
						{
							Option: govtypes.OptionYes,
							Weight: sdk.NewDecWithPrec(1, 1),
						},
						{
							Option: govtypes.OptionAbstain,
							Weight: sdk.NewDecWithPrec(2, 1),
						},
						{
							Option: govtypes.OptionNo,
							Weight: sdk.NewDecWithPrec(3, 1),
						},
						{
							Option: govtypes.OptionNoWithVeto,
							Weight: sdk.NewDecWithPrec(4, 1),
						},
					},
				},
			},
			expectedAddresses: func(blend sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"directWeightVote":
					// liquid amount
					sdk.NewDec(1).Mul(blend.Mul(malus)).
						// voted yes
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(1, 1))).
						// voted abstain
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(2, 1)).Mul(blend)).
						// voted no
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(3, 1)).Mul(noVotesMultiplier)).
						// voted noWithVeto
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(4, 1)).Mul(noVotesMultiplier).Mul(bonus)),
				}
			},
			expectedTotal:    70,
			expectedUnstaked: 4,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      0,
				govtypes.OptionYes:        2,
				govtypes.OptionAbstain:    14,
				govtypes.OptionNo:         22,
				govtypes.OptionNoWithVeto: 30,
			},
		},
		{
			name: "indirect votes",
			accounts: []Account{
				{
					Address:      "indirectVote",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(20),
					Vote:         nil,
					Delegations: []Delegation{
						// one deleg didn't vote
						{
							Amount: sdk.NewDec(2),
							Vote:   nil,
						},
						// one deleg voted yes
						{
							Amount: sdk.NewDec(3),
							Vote:   voteYes,
						},
						// one deleg voted abstain
						{
							Amount: sdk.NewDec(4),
							Vote:   voteAbstain,
						},
						// one deleg voted no
						{
							Amount: sdk.NewDec(5),
							Vote:   voteNo,
						},
						// one deleg voted noWithVeto
						{
							Amount: sdk.NewDec(6),
							Vote:   voteNoWithVeto,
						},
					},
				},
			},
			expectedAddresses: func(blend sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"indirectVote":
					// liquid amount
					sdk.NewDec(1).Mul(blend.Mul(malus)).
						// from deleg who didn't vote
						Add(sdk.NewDec(2).Mul(blend).Mul(malus)).
						// from deleg who voted yes
						Add(sdk.NewDec(3)).
						// from deleg who voted abstain
						Add(sdk.NewDec(4).Mul(blend)).
						// from deleg who voted no
						Add(sdk.NewDec(5).Mul(noVotesMultiplier)).
						// from deleg who voted noWithVeto
						Add(sdk.NewDec(6).Mul(noVotesMultiplier).Mul(bonus)),
				}
			},
			expectedTotal:    71,
			expectedUnstaked: 4,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      7,
				govtypes.OptionYes:        3,
				govtypes.OptionAbstain:    14,
				govtypes.OptionNo:         20,
				govtypes.OptionNoWithVeto: 25,
			},
		},
		{
			name: "indirect weighted votes",
			accounts: []Account{
				{
					Address:      "directWeightVote",
					LiquidAmount: sdk.NewDec(1),
					StakedAmount: sdk.NewDec(20),
					Vote:         nil,
					Delegations: []Delegation{
						// one deleg used a weighted vote
						{
							Amount: sdk.NewDec(18),
							Vote: govtypes.WeightedVoteOptions{
								{
									Option: govtypes.OptionYes,
									Weight: sdk.NewDecWithPrec(1, 1),
								},
								{
									Option: govtypes.OptionAbstain,
									Weight: sdk.NewDecWithPrec(2, 1),
								},
								{
									Option: govtypes.OptionNo,
									Weight: sdk.NewDecWithPrec(3, 1),
								},
								{
									Option: govtypes.OptionNoWithVeto,
									Weight: sdk.NewDecWithPrec(4, 1),
								},
							},
						},
						// one deleg voted no
						{
							Amount: sdk.NewDec(2),
							Vote:   voteNo,
						},
					},
				},
			},
			expectedAddresses: func(blend sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"directWeightVote":
					// liquid amount
					sdk.NewDec(1).Mul(blend.Mul(malus)).
						// voted yes
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(1, 1))).
						// voted abstain
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(2, 1)).Mul(blend)).
						// voted no
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(3, 1)).Mul(noVotesMultiplier)).
						Add(sdk.NewDec(2).Mul(noVotesMultiplier)).
						// voted noWithVeto
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(4, 1)).Mul(noVotesMultiplier).Mul(bonus)),
				}
			},
			expectedTotal:    78,
			expectedUnstaked: 4,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      0,
				govtypes.OptionYes:        2,
				govtypes.OptionAbstain:    14,
				govtypes.OptionNo:         30,
				govtypes.OptionNoWithVeto: 30,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			fmt.Println(sdk.NewDecWithPrec(55, 1).Mul(malus))

			airdrop, err := distribution(tt.accounts)

			require.NoError(err)
			expectedRes := tt.expectedAddresses(airdrop.blend)
			assert.Equal(len(expectedRes), len(airdrop.addresses), "unexpected number of res")
			for k, v := range airdrop.addresses {
				ev, ok := expectedRes[k]
				if assert.True(ok, "unexpected address '%s'", k) {
					assert.Equal(ev.TruncateInt(), v, "unexpected airdrop amount for address '%s'", k)
				}
			}
			assert.Equal(tt.expectedTotal, airdrop.total.Ceil().RoundInt64(), "unexpected airdrop.total")
			assert.Equal(tt.expectedUnstaked, airdrop.unstaked.Ceil().RoundInt64(), "unexpected airdrop.unstaked")
			for _, v := range voteOptions {
				assert.Equal(tt.expectedVotes[v], airdrop.votes[v].Ceil().RoundInt64(), "unexpected airdrop.votes[%s]", v)
			}
		})
	}
}
