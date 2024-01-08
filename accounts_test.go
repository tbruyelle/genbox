package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestGetAccounts(t *testing.T) {
	var (
		accAddrs = createAccountAddrs(2)
		accAddr1 = accAddrs[0].String()
		// voterAddr2 = accAddrs[1]
		valAddrs = createValidatorAddrs(2)
		// Validator1
		valAddr1       = valAddrs[0]
		valAddr1Str    = valAddr1.String()
		valAccAddr1Str = sdk.AccAddress(valAddrs[0]).String()
		// Validator2
		valAddr2       = valAddrs[1]
		valAddr2Str    = valAddr2.String()
		valAccAddr2Str = sdk.AccAddress(valAddrs[1]).String()
		newVal         = func(addr sdk.ValAddress, bonded, shares int64, vote govtypes.WeightedVoteOptions) govtypes.ValidatorGovInfo {
			return govtypes.ValidatorGovInfo{
				Address:             addr,
				BondedTokens:        sdk.NewInt(bonded),
				DelegatorShares:     sdk.NewDec(shares),
				DelegatorDeductions: sdk.ZeroDec(),
				Vote:                vote,
			}
		}
		// Some votes
		noVote  govtypes.WeightedVoteOptions
		voteYes = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionYes,
			Weight: sdk.NewDec(1),
		}}
		voteNo = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionNo,
			Weight: sdk.NewDec(1),
		}}
		voteAbstain = govtypes.WeightedVoteOptions{{
			Option: govtypes.OptionAbstain,
			Weight: sdk.NewDec(1),
		}}
	)
	tests := []struct {
		name             string
		delegsByAddr     map[string][]stakingtypes.Delegation
		votesByAddr      map[string]govtypes.WeightedVoteOptions
		valsByAddr       map[string]govtypes.ValidatorGovInfo
		expectedAccounts []Account
	}{
		{
			name:             "no delegation",
			expectedAccounts: []Account{},
		},
		{
			name: "one delegation: inactive validator",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.ZeroDec(),
				},
			},
		},
		{
			name: "one delegation: nobody voted",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1Str: newVal(valAddr1, 1000000, 1000000, noVote),
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1Str,
						Amount:           sdk.NewDec(1000),
					}},
				},
				{
					Address:      valAccAddr1Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
				},
			},
		},
		{
			name: "one delegation: inherit validator vote",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1Str: newVal(valAddr1, 1000000, 1000000, voteNo),
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1Str,
						Amount:           sdk.NewDec(1000),
						Vote:             voteNo,
					}},
				},
				{
					Address:      valAccAddr1Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
			},
		},
		{
			name: "one delegation: voted",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1Str: newVal(valAddr1, 1000000, 1000000, voteNo),
			},
			votesByAddr: map[string]govtypes.WeightedVoteOptions{
				accAddr1: voteYes,
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Vote:         voteYes,
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1Str,
						Amount:           sdk.NewDec(1000),
						Vote:             voteNo,
					}},
				},
				{
					Address:      valAccAddr1Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
			},
		},
		{
			name: "multiple delegations: inherit validator votes",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr2Str,
						Shares:           sdk.NewDec(2000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1Str: newVal(valAddr1, 1000000, 1000000, voteNo),
				valAddr2Str: newVal(valAddr2, 2000000, 2000000, voteYes),
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(3000),
					Delegations: []Delegation{
						{
							ValidatorAddress: valAddr1Str,
							Amount:           sdk.NewDec(1000),
							Vote:             voteNo,
						},
						{
							ValidatorAddress: valAddr2Str,
							Amount:           sdk.NewDec(2000),
							Vote:             voteYes,
						},
					},
				},
				{
					Address:      valAccAddr1Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
				{
					Address:      valAccAddr2Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(2000000 - 2000),
					Vote:         voteYes,
				},
			},
		},
		{
			name: "multiple delegations: voted",
			delegsByAddr: map[string][]stakingtypes.Delegation{
				accAddr1: {
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr1Str,
						Shares:           sdk.NewDec(1000),
					},
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr2Str,
						Shares:           sdk.NewDec(2000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1Str: newVal(valAddr1, 1000000, 1000000, voteNo),
				valAddr2Str: newVal(valAddr2, 2000000, 2000000, voteYes),
			},
			votesByAddr: map[string]govtypes.WeightedVoteOptions{
				accAddr1: voteAbstain,
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(3000),
					Vote:         voteAbstain,
					Delegations: []Delegation{
						{
							ValidatorAddress: valAddr1Str,
							Amount:           sdk.NewDec(1000),
							Vote:             voteNo,
						},
						{
							ValidatorAddress: valAddr2Str,
							Amount:           sdk.NewDec(2000),
							Vote:             voteYes,
						},
					},
				},
				{
					Address:      valAccAddr1Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
				{
					Address:      valAccAddr2Str,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(2000000 - 2000),
					Vote:         voteYes,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accounts := getAccounts(tt.delegsByAddr, tt.votesByAddr, tt.valsByAddr)

			assert.Equal(t, tt.expectedAccounts, accounts)
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
