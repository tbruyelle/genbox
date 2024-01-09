package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Account struct {
	Address      string
	LiquidAmount sdk.Dec // TODO fill with bank balances
	StakedAmount sdk.Dec
	Vote         govtypes.WeightedVoteOptions
	Delegations  []Delegation
}

type Delegation struct {
	Amount           sdk.Dec
	ValidatorAddress string
	Vote             govtypes.WeightedVoteOptions
}

// getAccounts returns the list of all account with their vote and
// power, from direct or indirect votes.
func getAccounts(
	delegsByAddr map[string][]stakingtypes.Delegation,
	votesByAddr map[string]govtypes.WeightedVoteOptions,
	valsByAddr map[string]govtypes.ValidatorGovInfo,
) []Account {
	// TODO write test and refac
	accounts := []Account{}
	for addr, delegs := range delegsByAddr {
		account := Account{
			Address:      addr,
			LiquidAmount: sdk.ZeroDec(),
			StakedAmount: sdk.ZeroDec(),
			Vote:         votesByAddr[addr],
		}
		for _, deleg := range delegs {
			// Find validator
			val, ok := valsByAddr[deleg.ValidatorAddress]
			if !ok {
				// Validator isn't in active set or jailed, ignore
				continue
			}

			// Compute delegation voting power
			delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
			account.StakedAmount = account.StakedAmount.Add(delegVotingPower)

			// Populate delegations with validator votes
			account.Delegations = append(account.Delegations, Delegation{
				ValidatorAddress: val.Address.String(),
				Amount:           delegVotingPower,
				Vote:             val.Vote,
			})
		}
		accounts = append(accounts, account)
	}

	// TODO check for sanity, remove later
	for _, a := range accounts {
		staked := sdk.ZeroDec()
		for _, d := range a.Delegations {
			staked = staked.Add(d.Amount)
		}
		if !staked.Equal(a.StakedAmount) {
			panic(fmt.Sprintf("NOPE %v %v", staked, a.StakedAmount))
		}
	}
	return accounts
}
