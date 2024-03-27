package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/browser"

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
)

type airdrop struct {
	// params hold the distribution parameters that resulted in this airdrop
	params distriParams
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

type distriParams struct {
	yesVotesMultiplier sdk.Dec
	noVotesMultiplier  sdk.Dec
	bonus              sdk.Dec
	malus              sdk.Dec
	supplyFactor       sdk.Dec
}

func (d distriParams) String() string {
	return fmt.Sprintf("Yes x%.1f / No x%.1f",
		d.yesVotesMultiplier.MustFloat64(), d.noVotesMultiplier.MustFloat64())
}

func defaultDistriParams() distriParams {
	return distriParams{
		yesVotesMultiplier: sdk.OneDec(),               // Y get x1
		noVotesMultiplier:  sdk.NewDec(4),              // N & NWV get 1+x3
		bonus:              sdk.NewDecWithPrec(103, 2), // 3% bonus
		malus:              sdk.NewDecWithPrec(97, 2),  // -3% malus
		supplyFactor:       sdk.NewDecWithPrec(1, 1),   // Decrease final supply by a factor of 10
	}
}

func (d distrib) votePercentages() map[govtypes.VoteOption]sdk.Dec {
	percs := make(map[govtypes.VoteOption]sdk.Dec)
	for k, v := range d.votes {
		percs[k] = v.Quo(d.supply)
	}
	return percs
}

func distribution(accounts []Account, params distriParams) (airdrop, error) {
	airdrop := airdrop{
		params:    params,
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
		yesAtoneTotalAmt     = airdrop.atom.votes[govtypes.OptionYes].Mul(params.yesVotesMultiplier)
		noAtoneTotalAmt      = airdrop.atom.votes[govtypes.OptionNo].Add(airdrop.atom.votes[govtypes.OptionNoWithVeto]).Mul(params.noVotesMultiplier)
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
			yesAirdropAmt        = yesAtomAmt.Mul(params.yesVotesMultiplier).Mul(params.supplyFactor)
			noAirdropAmt         = noAtomAmt.Mul(params.noVotesMultiplier).Mul(params.supplyFactor)
			noWithVetoAirdropAmt = noWithVetoAtomAmt.Mul(params.noVotesMultiplier).Mul(params.bonus).Mul(params.supplyFactor)
			abstainAirdropAmt    = abstainAtomAmt.Mul(airdrop.nonVotersMultiplier).Mul(params.supplyFactor)
			noVoteAirdropAmt     = noVoteAtomAmt.Mul(airdrop.nonVotersMultiplier).Mul(params.malus).Mul(params.supplyFactor)

			// Liquid amount gets the same multiplier as those who didn't vote.
			liquidMultiplier = airdrop.nonVotersMultiplier.Mul(params.malus)

			// total airdrop for this account
			liquidAirdropAmt = acc.LiquidAmount.Mul(liquidMultiplier).Mul(params.supplyFactor)
			stakedAirdropAmt = yesAirdropAmt.Add(noAirdropAmt).Add(noWithVetoAirdropAmt).
						Add(abstainAirdropAmt).Add(noVoteAirdropAmt)
			airdropAmt = liquidAirdropAmt.Add(stakedAirdropAmt)
		)
		// increment airdrop votes
		airdrop.atone.votes.add(govtypes.OptionYes, yesAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionNo, noAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionNoWithVeto, noWithVetoAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionAbstain, abstainAirdropAmt)
		airdrop.atone.votes.add(govtypes.OptionEmpty, noVoteAirdropAmt)
		// increment airdrop supply
		airdrop.atone.supply = airdrop.atone.supply.Add(airdropAmt)
		airdrop.atone.unstaked = airdrop.atone.unstaked.Add(liquidAirdropAmt)
		// add address and amount (skipping 0 balance)
		if amtInt := airdropAmt.RoundInt(); !amtInt.IsZero() {
			airdrop.addresses[acc.Address] = amtInt
		}
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

func printAirdropsStats(chartMode bool, airdrops []airdrop) error {
	if chartMode {
		f, err := os.CreateTemp("", "chart*.html")
		if err != nil {
			return err
		}
		defer f.Close()
		renderBarChart(f, airdrops)
		renderPieChart(f, "$ATOM distribution", airdrops[0].atom)
		for _, airdrop := range airdrops {
			renderPieChart(f, fmt.Sprintf("$ATONE distribution %s", airdrop.params), airdrop.atone)
		}
		fmt.Printf("Charts rendered in %s\n", f.Name())
		browser.OpenFile(f.Name())
		return nil
	}

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
		votePercs := d.votePercentages()
		table.Append([]string{
			"Percentage over total",
			"",
			humanPercent(votePercs[govtypes.OptionEmpty]),
			humanPercent(votePercs[govtypes.OptionYes]),
			humanPercent(votePercs[govtypes.OptionNo]),
			humanPercent(votePercs[govtypes.OptionNoWithVeto]),
			humanPercent(votePercs[govtypes.OptionAbstain]),
			humanPercent(d.unstaked.Quo(d.supply)),
		})
		table.Render()
		fmt.Println()
	}
	fmt.Println("$ATOM distribution")
	printDistrib(airdrops[0].atom)
	for _, airdrop := range airdrops {
		fmt.Printf("$ATONE distribution (params: %s) (ratio: x%.3f, nonVotersMultiplier: %.3f, icfSlash: %s $ATOM)\n",
			airdrop.params,
			airdrop.atone.supply.Quo(airdrop.atom.supply).MustFloat64(),
			airdrop.nonVotersMultiplier.MustFloat64(),
			humand(airdrop.icfSlash),
		)
		printDistrib(airdrop.atone)
	}
	return nil
}

func renderBarChart(f *os.File, airdrops []airdrop) error {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Votes distribution"}),
		charts.WithLegendOpts(opts.Legend{Show: true, Right: "right", Orient: "vertical"}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Formatter: opts.FuncOpts("function(params){ return params.value.toFixed(2)+'%'}"),
		}),
	)

	bar.SetXAxis([]string{"Yes", "No", "NWV", "Abstain", "DNV", "Unstaked"})
	generateData := func(d distrib) []opts.BarData {
		var (
			votePercs  = d.votePercentages()
			data       = make([]opts.BarData, 6)
			oneHundred = sdk.NewDec(100)
		)
		data[0] = opts.BarData{
			Name:  "Yes",
			Value: votePercs[govtypes.OptionYes].Mul(oneHundred).MustFloat64(),
		}
		data[1] = opts.BarData{
			Name:  "No",
			Value: votePercs[govtypes.OptionNo].Mul(oneHundred).MustFloat64(),
		}
		data[2] = opts.BarData{
			Name:  "NWV",
			Value: votePercs[govtypes.OptionNoWithVeto].Mul(oneHundred).MustFloat64(),
		}
		data[3] = opts.BarData{
			Name:  "Abstain",
			Value: votePercs[govtypes.OptionAbstain].Mul(oneHundred).MustFloat64(),
		}
		data[4] = opts.BarData{
			Name:  "DNV",
			Value: votePercs[govtypes.OptionEmpty].Mul(oneHundred).MustFloat64(),
		}
		data[5] = opts.BarData{
			Name:  "Unstaked",
			Value: d.unstaked.Quo(d.supply).Mul(oneHundred).MustFloat64(),
		}
		return data
	}
	bar.AddSeries("$ATOM", generateData(airdrops[0].atom))
	for _, airdrop := range airdrops {
		bar.AddSeries(fmt.Sprintf("$ATONE %s", airdrop.params), generateData(airdrop.atone))
	}
	return bar.Render(f)
}

func renderPieChart(f *os.File, title string, d distrib) error {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: false,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:      true,
			Formatter: opts.FuncOpts("function(params){ return params.name+': '+params.value.toFixed(2)+'%'}"),
		}),
	)
	var (
		data       = make([]opts.PieData, 6)
		votePercs  = d.votePercentages()
		oneHundred = sdk.NewDec(100)
	)
	data[0] = opts.PieData{
		Name:      "Yes",
		ItemStyle: &opts.ItemStyle{Color: "#ff6f69"},
		Value:     votePercs[govtypes.OptionYes].Mul(oneHundred).MustFloat64(),
	}
	data[1] = opts.PieData{
		Name:      "No",
		ItemStyle: &opts.ItemStyle{Color: "#96ceb4"},
		Value:     votePercs[govtypes.OptionNo].Mul(oneHundred).MustFloat64(),
	}
	data[2] = opts.PieData{
		Name:      "NWV",
		ItemStyle: &opts.ItemStyle{Color: "#87b9a2"},
		Value:     votePercs[govtypes.OptionNoWithVeto].Mul(oneHundred).MustFloat64(),
	}
	data[3] = opts.PieData{
		Name:      "Abstain",
		ItemStyle: &opts.ItemStyle{Color: "#ffcc5c"},
		Value:     votePercs[govtypes.OptionAbstain].Mul(oneHundred).MustFloat64(),
	}
	data[4] = opts.PieData{
		Name:      "DNV",
		ItemStyle: &opts.ItemStyle{Color: "#ffeead"},
		Value:     votePercs[govtypes.OptionEmpty].Mul(oneHundred).MustFloat64(),
	}
	data[5] = opts.PieData{
		Name:      "Unstaked",
		ItemStyle: &opts.ItemStyle{Color: "#fff8de"},
		Value:     d.unstaked.Quo(d.supply).Mul(oneHundred).MustFloat64(),
	}
	pie.AddSeries("pie", data).
		SetSeriesOptions(charts.WithLabelOpts(opts.Label{
			Show:      true,
			Formatter: opts.FuncOpts("function(params){ return params.name+': '+params.value.toFixed(2)+'%'}"),
		}))
	return pie.Render(f)
}
