package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TODO add liquid balance?
type AccountVote struct {
	Address      string
	TotalPower   sdk.Dec       // TODO use for check TODO Rename
	PoweredVotes []PoweredVote // TODO consider DirectVote and IndirectVotes field?
}

type PoweredVote struct {
	Power     sdk.Dec // TODO Rename to Delegation since it can be filled without votes
	Vote      govtypes.Vote
	Inherited bool
}

// getAccountVotes returns the list of all account with their vote and
// power, from direct or indirect votes.
func getAccountVotes(
	delegsByAddr map[string][]stakingtypes.Delegation,
	votesByAddr map[string]govtypes.Vote,
	valsByAddr map[string]govtypes.ValidatorGovInfo,
) []AccountVote {
	// TODO write test and refac
	accountVotes := []AccountVote{}
	for addr, delegs := range delegsByAddr {
		accountVote := AccountVote{
			Address:    addr,
			TotalPower: sdk.ZeroDec(),
		}
		// Did this address vote ?
		directVote, hasVoted := votesByAddr[addr]
		// TODO check if it's a validator (and validator can have delegation!)
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
			accountVote.TotalPower = accountVote.TotalPower.Add(delegVotingPower)

			if !hasVoted {
				// addr hasn't voted: inherit validator vote
				validatorVote := findValidatorVote(deleg.ValidatorAddress, votesByAddr)
				accountVote.PoweredVotes = append(accountVote.PoweredVotes, PoweredVote{
					Power:     delegVotingPower,
					Vote:      validatorVote, // if validator hasn't voted this will be empty
					Inherited: true,
				})
			}
		}
		if hasVoted {
			// Add the direct vote
			accountVote.PoweredVotes = append(accountVote.PoweredVotes, PoweredVote{
				Power:     accountVote.TotalPower,
				Vote:      directVote,
				Inherited: false,
			})
		}
		if !accountVote.TotalPower.IsZero() {
			accountVotes = append(accountVotes, accountVote)
		}
	}
	// Add validator accounts
	for _, val := range valsByAddr {
		vote := findValidatorVote(val.Address.String(), votesByAddr)

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		// TODO add a AccAddress field in the struct used in valsByAddr?
		valAccAddr := sdk.AccAddress(val.Address.Bytes()) // TODO ensure this is a correct way to derive account address
		accountVotes = append(accountVotes, AccountVote{
			Address:    valAccAddr.String(),
			TotalPower: votingPower,
			PoweredVotes: []PoweredVote{{
				Power: votingPower,
				Vote:  vote,
			}},
		})
	}
	return accountVotes
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
