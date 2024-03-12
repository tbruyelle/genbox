package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func distribution(accounts []Account) error {
	// Some constants
	var (
		noMultiplier = sdk.NewDec(4)              // N & NWV get 1+x3
		bonus        = sdk.NewDecWithPrec(103, 2) // 3% bonus
		malus        = sdk.NewDecWithPrec(97, 2)  // -3% malus
	)
	_ = noMultiplier
	_ = bonus
	// Get amounts of Y, N and NWV
	var (
		amts     = newVoteMap()
		totalAmt = sdk.ZeroDec()
	)
	for i := range accounts {
		// init VotePercs
		acc := &accounts[i]
		acc.VotePercs = newVoteMap()
		if acc.StakedAmount.IsZero() {
			// No stake, consider non-voter
			acc.VotePercs[govtypes.OptionEmpty] = sdk.NewDec(1)
			continue
		}
		if len(acc.Vote) == 0 {
			// not a direct voter, check for delegated votes
			for _, del := range acc.Delegations {
				// Compute percentage of the delegation over the total staked amount
				delPerc := del.Amount.Quo(acc.StakedAmount)
				if len(del.Vote) == 0 {
					// user didn't vote and delegation didn't either, use the UNSPECIFIED
					// vote option to track it.
					acc.VotePercs[govtypes.OptionEmpty] = acc.VotePercs[govtypes.OptionEmpty].
						Add(sdk.NewDec(1).Mul(delPerc))
				} else {
					for _, vote := range del.Vote {
						acc.VotePercs[vote.Option] = acc.VotePercs[vote.Option].Add(vote.Weight.Mul(delPerc))

						amt := del.Amount.Mul(vote.Weight)
						amts[vote.Option] = amts[vote.Option].Add(amt)
						totalAmt = totalAmt.Add(amt)
					}
				}
			}
		} else {
			// direct voter
			for _, vote := range acc.Vote {
				acc.VotePercs[vote.Option] = vote.Weight

				amt := acc.StakedAmount.Mul(vote.Weight)
				amts[vote.Option] = amts[vote.Option].Add(amt)
				totalAmt = totalAmt.Add(amt)
			}
		}
	}
	// Compute percentage of Y, N and NWM amouts relative to totalAmt
	percs := make(map[govtypes.VoteOption]sdk.Dec)
	for k, v := range amts {
		percs[k] = v.Quo(totalAmt)
		fmt.Println(k, percs[k])
	}
	// Compute blend
	blend := percs[govtypes.OptionYes].
		Add(percs[govtypes.OptionNo].Mul(noMultiplier)).
		Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier))
	fmt.Println("BLEND", blend)

	totalAirdrop := sdk.ZeroDec()
	for i := range accounts {
		acc := &accounts[i]
		percs := acc.VotePercs
		stakingMultiplier := percs[govtypes.OptionYes].
			Add(percs[govtypes.OptionNo].Mul(noMultiplier)).
			Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier).Mul(bonus)).
			Add(percs[govtypes.OptionAbstain].Mul(blend)).
			Add(percs[govtypes.OptionEmpty].Mul(blend).Mul(malus))

		acc.AirdropAmount = acc.LiquidAmount.
			Add(acc.StakedAmount.Mul(stakingMultiplier))
		totalAirdrop = totalAirdrop.Add(acc.AirdropAmount)
	}
	fmt.Println("TOTAL SUPPLY ", totalAmt)
	fmt.Println("TOTAL AIRDROP", totalAirdrop)
	fmt.Println("RATIO", totalAirdrop.Quo(totalAmt))
	// output
	// address : airdropAmount

	return nil
}

func newVoteMap() map[govtypes.VoteOption]sdk.Dec {
	return map[govtypes.VoteOption]sdk.Dec{
		govtypes.OptionYes:        sdk.ZeroDec(),
		govtypes.OptionNo:         sdk.ZeroDec(),
		govtypes.OptionNoWithVeto: sdk.ZeroDec(),
		govtypes.OptionAbstain:    sdk.ZeroDec(),
		govtypes.OptionEmpty:      sdk.ZeroDec(),
	}
}
