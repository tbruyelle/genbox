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
	yesVotesMultiplier = sdk.OneDec()               // Y get x1
	noVotesMultiplier  = sdk.NewDec(4)              // N & NWV get 1+x3
	bonus              = sdk.NewDecWithPrec(103, 2) // 3% bonus
	malus              = sdk.NewDecWithPrec(97, 2)  // -3% malus
)

type airdrop struct {
	// addresses contains the airdrop amount per address.
	addresses map[string]sdk.Int
	// nonVotersMultiplier ensures that non-voters don't hold more than 1/3 of
	// the supply
	nonVotersMultiplier sdk.Dec
	// $ATOM distribution
	atom distrib
	// $ATONE distribution
	atone distrib
	// Amount of $ATOM slashed for the ICF
	icfSlash sdk.Dec
}

type distrib struct {
	// total supply of the distrib
	supply sdk.Dec
	// votes holds the part of the distrib per vote.
	votes voteMap
	// unstaked is part of the distrib for unstaked amounts.
	unstaked sdk.Dec
}

func distribution(accounts []Account) (airdrop, error) {
	airdrop := airdrop{
		addresses: make(map[string]sdk.Int),
		icfSlash:  sdk.ZeroDec(),
		atom: distrib{
			supply:   sdk.ZeroDec(),
			votes:    newVoteMap(),
			unstaked: sdk.ZeroDec(),
		},
		atone: distrib{
			supply:   sdk.ZeroDec(),
			votes:    newVoteMap(),
			unstaked: sdk.ZeroDec(),
		},
	}
	for _, acc := range accounts {
		var (
			voteWeights = acc.voteWeights()

			// Detail of vote distribution for $ATOM
			yesAtomAmt        = voteWeights[govtypes.OptionYes].Mul(acc.StakedAmount)
			noAtomAmt         = voteWeights[govtypes.OptionNo].Mul(acc.StakedAmount)
			noWithVetoAtomAmt = voteWeights[govtypes.OptionNoWithVeto].Mul(acc.StakedAmount)
			abstainAtomAmt    = voteWeights[govtypes.OptionAbstain].Mul(acc.StakedAmount)
			noVoteAtomAmt     = voteWeights[govtypes.OptionEmpty].Mul(acc.StakedAmount)
		)
		// increment $ATOM votes
		airdrop.atom.votes.add(govtypes.OptionYes, yesAtomAmt)
		airdrop.atom.votes.add(govtypes.OptionNo, noAtomAmt)
		airdrop.atom.votes.add(govtypes.OptionNoWithVeto, noWithVetoAtomAmt)
		airdrop.atom.votes.add(govtypes.OptionAbstain, abstainAtomAmt)
		airdrop.atom.votes.add(govtypes.OptionEmpty, noVoteAtomAmt)
		// increment $ATOM supply
		airdrop.atom.supply = airdrop.atom.supply.Add(acc.StakedAmount.Add(acc.LiquidAmount))
		airdrop.atom.unstaked = airdrop.atom.unstaked.Add(acc.LiquidAmount)

	}

	// Compute nonVotersMultiplier to have non-voters <= 33%
	var (
		yesAtoneTotalAmt     = airdrop.atom.votes[govtypes.OptionYes].Mul(yesVotesMultiplier)
		noAtoneTotalAmt      = airdrop.atom.votes[govtypes.OptionNo].Add(airdrop.atom.votes[govtypes.OptionNoWithVeto]).Mul(noVotesMultiplier)
		noVotersAtomTotalAmt = airdrop.atom.votes[govtypes.OptionAbstain].Add(airdrop.atom.votes[govtypes.OptionEmpty]).Add(airdrop.atom.unstaked)
		targetNonVotersPerc  = sdk.NewDecWithPrec(33, 2)
	)
	// Formula is:
	// nonVotersMultiplier = (t x (yesAtone + noAtone)) / ((1 - t) x nonVoterAtom)
	// where t is the targetNonVotersPerc
	airdrop.nonVotersMultiplier = targetNonVotersPerc.Mul(yesAtoneTotalAmt.Add(noAtoneTotalAmt)).
		Quo((sdk.OneDec().Sub(targetNonVotersPerc)).Mul(noVotersAtomTotalAmt))

	for _, acc := range accounts {
		if slices.Contains(icfWallets, acc.Address) {
			// Slash ICF
			airdrop.icfSlash = airdrop.icfSlash.Add(acc.LiquidAmount).Add(acc.StakedAmount)
			continue
		}

		var (
			voteWeights       = acc.voteWeights()
			yesAtomAmt        = voteWeights[govtypes.OptionYes].Mul(acc.StakedAmount)
			noAtomAmt         = voteWeights[govtypes.OptionNo].Mul(acc.StakedAmount)
			noWithVetoAtomAmt = voteWeights[govtypes.OptionNoWithVeto].Mul(acc.StakedAmount)
			abstainAtomAmt    = voteWeights[govtypes.OptionAbstain].Mul(acc.StakedAmount)
			noVoteAtomAmt     = voteWeights[govtypes.OptionEmpty].Mul(acc.StakedAmount)
			// Apply airdrop multipliers:
			// Yes:         x yesVotesMultiplier
			// No:         	x noVotesMultiplier
			// NoWithVeto: 	x noVotesMultiplier x bonus
			// Abstain:    	x nonVotersMultiplier
			// Didn't vote: x nonVotersMultiplier x malus
			yesAirdropAmt        = yesAtomAmt.Mul(yesVotesMultiplier)
			noAirdropAmt         = noAtomAmt.Mul(noVotesMultiplier)
			noWithVetoAirdropAmt = noWithVetoAtomAmt.Mul(noVotesMultiplier).Mul(bonus)
			abstainAirdropAmt    = abstainAtomAmt.Mul(airdrop.nonVotersMultiplier)
			noVoteAirdropAmt     = noVoteAtomAmt.Mul(airdrop.nonVotersMultiplier).Mul(malus)

			// Liquid amount gets the same multiplier as those who didn't vote.
			liquidMultiplier = airdrop.nonVotersMultiplier.Mul(malus)

			// total airdrop for this account
			liquidAirdrop = acc.LiquidAmount.Mul(liquidMultiplier)
			stakedAirdrop = yesAirdropAmt.Add(noAirdropAmt).Add(noWithVetoAirdropAmt).
					Add(abstainAirdropAmt).Add(noVoteAirdropAmt)
			airdropAmt = liquidAirdrop.Add(stakedAirdrop)
		)
		// increment airdrop votes
		airdrop.atone.votes.add(govtypes.OptionYes, yesAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionNo, noAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionNoWithVeto, noWithVetoAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionAbstain, abstainAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionEmpty, noVoteAirdropAmt)
		// increment airdrop supply
		airdrop.atone.supply = airdrop.atone.supply.Add(airdropAmt)
		airdrop.atone.unstaked = airdrop.atone.unstaked.Add(liquidAirdrop)
		// add address and amount
		airdrop.addresses[acc.Address] = airdropAmt.TruncateInt()
	}
	return airdrop, nil
}

// convenient type for manipulating vote counts.
type voteMap map[govtypes.VoteOption]sdk.Dec

var (
	allVoteOptions = []govtypes.VoteOption{
		govtypes.OptionEmpty,
		govtypes.OptionYes,
		govtypes.OptionAbstain,
		govtypes.OptionNo,
		govtypes.OptionNoWithVeto,
	}
	activeVoteOptions = []govtypes.VoteOption{
		govtypes.OptionYes,
		govtypes.OptionNo,
		govtypes.OptionNoWithVeto,
	}
)

func newVoteMap() voteMap {
	m := make(map[govtypes.VoteOption]sdk.Dec)
	for _, v := range allVoteOptions {
		m[v] = sdk.ZeroDec()
	}
	return m
}

func (m voteMap) add(v govtypes.VoteOption, d sdk.Dec) {
	m[v] = m[v].Add(d)
}

func (m voteMap) total() sdk.Dec {
	d := sdk.ZeroDec()
	for _, v := range m {
		d = d.Add(v)
	}
	return d
}

func (m voteMap) toPercentages() map[govtypes.VoteOption]sdk.Dec {
	total := m.total()
	percs := make(map[govtypes.VoteOption]sdk.Dec)
	for k, v := range m {
		percs[k] = v.Quo(total)
	}
	return percs
}

func printAirdropStats(airdrop airdrop) {
	printDistrib := func(d distrib) {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetHeader([]string{"", "TOTAL", "DID NOT VOTE", "YES", "NO", "NOWITHVETO", "ABSTAIN", "NOT STAKED"})
		table.Append([]string{
			"Distributed",
			humand(d.supply),
			humand(d.votes[govtypes.OptionEmpty]),
			humand(d.votes[govtypes.OptionYes]),
			humand(d.votes[govtypes.OptionNo]),
			humand(d.votes[govtypes.OptionNoWithVeto]),
			humand(d.votes[govtypes.OptionAbstain]),
			humand(d.unstaked),
		})
		table.Append([]string{
			"Percentage over total",
			"",
			humanPercent(d.votes[govtypes.OptionEmpty].Quo(d.supply)),
			humanPercent(d.votes[govtypes.OptionYes].Quo(d.supply)),
			humanPercent(d.votes[govtypes.OptionNo].Quo(d.supply)),
			humanPercent(d.votes[govtypes.OptionNoWithVeto].Quo(d.supply)),
			humanPercent(d.votes[govtypes.OptionAbstain].Quo(d.supply)),
			humanPercent(d.unstaked.Quo(d.supply)),
		})
		table.Render()
	}
	fmt.Println("$ATOM distribution")
	printDistrib(airdrop.atom)
	fmt.Println()
	fmt.Printf("$ATONE distribution (ratio: x%.3f, nonVotersMultiplier: %.3f, icfSlash: %s $ATOM)\n",
		airdrop.atone.supply.Quo(airdrop.atom.supply).MustFloat64(),
		airdrop.nonVotersMultiplier.MustFloat64(),
		humand(airdrop.icfSlash),
	)
	printDistrib(airdrop.atone)
}
