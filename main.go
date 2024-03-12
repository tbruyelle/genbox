package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	h "github.com/dustin/go-humanize"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func human(i sdk.Int) string {
	M := sdk.NewInt(1_000_000)
	return h.Comma(i.Quo(M).Int64())
}

func humani(i int64) string {
	return h.Comma(i / 1_000_000)
}

var commands = []string{"tally", "accounts", "genesis", "autostaking", "distribution"}

func main() {
	if len(os.Args) != 3 || !slices.Contains(commands, os.Args[1]) {
		fmt.Fprintf(os.Stderr, "Usage:\n%s [%s] [datapath]\n",
			filepath.Base(os.Args[0]), strings.Join(commands, "|"))
		os.Exit(1)
	}

	var (
		command         = os.Args[1]
		datapath        = os.Args[2]
		accountsFile    = filepath.Join(datapath, "accounts.json")
		bankGenesisFile = filepath.Join(datapath, "bank.genesis")
	)

	switch command {
	case "genesis":
		accounts, err := parseAccounts(accountsFile)
		if err != nil {
			panic(err)
		}
		if err := writeBankGenesis(accounts, bankGenesisFile); err != nil {
			panic(err)
		}
		fmt.Printf("%s file created.\n", bankGenesisFile)
		os.Exit(0)
	case "autostaking":
		err := autoStaking(filepath.Join(datapath, "genesis.json"))
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	case "distribution":
		accounts, err := parseAccounts(accountsFile)
		if err != nil {
			panic(err)
		}
		res, err := distribution(accounts)
		if err != nil {
			panic(err)
		}
		airdropFile := filepath.Join(datapath, "airdrop.json")
		f, err := os.Open(airdropFile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(res); err != nil {
			panic(err)
		}
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
