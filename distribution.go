package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/olekukonko/tablewriter"

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
	yesMultiplier = sdk.NewDec(1)              // Y get x1
	noMultiplier  = sdk.NewDec(4)              // N & NWV get 1+x3
	bonus         = sdk.NewDecWithPrec(103, 2) // 3% bonus
	malus         = sdk.NewDecWithPrec(97, 2)  // -3% malus
)

func distribution(accounts []Account) (map[string]sdk.Dec, sdk.Dec, error) {
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
					acc.VotePercs[govtypes.OptionEmpty] = acc.VotePercs[govtypes.OptionEmpty].Add(delPerc)
					amts[govtypes.OptionEmpty] = amts[govtypes.OptionEmpty].Add(del.Amount)
					totalAmt = totalAmt.Add(del.Amount)
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
	}
	// Compute blend
	blend := percs[govtypes.OptionYes].
		Add(percs[govtypes.OptionNo].Mul(noMultiplier)).
		Add(percs[govtypes.OptionNoWithVeto].Mul(noMultiplier))

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
	fmt.Println("BLEND", blend)
	fmt.Println("TOTAL SUPPLY ", humand(totalSupply))
	fmt.Println("TOTAL AIRDROP", humand(totalAirdrop))
	fmt.Println("RATIO", totalAirdrop.Quo(totalSupply))
	fmt.Println("PERCS", percs)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", "TOTAL", "DID NOT VOTE", "YES", "NO", "NOWITHVETO", "ABSTAIN", "NOT STAKED"})
	var (
		totalDidntVoteAirdrop  = totalAirdrop.Mul(percs[govtypes.OptionEmpty])
		totalYesAirdrop        = totalAirdrop.Mul(percs[govtypes.OptionYes])
		totalNoAirdrop         = totalAirdrop.Mul(percs[govtypes.OptionNo])
		totalNoWithVetoAirdrop = totalAirdrop.Mul(percs[govtypes.OptionNoWithVeto])
		totalAbstainAirdrop    = totalAirdrop.Mul(percs[govtypes.OptionAbstain])
		totalStakedAirdrop     = totalDidntVoteAirdrop.Add(totalYesAirdrop).
					Add(totalNoAirdrop).Add(totalNoWithVetoAirdrop).Add(totalAbstainAirdrop)
		totalUnstakedAirdrop = totalAirdrop.Sub(totalStakedAirdrop)
	)
	table.Append([]string{
		"Distributed $ATONE",
		humand(totalAirdrop),
		humand(totalDidntVoteAirdrop),
		humand(totalYesAirdrop),
		humand(totalNoAirdrop),
		humand(totalNoWithVetoAirdrop),
		humand(totalAbstainAirdrop),
		humand(totalUnstakedAirdrop),
	})
	table.Append([]string{
		"Percentage over total",
		"",
		humanPercent(totalDidntVoteAirdrop.Quo(totalAirdrop)),
		humanPercent(totalYesAirdrop.Quo(totalAirdrop)),
		humanPercent(totalNoAirdrop.Quo(totalAirdrop)),
		humanPercent(totalNoWithVetoAirdrop.Quo(totalAirdrop)),
		humanPercent(totalAbstainAirdrop.Quo(totalAirdrop)),
		humanPercent(totalUnstakedAirdrop.Quo(totalAirdrop)),
	})
	table.Render()
	// output
	// address : airdropAmount
	return res, blend, nil
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
