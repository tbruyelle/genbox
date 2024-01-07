package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type AccountVote struct {
	Address string
	Power   sdk.Dec
	Vote    govtypes.Vote
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
		// Did this address vote ?
		vote, ok := votesByAddr[addr]
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
				// Deduct voter power from validator delegation power
				val.DelegatorDeductions = val.DelegatorDeductions.Add(deleg.GetShares())
				valsByAddr[deleg.ValidatorAddress] = val

				// Compute delegation voting power
				delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
				// Sum to voter voting power
				votingPower = votingPower.Add(delegVotingPower)
			}
			accountVotes = append(accountVotes, AccountVote{
				Address: addr,
				Power:   votingPower,
				Vote:    vote,
			})
		} else {
			// Didn't vote: check if validator has voted in delegations
			for _, deleg := range delegs {
				val, ok := valsByAddr[deleg.ValidatorAddress]
				if !ok {
					// Validator isn't in active set or jailed, ignore
					continue
				}
				vote := findValidatorVote(deleg.ValidatorAddress, votesByAddr)
				if len(vote.Options) > 0 {
					// voter inherits validator vote
					delegVotingPower := deleg.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
					accountVotes = append(accountVotes, AccountVote{
						Address: addr,
						Power:   delegVotingPower,
						Vote:    vote,
					})
				}
				// FIXME if nobody voted (nor delegator nor validator), what should we do ? consider abstain?
				// Currently the delegator is completely slashed.
			}
		}
	}
	// Loop on validators' vote
	for _, val := range valsByAddr {
		vote := findValidatorVote(val.Address.String(), votesByAddr)
		if len(vote.Options) == 0 {
			continue
		}

		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		// TODO add a AccAddress field in the struct used in valsByAddr?
		valAccAddr := sdk.AccAddress(val.Address.Bytes())
		accountVotes = append(accountVotes, AccountVote{
			Address: valAccAddr.String(),
			Power:   votingPower,
			Vote:    vote,
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
