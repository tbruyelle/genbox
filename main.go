package main

import (
	"fmt"
	"os"

	h "github.com/dustin/go-humanize"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const ticker = "govno"

func main() {
	//-----------------------------------------
	// Read data from files

	datapath := os.Args[1]
	votesByAddr, err := parseVotesByAddr(datapath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s votes\n", h.Comma(int64(len(votesByAddr))))
	valsByAddr, err := parseValidatorsByAddr(datapath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d validators\n", len(valsByAddr))
	delegsByAddr, err := parseDelegationsByAddr(datapath)
	if err != nil {
		panic(err)
	}
	var numDeleg int
	for _, d := range delegsByAddr {
		numDeleg += len(d)
	}
	fmt.Printf("%s delegations for %s delegators\n", h.Comma(int64(numDeleg)),
		h.Comma(int64(len(delegsByAddr))))

	//-----------------------------------------
	// Tally from data

	results, totalVotingPower := tally(votesByAddr, valsByAddr, delegsByAddr)
	// Optionnaly print and compare tally with prop data
	printTallyResults(results, totalVotingPower, parseProp(datapath))

	//-----------------------------------------
	// Compute balances

	// balanceFactors maps vote option and airdrop/slash functions
	balanceFactors := map[govtypes.VoteOption]func(sdk.Dec) sdk.Dec{
		// XXX these are basic raw examples of airdrop/slash functions
		govtypes.OptionYes:        func(d sdk.Dec) sdk.Dec { return sdk.ZeroDec() },
		govtypes.OptionAbstain:    func(d sdk.Dec) sdk.Dec { return d.QuoInt64(2) },
		govtypes.OptionNo:         func(d sdk.Dec) sdk.Dec { return d },
		govtypes.OptionNoWithVeto: func(d sdk.Dec) sdk.Dec { return d.MulInt64(2) },
	}
	balances := computeBalances(delegsByAddr, votesByAddr, valsByAddr, balanceFactors)

	// Write bank genesis
	err = writeBankGenesis(balances)
	if err != nil {
		panic(err)
	}
}
