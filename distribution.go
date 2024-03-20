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
	// blend is the neutral multiplier, for which the $ATOM is neither rewarded
	// nor diluted.
	blend sdk.Dec
	// supply of the airdrop.
	supply sdk.Dec
	// votes holds the part of the airdrop per vote.
	votes voteMap
	// unstaked is part of the airdrop for unstaked amounts.
	unstaked sdk.Dec
}

func distribution(accounts []Account) (airdrop, error) {
	var (
		blend    = computeBlend(accounts)
		icfSlash = sdk.ZeroDec()
		airdrop  = airdrop{
			addresses: make(map[string]sdk.Int),
			blend:     blend,
			supply:    sdk.ZeroDec(),
			votes:     newVoteMap(),
			unstaked:  sdk.ZeroDec(),
		}
		atomSupply = sdk.ZeroDec()
	)
	for _, acc := range accounts {
		atomSupply = atomSupply.Add(acc.StakedAmount).Add(acc.LiquidAmount)
		if slices.Contains(icfWallets, acc.Address) {
			// Slash ICF
			icfSlash = icfSlash.Add(acc.LiquidAmount).Add(acc.StakedAmount)
			continue
		}
		var (
			// stakingMultiplier details:
			// Yes:         x yesVotesMultiplier
			// No:         	x noVotesMultiplier
			// NoWithVeto: 	x noVotesMultiplier x bonus
			// Abstain:    	x blend
			// Didn't vote: x blend x malus
			voteWeights          = acc.voteWeights()
			yesAirdropAmt        = voteWeights[govtypes.OptionYes].Mul(yesVotesMultiplier).Mul(acc.StakedAmount)
			noAirdropAmt         = voteWeights[govtypes.OptionNo].Mul(noVotesMultiplier).Mul(acc.StakedAmount)
			noWithVetoAirdropAmt = voteWeights[govtypes.OptionNoWithVeto].Mul(noVotesMultiplier).Mul(bonus).Mul(acc.StakedAmount)
			abstainAirdropAmt    = voteWeights[govtypes.OptionAbstain].Mul(blend).Mul(acc.StakedAmount)
			noVoteAirdropAmt     = voteWeights[govtypes.OptionEmpty].Mul(blend).Mul(malus).Mul(acc.StakedAmount)

			// Liquid amount gets the same multiplier as those who didn't vote.
			liquidMultiplier = blend.Mul(malus)

			// total airdrop for this account
			liquidAirdrop = acc.LiquidAmount.Mul(liquidMultiplier)
			stakedAirdrop = yesAirdropAmt.Add(noAirdropAmt).Add(noWithVetoAirdropAmt).
					Add(abstainAirdropAmt).Add(noVoteAirdropAmt)
			airdropAmt = liquidAirdrop.Add(stakedAirdrop)
		)
		// increment airdrop votes
		airdrop.votes.add(govtypes.OptionYes, yesAirdropAmt)
		airdrop.votes.add(govtypes.OptionNo, noAirdropAmt)
		airdrop.votes.add(govtypes.OptionNoWithVeto, noWithVetoAirdropAmt)
		airdrop.votes.add(govtypes.OptionAbstain, abstainAirdropAmt)
		airdrop.votes.add(govtypes.OptionEmpty, noVoteAirdropAmt)
		// increment airdrop supply
		airdrop.supply = airdrop.supply.Add(airdropAmt)
		airdrop.unstaked = airdrop.unstaked.Add(liquidAirdrop)
		// add address and amount
		airdrop.addresses[acc.Address] = airdropAmt.TruncateInt()
	}

	fmt.Println("BLEND", blend)
	fmt.Println("ATOM  SUPPLY", humand(atomSupply))
	fmt.Println("ATONE SUPPLY", humand(airdrop.supply))
	fmt.Println("RATIO", airdrop.supply.Quo(atomSupply))
	fmt.Println("ICF SLASH", humand(icfSlash))

	return airdrop, nil
}

func computeBlend(accounts []Account) sdk.Dec {
	activeVoteAmts := newVoteMap()
	for _, acc := range accounts {
		for voteOpt, weight := range acc.voteWeights() {
			switch voteOpt {
			case govtypes.OptionYes, govtypes.OptionNo, govtypes.OptionNoWithVeto:
				activeVoteAmts.add(voteOpt, acc.StakedAmount.Mul(weight))
			}
		}
	}
	// Compute percentage of Y, N and NWM amouts relative to activeVoteAmts
	activePercs := activeVoteAmts.toPercentages()

	// Compute blend
	blend := activePercs[govtypes.OptionYes].Mul(yesVotesMultiplier).
		Add(activePercs[govtypes.OptionNo].Mul(noVotesMultiplier)).
		Add(activePercs[govtypes.OptionNoWithVeto].Mul(noVotesMultiplier))
	return blend
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

func printAirdropStats(a airdrop) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", "TOTAL", "DID NOT VOTE", "YES", "NO", "NOWITHVETO", "ABSTAIN", "NOT STAKED"})
	table.Append([]string{
		"Distributed $ATONE",
		humand(a.supply),
		humand(a.votes[govtypes.OptionEmpty]),
		humand(a.votes[govtypes.OptionYes]),
		humand(a.votes[govtypes.OptionNo]),
		humand(a.votes[govtypes.OptionNoWithVeto]),
		humand(a.votes[govtypes.OptionAbstain]),
		humand(a.unstaked),
	})
	table.Append([]string{
		"Percentage over total",
		"",
		humanPercent(a.votes[govtypes.OptionEmpty].Quo(a.supply)),
		humanPercent(a.votes[govtypes.OptionYes].Quo(a.supply)),
		humanPercent(a.votes[govtypes.OptionNo].Quo(a.supply)),
		humanPercent(a.votes[govtypes.OptionNoWithVeto].Quo(a.supply)),
		humanPercent(a.votes[govtypes.OptionAbstain].Quo(a.supply)),
		humanPercent(a.unstaked.Quo(a.supply)),
	})
	table.Render()
}
