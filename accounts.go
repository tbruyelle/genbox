package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Account struct {
	Address      string
	LiquidAmount sdk.Dec // TODO fill with bank balances
	StakedAmount sdk.Dec // TODO compare with sum of delegations?
	Vote         govtypes.Vote
	Delegations  []Delegation
}

type Delegation struct {
	Amount           sdk.Dec
	ValidatorAddress string
	Vote             govtypes.Vote
}

// getAccounts returns the list of all account with their vote and
// power, from direct or indirect votes.
func getAccounts(
	delegsByAddr map[string][]stakingtypes.Delegation,
	votesByAddr map[string]govtypes.Vote,
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
			// Deduct voter power from validator delegation power, because in the
			// following loop on validator we want to compute the validator self
			// delegation.
			val.DelegatorDeductions = val.DelegatorDeductions.Add(deleg.GetShares())
			valsByAddr[deleg.ValidatorAddress] = val

			// Compute delegation voting power
			delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
			account.StakedAmount = account.StakedAmount.Add(delegVotingPower)

			// Populate delegations with validator votes
			account.Delegations = append(account.Delegations, Delegation{
				ValidatorAddress: val.Address.String(),
				Amount:           delegVotingPower,
				Vote:             findValidatorVote(deleg.ValidatorAddress, votesByAddr),
			})
		}
		accounts = append(accounts, account)
	}
	// Add validator accounts
	for _, val := range valsByAddr {
		vote := findValidatorVote(val.Address.String(), votesByAddr)

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		// TODO add a AccAddress field in the struct used in valsByAddr?
		valAccAddr := sdk.AccAddress(val.Address.Bytes()) // TODO ensure this is a correct way to derive account address
		// TODO if validator has delegations, he's already in accounts: handle that!
		accounts = append(accounts, Account{
			Address:      valAccAddr.String(),
			LiquidAmount: sdk.ZeroDec(),
			StakedAmount: votingPower,
			Vote:         vote,
		})
	}
	return accounts
}

// TODO use a struct to hold xxxByAddr maps?
func findValidatorVote(valAddrStr string, votesByAddr map[string]govtypes.Vote) govtypes.Vote {
	// Convert validator address to account address to find vote
	valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
	if err != nil {
		panic(err)
	}
	valAccAddrStr := sdk.AccAddress(valAddr.Bytes()).String()
	return votesByAddr[valAccAddrStr]
}
