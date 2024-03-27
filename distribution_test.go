package main

import (
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
		noVotesMultiplier = defaultDistriParams.noVotesMultiplier
		bonus             = defaultDistriParams.bonus
		malus             = defaultDistriParams.malus
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
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(20),
					Vote:         voteYes,
				},
				{
					Address:      "abstain",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(20),
					Vote:         voteAbstain,
				},
				{
					Address:      "no",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(20),
					Vote:         voteNo,
				},
				{
					Address:      "noWithVeto",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(20),
					Vote:         voteNoWithVeto,
				},
				{
					Address:      "didntVote",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(20),
					Delegations: []Delegation{{
						Amount: sdk.NewDec(20),
					}},
				},
			},
			expectedAddresses: func(nonVotersMult sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"yes":        sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2)),
					"abstain":    sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(nonVotersMult)),
					"no":         sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier)),
					"noWithVeto": sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier).Mul(bonus)),
					"didntVote":  sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(nonVotersMult).Mul(malus)),
				}
			},
			expectedTotal:    27,
			expectedUnstaked: 5,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      2,
				govtypes.OptionYes:        2,
				govtypes.OptionAbstain:    2,
				govtypes.OptionNo:         8,
				govtypes.OptionNoWithVeto: 8,
			},
		},
		{
			name: "direct votes with small bags",
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
			expectedAddresses: func(nonVotersMult sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"no":         sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier)).QuoInt64(10),
					"noWithVeto": sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).Add(sdk.NewDec(2).Mul(noVotesMultiplier).Mul(bonus)).QuoInt64(10),
				}
			},
			expectedTotal:    3,
			expectedUnstaked: 0,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      0,
				govtypes.OptionYes:        0,
				govtypes.OptionAbstain:    0,
				govtypes.OptionNo:         1,
				govtypes.OptionNoWithVeto: 1,
			},
		},
		{
			name: "direct weighted votes",
			accounts: []Account{
				{
					Address:      "directWeightVote",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(180),
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
			expectedAddresses: func(nonVotersMult sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"directWeightVote":
					// liquid amount
					sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).
						// voted yes
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(1, 1))).
						// voted abstain
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(2, 1)).Mul(nonVotersMult)).
						// voted no
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(3, 1)).Mul(noVotesMultiplier)).
						// voted noWithVeto
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(4, 1)).Mul(noVotesMultiplier).Mul(bonus)),
				}
			},
			expectedTotal:    79,
			expectedUnstaked: 5,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      0,
				govtypes.OptionYes:        2,
				govtypes.OptionAbstain:    20,
				govtypes.OptionNo:         22,
				govtypes.OptionNoWithVeto: 30,
			},
		},
		{
			name: "indirect votes",
			accounts: []Account{
				{
					Address:      "indirectVote",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(200),
					Vote:         nil,
					Delegations: []Delegation{
						// one deleg didn't vote
						{
							Amount: sdk.NewDec(20),
							Vote:   nil,
						},
						// one deleg voted yes
						{
							Amount: sdk.NewDec(30),
							Vote:   voteYes,
						},
						// one deleg voted abstain
						{
							Amount: sdk.NewDec(40),
							Vote:   voteAbstain,
						},
						// one deleg voted no
						{
							Amount: sdk.NewDec(50),
							Vote:   voteNo,
						},
						// one deleg voted noWithVeto
						{
							Amount: sdk.NewDec(60),
							Vote:   voteNoWithVeto,
						},
					},
				},
			},
			expectedAddresses: func(nonVotersMult sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"indirectVote":
					// liquid amount
					sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).
						// from deleg who didn't vote
						Add(sdk.NewDec(2).Mul(nonVotersMult).Mul(malus)).
						// from deleg who voted yes
						Add(sdk.NewDec(3)).
						// from deleg who voted abstain
						Add(sdk.NewDec(4).Mul(nonVotersMult)).
						// from deleg who voted no
						Add(sdk.NewDec(5).Mul(noVotesMultiplier)).
						// from deleg who voted noWithVeto
						Add(sdk.NewDec(6).Mul(noVotesMultiplier).Mul(bonus)),
				}
			},
			expectedTotal:    71,
			expectedUnstaked: 3,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      6,
				govtypes.OptionYes:        3,
				govtypes.OptionAbstain:    13,
				govtypes.OptionNo:         20,
				govtypes.OptionNoWithVeto: 25,
			},
		},
		{
			name: "indirect weighted votes",
			accounts: []Account{
				{
					Address:      "directWeightVote",
					LiquidAmount: sdk.NewDec(10),
					StakedAmount: sdk.NewDec(330),
					Vote:         nil,
					Delegations: []Delegation{
						// one deleg used a weighted vote
						{
							Amount: sdk.NewDec(180),
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
						// one other deleg used a weighted vote
						{
							Amount: sdk.NewDec(100),
							Vote: govtypes.WeightedVoteOptions{
								{
									Option: govtypes.OptionYes,
									Weight: sdk.NewDecWithPrec(4, 1),
								},
								{
									Option: govtypes.OptionAbstain,
									Weight: sdk.NewDecWithPrec(6, 1),
								},
							},
						},
						// one deleg voted no
						{
							Amount: sdk.NewDec(20),
							Vote:   voteNo,
						},
						// one deleg didn't vote
						{
							Amount: sdk.NewDec(30),
							Vote:   nil,
						},
					},
				},
			},
			expectedAddresses: func(nonVotersMult sdk.Dec) map[string]sdk.Dec {
				return map[string]sdk.Dec{
					"directWeightVote":
					// liquid amount
					sdk.NewDec(1).Mul(nonVotersMult.Mul(malus)).
						// voted yes
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(1, 1))).
						Add(sdk.NewDec(10).Mul(sdk.NewDecWithPrec(4, 1))).
						// voted abstain
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(2, 1)).Mul(nonVotersMult)).
						Add(sdk.NewDec(10).Mul(sdk.NewDecWithPrec(6, 1)).Mul(nonVotersMult)).
						// voted no
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(3, 1)).Mul(noVotesMultiplier)).
						Add(sdk.NewDec(2).Mul(noVotesMultiplier)).
						// voted noWithVeto
						Add(sdk.NewDec(18).Mul(sdk.NewDecWithPrec(4, 1)).Mul(noVotesMultiplier).Mul(bonus)).
						// didn't vote
						Add(sdk.NewDec(3).Mul(nonVotersMult.Mul(malus))),
				}
			},
			expectedTotal:    96,
			expectedUnstaked: 2,
			expectedVotes: map[govtypes.VoteOption]int64{
				govtypes.OptionEmpty:      7,
				govtypes.OptionYes:        6,
				govtypes.OptionAbstain:    22,
				govtypes.OptionNo:         30,
				govtypes.OptionNoWithVeto: 30,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			airdrop, err := distribution(tt.accounts)

			require.NoError(err)
			expectedRes := tt.expectedAddresses(airdrop.nonVotersMultiplier)
			assert.Equal(len(expectedRes), len(airdrop.addresses), "unexpected number of res")
			for k, v := range airdrop.addresses {
				ev, ok := expectedRes[k]
				if assert.True(ok, "unexpected address '%s' balance '%s'", k, v) {
					assert.Equal(ev.RoundInt64(), v.Int64(), "unexpected airdrop amount for address '%s'", k)
				}
			}
			assert.Equal(tt.expectedTotal, airdrop.atone.supply.RoundInt64(), "unexpected airdrop.total")
			assert.Equal(tt.expectedUnstaked, airdrop.atone.unstaked.RoundInt64(), "unexpected airdrop.unstaked")
			for _, v := range allVoteOptions {
				assert.Equal(tt.expectedVotes[v], airdrop.atone.votes[v].RoundInt64(), "unexpected airdrop.votes[%s]", v)
			}
		})
	}
}
