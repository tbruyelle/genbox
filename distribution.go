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
	yesVotesMultiplier = sdk.NewDec(1)              // Y get x1
	noVotesMultiplier  = sdk.NewDec(4)              // N & NWV get 1+x3
	bonus              = sdk.NewDecWithPrec(103, 2) // 3% bonus
	malus              = sdk.NewDecWithPrec(97, 2)  // -3% malus
)

func distribution(accounts []Account) (map[string]sdk.Dec, sdk.Dec, error) {
	// Get amounts of Y, N and NWV
	var (
		amts                = newVoteMap()
		totalAmt            = sdk.ZeroDec()
		activeVotesTotalAmt = sdk.ZeroDec()
		totalSupply         = sdk.ZeroDec()
	)
	for i := range accounts {
		// init VotePercs
		acc := &accounts[i]
		acc.votePercs = newVoteMap()
		totalSupply = totalSupply.Add(acc.StakedAmount).Add(acc.LiquidAmount)
		if acc.StakedAmount.IsZero() {
			// No stake, consider non-voter
			acc.votePercs[govtypes.OptionEmpty] = sdk.NewDec(1)
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
					acc.votePercs[govtypes.OptionEmpty] = acc.votePercs[govtypes.OptionEmpty].Add(delPerc)
					amts[govtypes.OptionEmpty] = amts[govtypes.OptionEmpty].Add(del.Amount)
					totalAmt = totalAmt.Add(del.Amount)
				} else {
					for _, vote := range del.Vote {
						acc.votePercs[vote.Option] = acc.votePercs[vote.Option].Add(vote.Weight.Mul(delPerc))

						amt := del.Amount.Mul(vote.Weight)
						amts[vote.Option] = amts[vote.Option].Add(amt)
						totalAmt = totalAmt.Add(amt)
						if vote.Option != govtypes.OptionAbstain {
							activeVotesTotalAmt = activeVotesTotalAmt.Add(amt)
						}
					}
				}
			}
		} else {
			// direct voter
			for _, vote := range acc.Vote {
				acc.votePercs[vote.Option] = vote.Weight

				amt := acc.StakedAmount.Mul(vote.Weight)
				amts[vote.Option] = amts[vote.Option].Add(amt)
				totalAmt = totalAmt.Add(amt)
				if vote.Option != govtypes.OptionAbstain {
					activeVotesTotalAmt = activeVotesTotalAmt.Add(amt)
				}
			}
		}
	}
	// Compute percentage of Y, N and NWM amouts relative to activeVotesTotalAmt
	relativePercs := make(map[govtypes.VoteOption]sdk.Dec)
	for _, v := range []govtypes.VoteOption{
		govtypes.OptionYes,
		govtypes.OptionNo,
		govtypes.OptionNoWithVeto,
	} {
		relativePercs[v] = amts[v].Quo(activeVotesTotalAmt)
	}

	// Compute blend
	blend := relativePercs[govtypes.OptionYes].Mul(yesVotesMultiplier).
		Add(relativePercs[govtypes.OptionNo].Mul(noVotesMultiplier)).
		Add(relativePercs[govtypes.OptionNoWithVeto].Mul(noVotesMultiplier))

	totalAirdrop := sdk.ZeroDec()
	icfSlash := sdk.ZeroDec()
	res := make(map[string]sdk.Dec)
	airdropByVote := newVoteMap()
	for _, acc := range accounts {
		if slices.Contains(icfWallets, acc.Address) {
			// Slash ICF
			icfSlash = icfSlash.Add(acc.LiquidAmount).Add(acc.StakedAmount)
			continue
		}
		acctPercs := acc.votePercs
		// stakingMultiplier details:
		// Yes:		x yesVotesMultiplier
		// No:         	x noVotesMultiplier
		// NoWithVeto: 	x noVotesMultiplier x bonus
		// Abstain:    	x blend
		// Didn't vote: x blend x malus
		var (
			yesAirdropAmt        = acctPercs[govtypes.OptionYes].Mul(yesVotesMultiplier).Mul(acc.StakedAmount)
			noAirdropAmt         = acctPercs[govtypes.OptionNo].Mul(noVotesMultiplier).Mul(acc.StakedAmount)
			noWithVetoAirdropAmt = acctPercs[govtypes.OptionNoWithVeto].Mul(noVotesMultiplier).Mul(bonus).Mul(acc.StakedAmount)
			abstainAirdropAmt    = acctPercs[govtypes.OptionAbstain].Mul(blend).Mul(acc.StakedAmount)
			noVoteAirdropAmt     = acctPercs[govtypes.OptionEmpty].Mul(blend).Mul(malus).Mul(acc.StakedAmount)
		)
		airdropByVote[govtypes.OptionYes] = airdropByVote[govtypes.OptionYes].Add(yesAirdropAmt)
		airdropByVote[govtypes.OptionNo] = airdropByVote[govtypes.OptionNo].Add(noAirdropAmt)
		airdropByVote[govtypes.OptionNoWithVeto] = airdropByVote[govtypes.OptionNoWithVeto].Add(noWithVetoAirdropAmt)
		airdropByVote[govtypes.OptionAbstain] = airdropByVote[govtypes.OptionAbstain].Add(abstainAirdropAmt)
		airdropByVote[govtypes.OptionEmpty] = airdropByVote[govtypes.OptionEmpty].Add(noVoteAirdropAmt)

		// Liquid amount gets the same multiplier as those who didn't vote.
		liquidMultiplier := blend.Mul(malus)

		airdrop := acc.LiquidAmount.Mul(liquidMultiplier).
			Add(yesAirdropAmt).Add(noAirdropAmt).Add(noWithVetoAirdropAmt).
			Add(abstainAirdropAmt).Add(noVoteAirdropAmt)
		totalAirdrop = totalAirdrop.Add(airdrop)
		res[acc.Address] = airdrop

		// track also liquid amounts for the absolute percentages
		totalAmt = totalAmt.Add(acc.LiquidAmount)
	}

	fmt.Println("BLEND", blend)
	fmt.Println("TOTAL SUPPLY ", humand(totalSupply))
	fmt.Println("TOTAL AIRDROP", humand(totalAirdrop))
	fmt.Println("RATIO", totalAirdrop.Quo(totalSupply))
	fmt.Println("RELATIVE PERCS", relativePercs)
	fmt.Println("ICF SLASH", humand(icfSlash))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", "TOTAL", "DID NOT VOTE", "YES", "NO", "NOWITHVETO", "ABSTAIN", "NOT STAKED"})
	var (
		totalDidntVoteAirdrop  = airdropByVote[govtypes.OptionEmpty]
		totalYesAirdrop        = airdropByVote[govtypes.OptionYes]
		totalNoAirdrop         = airdropByVote[govtypes.OptionNo]
		totalNoWithVetoAirdrop = airdropByVote[govtypes.OptionNoWithVeto]
		totalAbstainAirdrop    = airdropByVote[govtypes.OptionAbstain]
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
