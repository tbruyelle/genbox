package main

import (
	"encoding/json"
	"os"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func computeDistribution(
	delegsByAddr map[string][]stakingtypes.Delegation,
	votesByAddr map[string]govtypes.Vote,
	valsByAddr map[string]govtypes.ValidatorGovInfo,
	balanceFactors map[govtypes.VoteOption]func(sdk.Dec) sdk.Dec,
) []banktypes.Balance {
	// TODO write test and refac
	balances := []banktypes.Balance{}
	for addr, delegs := range delegsByAddr {
		// Did this address vote ?
		vote, ok := votesByAddr[addr]
		balance := sdk.ZeroDec()
		// TODO check if it's a validator (and validator can have delegation!)
		if ok {
			votingPower := sdk.ZeroDec()
			// Sum delegations voting power
			for _, deleg := range delegs {
				// Find validator
				val, ok := valsByAddr[deleg.ValidatorAddress]
				if !ok {
					// Validator isn't in active set or jailed, ignore
					continue
				}
				// Compute delegation voting power
				delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
				// Sum to voter voting power
				votingPower = votingPower.Add(delegVotingPower)
			}
			// Iterate over vote options
			for _, option := range vote.Options {
				subPower := votingPower.Mul(option.Weight)
				// update balance according to vote
				balance = balance.Add(balanceFactors[option.Option](subPower))
			}
		} else {
			// Didn't vote: check if validator has voted in delegations
			for _, deleg := range delegs {
				val, ok := valsByAddr[deleg.ValidatorAddress]
				if !ok {
					// Validator isn't in active set or jailed, ignore
					continue
				}
				// Convert validator address to account address to find vote
				// FIXME validator vote is already available in val.Vote
				valAddr, err := sdk.ValAddressFromBech32(deleg.ValidatorAddress)
				if err != nil {
					panic(err)
				}
				valAddrStr := sdk.AccAddress(valAddr.Bytes()).String()
				if vote, ok := votesByAddr[valAddrStr]; ok {
					// voter inherits validator vote
					delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
					// Iterate over vote options
					for _, option := range vote.Options {
						subPower := delegVotingPower.Mul(option.Weight)
						// update balance according to vote
						balance = balance.Add(balanceFactors[option.Option](subPower))
					}
				}
			}
			// FIXME if nobody voted (nor delegator nor validator), what should we do ? consider abstain?
			// Currently the delegator is completely slashed.
		}
		if !balance.IsZero() {
			// Append voter balance to bank genesis
			balances = append(balances, banktypes.Balance{
				Address: addr,
				Coins:   sdk.NewCoins(sdk.NewCoin("u"+ticker, balance.TruncateInt())),
			})
		}
	}
	return balances
}

func writeBankGenesis(balances []banktypes.Balance) error {
	g := banktypes.GenesisState{
		DenomMetadata: []banktypes.Metadata{
			{
				Display:     ticker,
				Symbol:      strings.ToUpper(ticker),
				Base:        "u" + ticker,
				Name:        "Atom One Govno",
				Description: "The governance token of Atom One Hub",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Aliases:  []string{"micro" + ticker},
						Denom:    "u" + ticker,
						Exponent: 0,
					},
					{
						Aliases:  []string{"milli" + ticker},
						Denom:    "m" + ticker,
						Exponent: 3,
					},
					{
						Aliases:  []string{ticker},
						Denom:    ticker,
						Exponent: 6,
					},
				},
			},
		},
		Balances: balances,
	}
	bz, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("bank.genesis", bz, 0o666)
}
