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
		simpleCaseBlend = sdk.NewDecWithPrec(225, 2)
	)

	tests := []struct {
		name        string
		accounts    []Account
		expectedRes map[string]sdk.Dec
	}{
		{
			name: "simple case",
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
			expectedRes: map[string]sdk.Dec{
				"yes":        sdk.NewDec(1).Add(sdk.NewDec(2)),
				"abstain":    sdk.NewDec(1).Add(sdk.NewDec(2).Mul(simpleCaseBlend)),
				"no":         sdk.NewDec(1).Add(sdk.NewDec(2).Mul(noMultiplier)),
				"noWithVeto": sdk.NewDec(1).Add(sdk.NewDec(2).Mul(noMultiplier).Mul(bonus)),
				"didntVote":  sdk.NewDec(1).Add(sdk.NewDec(2).Mul(simpleCaseBlend).Mul(malus)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			fmt.Println(sdk.NewDecWithPrec(55, 1).Mul(malus))

			res, err := distribution(tt.accounts)

			require.NoError(err)
			assert.Equal(len(tt.expectedRes), len(res), "unexpected number of res")
			for k, v := range res {
				ev, ok := tt.expectedRes[k]
				if assert.True(ok, "unexpected address '%s'", k) {
					assert.Equal(ev.String(), v.String(), "unexpected airdrop amount for address '%s'", k)
				}
			}
		})
	}
}
