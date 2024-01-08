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
			expectedAccountVotes: []AccountVote{},
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
			expectedAccountVotes: []AccountVote{
				{
					Address:    accAddr1,
					TotalPower: sdk.NewDec(1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000),
						Inherited: true,
					}},
				},
				{
					Address:    valAccAddr1,
					TotalPower: sdk.NewDec(1000000 - 1000),
					PoweredVotes: []PoweredVote{{
						Power: sdk.NewDec(1000000 - 1000),
					}},
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
			expectedAccountVotes: []AccountVote{
				{
					Address:    accAddr1,
					TotalPower: sdk.NewDec(1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000),
						Vote:      voteNo,
						Inherited: true,
					}},
				},
				{
					Address:    valAccAddr1,
					TotalPower: sdk.NewDec(1000000 - 1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000000 - 1000),
						Vote:      voteNo,
						Inherited: false,
					}},
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
			expectedAccountVotes: []AccountVote{
				{
					Address:    accAddr1,
					TotalPower: sdk.NewDec(1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000),
						Vote:      voteYes,
						Inherited: false,
					}},
				},
				{
					Address:    valAccAddr1,
					TotalPower: sdk.NewDec(1000000 - 1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000000 - 1000),
						Vote:      voteNo,
						Inherited: false,
					}},
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
			expectedAccountVotes: []AccountVote{
				{
					Address:    accAddr1,
					TotalPower: sdk.NewDec(3000),
					PoweredVotes: []PoweredVote{
						{
							Power:     sdk.NewDec(1000),
							Vote:      voteNo,
							Inherited: true,
						},
						{
							Power:     sdk.NewDec(2000),
							Vote:      voteYes,
							Inherited: true,
						},
					},
				},
				{
					Address:    valAccAddr1,
					TotalPower: sdk.NewDec(1000000 - 1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000000 - 1000),
						Vote:      voteNo,
						Inherited: false,
					}},
				},
				{
					Address:    valAccAddr2,
					TotalPower: sdk.NewDec(2000000 - 2000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(2000000 - 2000),
						Vote:      voteYes,
						Inherited: false,
					}},
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
				accAddr1:    voteNo,
				valAccAddr1: voteNo,
				valAccAddr2: voteYes,
			},
			expectedAccountVotes: []AccountVote{
				{
					Address:    accAddr1,
					TotalPower: sdk.NewDec(3000),
					PoweredVotes: []PoweredVote{
						{
							Power:     sdk.NewDec(3000),
							Vote:      voteNo,
							Inherited: false,
						},
					},
				},
				{
					Address:    valAccAddr1,
					TotalPower: sdk.NewDec(1000000 - 1000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(1000000 - 1000),
						Vote:      voteNo,
						Inherited: false,
					}},
				},
				{
					Address:    valAccAddr2,
					TotalPower: sdk.NewDec(2000000 - 2000),
					PoweredVotes: []PoweredVote{{
						Power:     sdk.NewDec(2000000 - 2000),
						Vote:      voteYes,
						Inherited: false,
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balances := getAccountVotes(tt.delegsByAddr, tt.votesByAddr, tt.valsByAddr)

			assert.ElementsMatch(t, tt.expectedAccountVotes, balances)
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
