package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func distribution(accounts []Account) error {
	// Some constants
	var (
		noMultiplier = sdk.NewDec(3)              // N & NWV get x3
		bonus        = sdk.NewDecWithPrec(103, 2) // 3% bonus
	)
	// Get amounts of Y, N and NWV
	var (
		amts = map[govtypes.VoteOption]sdk.Dec{
			govtypes.OptionYes:        sdk.ZeroDec(),
			govtypes.OptionNo:         sdk.ZeroDec(),
			govtypes.OptionNoWithVeto: sdk.ZeroDec(),
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
		Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier).Mul(bonus))
	fmt.Println("BLEND", blend)

	return nil
}
