package main

import (
	"fmt"
	"os"

	h "github.com/dustin/go-humanize"
)

const ticker = "govno"

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "tally" && os.Args[1] != "genesis") {
		fmt.Fprintf(os.Stderr, "Usage:\n%s [tally|genesis] [datapath]\n", os.Args[0])
		os.Exit(1)
	}
	//-----------------------------------------
	// Read data from files

	var (
		command  = os.Args[1]
		datapath = os.Args[2]
	)
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

	switch command {
	case "tally":
		results, totalVotingPower := tally(votesByAddr, valsByAddr, delegsByAddr)
		// Optionnaly print and compare tally with prop data
		printTallyResults(results, totalVotingPower, parseProp(datapath))

	case "genesis":
		accountVotes := getAccountVotes(delegsByAddr, votesByAddr, valsByAddr)
		_ = accountVotes

		// Write bank genesis
		err = writeBankGenesis(accountVotes)
		if err != nil {
			panic(err)
		}
	}
}
