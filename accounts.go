package main

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Account struct {
	Address      string
	Type         string
	LiquidAmount sdk.Dec
	StakedAmount sdk.Dec
	Vote         govtypes.WeightedVoteOptions
	Delegations  []Delegation
}

type Delegation struct {
	Amount           sdk.Dec
	ValidatorAddress string
	Vote             govtypes.WeightedVoteOptions
}

// voteWeights returns a consolidated map of votes, merging direct and indirect
// votes with their respective weight summed.
// The map also uses the govtypes.OptionEmpty to hold the no-vote weight.
func (a Account) voteWeights() voteMap {
	v := newVoteMap()
	if a.StakedAmount.IsZero() {
		v[govtypes.OptionEmpty] = sdk.OneDec()
		return v
	}
	if len(a.Vote) == 0 {
		// not a direct voter, check for delegated votes
		for _, del := range a.Delegations {
			// Compute percentage of the delegation over the total staked amount
			delPerc := del.Amount.Quo(a.StakedAmount)
			if len(del.Vote) == 0 {
				// user didn't vote and delegation didn't either, use the UNSPECIFIED
				// vote option to track it.
				v.add(govtypes.OptionEmpty, delPerc)
			} else {
				for _, vote := range del.Vote {
					v.add(vote.Option, vote.Weight.Mul(delPerc))
				}
			}
		}
		return v
	}
	// direct voter
	for _, vote := range a.Vote {
		v[vote.Option] = vote.Weight
	}
	return v
}

func (a Account) String() string {
	bz, err := json.MarshalIndent(a, "", " ")
	if err != nil {
		panic(err)
	}
	return string(bz)
}

// getAccounts returns the list of all account with their vote and
// power, from direct or indirect votes.
func getAccounts(
	delegsByAddr map[string][]stakingtypes.Delegation,
	votesByAddr map[string]govtypes.WeightedVoteOptions,
	valsByAddr map[string]govtypes.ValidatorGovInfo,
	balancesByAddr map[string]sdk.Coin,
	accountTypesPerAddr map[string]string,
) []Account {
	accountsByAddr := make(map[string]Account, len(delegsByAddr))
	// Feed delegations
	for addr, delegs := range delegsByAddr {
		accType := accountTypesPerAddr[addr]
		if accType == "/cosmos.auth.v1beta1.ModuleAccount" ||
			accType == "/ibc.applications.interchain_accounts.v1.InterchainAccount" {
			// Ignore ModuleAccount & InterchainAccount
			continue
		}
		account := Account{
			Address:      addr,
			Type:         accType,
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
		accountsByAddr[addr] = account
	}
	// Feed balances
	for addr, balance := range balancesByAddr {
		acc, ok := accountsByAddr[addr]
		if ok {
			acc.LiquidAmount = balance.Amount.ToDec()
			accountsByAddr[addr] = acc
		} else {
			accType := accountTypesPerAddr[addr]
			if accType == "/cosmos.auth.v1beta1.ModuleAccount" ||
				accType == "/ibc.applications.interchain_accounts.v1.InterchainAccount" {
				// Ignore ModuleAccount & InterchainAccount
				continue
			}
			accountsByAddr[addr] = Account{
				Address:      addr,
				Type:         accType,
				LiquidAmount: balance.Amount.ToDec(),
				StakedAmount: sdk.ZeroDec(),
			}
		}
	}
	// Map to slice
	var accounts []Account
	for _, a := range accountsByAddr {
		accounts = append(accounts, a)
	}
	return accounts
}
