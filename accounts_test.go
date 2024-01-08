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
		valAddr1    = valAddrs[0].String()
		valAccAddr1 = sdk.AccAddress(valAddrs[0]).String()
		val1        = govtypes.ValidatorGovInfo{
			Address:             valAddrs[0],
			BondedTokens:        sdk.NewInt(1000000),
			DelegatorShares:     sdk.NewDec(1000000),
			DelegatorDeductions: sdk.ZeroDec(),
		}
		// Validator2
		valAddr2    = valAddrs[1].String()
		valAccAddr2 = sdk.AccAddress(valAddrs[1]).String()
		val2        = govtypes.ValidatorGovInfo{
			Address:             valAddrs[1],
			BondedTokens:        sdk.NewInt(2000000),
			DelegatorShares:     sdk.NewDec(2000000),
			DelegatorDeductions: sdk.ZeroDec(),
		}
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
		voteAbstain = govtypes.Vote{
			Options: []govtypes.WeightedVoteOption{{
				Option: govtypes.OptionAbstain,
				Weight: sdk.NewDec(1),
			}},
		}
	)
	tests := []struct {
		name             string
		delegsByAddr     map[string][]stakingtypes.Delegation
		votesByAddr      map[string]govtypes.Vote
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
						ValidatorAddress: valAddr1,
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
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1,
						Amount:           sdk.NewDec(1000),
					}},
				},
				{
					Address:      valAccAddr1,
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
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1,
						Amount:           sdk.NewDec(1000),
						Vote:             voteNo,
					}},
				},
				{
					Address:      valAccAddr1,
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
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000),
					Vote:         voteYes,
					Delegations: []Delegation{{
						ValidatorAddress: valAddr1,
						Amount:           sdk.NewDec(1000),
						Vote:             voteNo,
					}},
				},
				{
					Address:      valAccAddr1,
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
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr2,
						Shares:           sdk.NewDec(2000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
				valAddr2: val2,
			},
			votesByAddr: map[string]govtypes.Vote{
				valAccAddr1: voteNo,
				valAccAddr2: voteYes,
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(3000),
					Delegations: []Delegation{
						{
							ValidatorAddress: valAddr1,
							Amount:           sdk.NewDec(1000),
							Vote:             voteNo,
						},
						{
							ValidatorAddress: valAddr2,
							Amount:           sdk.NewDec(2000),
							Vote:             voteYes,
						},
					},
				},
				{
					Address:      valAccAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
				{
					Address:      valAccAddr2,
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
						ValidatorAddress: valAddr1,
						Shares:           sdk.NewDec(1000),
					},
					{
						DelegatorAddress: accAddr1,
						ValidatorAddress: valAddr2,
						Shares:           sdk.NewDec(2000),
					},
				},
			},
			valsByAddr: map[string]govtypes.ValidatorGovInfo{
				valAddr1: val1,
				valAddr2: val2,
			},
			votesByAddr: map[string]govtypes.Vote{
				accAddr1:    voteAbstain,
				valAccAddr1: voteNo,
				valAccAddr2: voteYes,
			},
			expectedAccounts: []Account{
				{
					Address:      accAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(3000),
					Vote:         voteAbstain,
					Delegations: []Delegation{
						{
							ValidatorAddress: valAddr1,
							Amount:           sdk.NewDec(1000),
							Vote:             voteNo,
						},
						{
							ValidatorAddress: valAddr2,
							Amount:           sdk.NewDec(2000),
							Vote:             voteYes,
						},
					},
				},
				{
					Address:      valAccAddr1,
					LiquidAmount: sdk.ZeroDec(),
					StakedAmount: sdk.NewDec(1000000 - 1000),
					Vote:         voteNo,
				},
				{
					Address:      valAccAddr2,
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
