package main

import (
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// Some constants
var (
	// list of ICF wallets
	icfWallets = []string{
		// Source https://github.com/gnolang/bounties/issues/18#issuecomment-1034700230
		"cosmos1z8mzakma7vnaajysmtkwt4wgjqr2m84tzvyfkz",
		"cosmos1unc788q8md2jymsns24eyhua58palg5kc7cstv",
		// The 2 addresses above have been emptied in favour of the following 2
		"cosmos1sufkm72dw7ua9crpfhhp0dqpyuggtlhdse98e7",
		"cosmos1z6czaavlk6kjd48rpf58kqqw9ssad2uaxnazgl",
	}
	noMultiplier = sdk.NewDec(4)              // N & NWV get 1+x3
	bonus        = sdk.NewDecWithPrec(103, 2) // 3% bonus
	malus        = sdk.NewDecWithPrec(97, 2)  // -3% malus
)

func distribution(accounts []Account) (map[string]sdk.Dec, error) {
	// Get amounts of Y, N and NWV
	var (
		amts        = newVoteMap()
		totalAmt    = sdk.ZeroDec()
		totalSupply = sdk.ZeroDec()
	)
	for i := range accounts {
		// init VotePercs
		acc := &accounts[i]
		acc.VotePercs = newVoteMap()
		totalSupply = totalSupply.Add(acc.StakedAmount).Add(acc.LiquidAmount)
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
	res := make(map[string]sdk.Dec)
	for _, acc := range accounts {
		if slices.Contains(icfWallets, acc.Address) {
			// Slash ICF
			continue
		}
		percs := acc.VotePercs
		// stakingMultiplier details:
		// Yes:					x 1
		// No:         	x noMultiplier
		// NoWithVeto: 	x noMultiplier x bonus
		// Abstain:    	x blend
		// Didn't vote: x blend x malus
		stakingMultiplier := percs[govtypes.OptionYes].
			Add(percs[govtypes.OptionNo].Mul(noMultiplier)).
			Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier).Mul(bonus)).
			Add(percs[govtypes.OptionAbstain].Mul(blend)).
			Add(percs[govtypes.OptionEmpty].Mul(blend).Mul(malus))
		// Liquid amount gets the same multiplier as those who didn't vote.
		liquidMultiplier := blend.Mul(malus)

		airdrop := acc.LiquidAmount.Mul(liquidMultiplier).
			Add(acc.StakedAmount.Mul(stakingMultiplier))
		totalAirdrop = totalAirdrop.Add(airdrop)
		res[acc.Address] = airdrop
	}
	fmt.Println("TOTAL SUPPLY ", humand(totalSupply))
	fmt.Println("TOTAL AIRDROP", humand(totalAirdrop))
	fmt.Println("RATIO", totalAirdrop.Quo(totalSupply))
	// output
	// address : airdropAmount
	return res, nil
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
