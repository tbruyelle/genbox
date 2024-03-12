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
		amts = map[govtypes.VoteOption]sdk.Dec{
			govtypes.OptionYes:        sdk.ZeroDec(),
			govtypes.OptionNo:         sdk.ZeroDec(),
			govtypes.OptionNoWithVeto: sdk.ZeroDec(),
			govtypes.OptionAbstain:    sdk.ZeroDec(),
		}
		totalAmt = sdk.ZeroDec()
	)
	for _, acc := range accounts {
		if len(acc.Vote) == 0 {
			// not a direct voter, check for delegated votes
			for _, del := range acc.Delegations {
				for _, vote := range del.Vote {
					v, ok := amts[vote.Option]
					if ok {
						amt := del.Amount.Mul(vote.Weight)
						amts[vote.Option] = v.Add(amt)
						totalAmt = totalAmt.Add(amt)
					}
				}
			}
			continue
		}
		// direct voter
		for _, vote := range acc.Vote {
			v, ok := amts[vote.Option]
			if ok {
				amt := acc.StakedAmount.Mul(vote.Weight)
				amts[vote.Option] = v.Add(amt)
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

	for _, acc := range accounts {
		percs := acc.VotePercs
		stakingMultiplier := percs[govtypes.OptionYes].
			Add(percs[govtypes.OptionNo].Mul(noMultiplier)).
			Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier).Mul(bonus)).
			Add(percs[govtypes.OptionAbstain].Mul(blend)).
			Add(percs[govtypes.OptionEmpty].Mul(blend).Mul(malus))

		acc.AirdropAmount = acc.LiquidAmount.
			Add(acc.StakedAmount.Mul(stakingMultiplier))
	}
	// output
	// address : airdropAmount

	return nil
}
