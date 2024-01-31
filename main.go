package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	h "github.com/dustin/go-humanize"
)

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "tally" && os.Args[1] != "accounts" && os.Args[1] != "genesis") {
		fmt.Fprintf(os.Stderr, "Usage:\n%s [tally|accounts|genesis] [datapath]\n", os.Args[0])
		os.Exit(1)
	}

	var (
		command         = os.Args[1]
		datapath        = os.Args[2]
		accountsFile    = filepath.Join(datapath, "accounts.json")
		bankGenesisFile = filepath.Join(datapath, "bank.genesis")
	)

	if command == "genesis" {
		if err := writeBankGenesis(accountsFile, bankGenesisFile); err != nil {
			panic(err)
		}
		fmt.Printf("%s file created.\n", bankGenesisFile)
		os.Exit(0)
	}

	//-----------------------------------------
	// Read data from files

	votesByAddr, err := parseVotesByAddr(datapath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s votes\n", h.Comma(int64(len(votesByAddr))))
	valsByAddr, err := parseValidatorsByAddr(datapath, votesByAddr)
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
	balancesByAddr, err := parseBalancesByAddr(datapath, "uatom")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s account balances\n", h.Comma(int64(len(balancesByAddr))))

	switch command {
	case "tally":
		results, totalVotingPower := tally(votesByAddr, valsByAddr, delegsByAddr)
		// Optionnaly print and compare tally with prop data
		printTallyResults(results, totalVotingPower, parseProp(datapath))

	case "accounts":
		accountTypesByAddr, err := parseAccountTypesPerAddr(datapath)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s accounts\n", h.Comma(int64(len(accountTypesByAddr))))

		accounts := getAccounts(delegsByAddr, votesByAddr, valsByAddr, balancesByAddr, accountTypesByAddr)

		bz, err := json.MarshalIndent(accounts, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(accountsFile, bz, 0o666); err != nil {
			panic(err)
		}
		fmt.Printf("%s file created.\n", accountsFile)
	}
}
