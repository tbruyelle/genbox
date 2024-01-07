package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestGetAccountVotes(t *testing.T) {
	var (
		accAddrs = createAccountAddrs(2)
		accAddr1 = accAddrs[0].String()
		// voterAddr2 = accAddrs[1]
		valAddrs    = createValidatorAddrs(2)
		valAddr1    = valAddrs[0].String()
		valAccAddr1 = sdk.AccAddress(valAddrs[0]).String()
		val1        = govtypes.ValidatorGovInfo{
			Address:             valAddrs[0],
			BondedTokens:        sdk.NewInt(1000000),
			DelegatorShares:     sdk.NewDec(1000000),
			DelegatorDeductions: sdk.ZeroDec(),
		}
		// valAddr2   = valAddrs[1]
		// Some votes
		voteYes = govtypes.Vote{
			Options: []govtypes.WeightedVoteOption{{
				Option: govtypes.OptionYes,
				Weight: sdk.NewDec(1),
			}},
		}
		voteNo = govtypes.Vote{
			Options: []govtypes.WeightedVoteOption{{
				Option: govtypes.OptionNo,
				Weight: sdk.NewDec(1),
			}},
		}
	)
	tests := []struct {
		name                 string
		delegsByAddr         map[string][]stakingtypes.Delegation
		votesByAddr          map[string]govtypes.Vote
		valsByAddr           map[string]govtypes.ValidatorGovInfo
		expectedAccountVotes []AccountVote
	}{
		{
			name:                 "no delegation",
			expectedAccountVotes: []AccountVote{},
		},
		{
			name: "one delegation without active validator",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			expectedAccountVotes: []AccountVote{},
		},
		{
			name: "one delegation with validator didn't vote",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
			},
			expectedAccountVotes: []AccountVote{},
		},
		{
			name: "one delegation with validator: inherit vote",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
			},
			votesByAddr: map[string]govtypes.Vote{
				valAccAddr1: voteNo,
			},
			expectedAccountVotes: []AccountVote{
				{
					Address: accAddr1,
					Power:   sdk.NewDec(1000),
					Vote:    voteNo,
				},
				{
					Address: valAccAddr1,
					Power:   sdk.NewDec(1000000),
					Vote:    voteNo,
				},
			},
		},
		{
			name: "one delegation with vote",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
			},
			votesByAddr: map[string]govtypes.Vote{
				valAccAddr1: voteNo,
				accAddr1:    voteYes,
			},
			expectedAccountVotes: []AccountVote{
				{
					Address: accAddr1,
					Power:   sdk.NewDec(1000),
					Vote:    voteYes,
				},
				{
					Address: valAccAddr1,
					Power:   sdk.NewDec(1000000 - 1000),
					Vote:    voteNo,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := getAccountVotes(tt.delegsByAddr, tt.votesByAddr, tt.valsByAddr)

			assert.Equal(t, tt.expectedAccountVotes, balances)
		})
	}
}

func createAccountAddrs(accNum int) []sdk.AccAddress {
	addrs := make([]sdk.AccAddress, accNum)
	for i := 0; i < accNum; i++ {
		pk := ed25519.GenPrivKey().PubKey()
		addrs[i] = sdk.AccAddress(pk.Address())
	}
	return addrs
}

func createValidatorAddrs(addrNum int) []sdk.ValAddress {
	addrs := make([]sdk.ValAddress, addrNum)
	for i := 0; i < addrNum; i++ {
		pk := ed25519.GenPrivKey().PubKey()
		addrs[i] = sdk.ValAddress(pk.Address())
	}
	return addrs
}
