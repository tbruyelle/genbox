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
		airdropFile     = filepath.Join(datapath, "airdrop.json")
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

	case "autostaking":
		err := autoStaking(filepath.Join(datapath, "genesis.json"))
		if err != nil {
			panic(err)
		}

	case "distribution":
		accounts, err := parseAccounts(accountsFile)
		if err != nil {
			panic(err)
		}
		airdrop, err := distribution(accounts)
		if err != nil {
			panic(err)
		}
		bz, err := json.MarshalIndent(airdrop.addresses, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(airdropFile, bz, 0o666); err != nil {
			panic(err)
		}
		printAirdropStats(airdrop)

	case "tally":
		votesByAddr, err := parseVotesByAddr(datapath)
		if err != nil {
			panic(err)
		}
		valsByAddr, err := parseValidatorsByAddr(datapath, votesByAddr)
		if err != nil {
			panic(err)
		}
		delegsByAddr, err := parseDelegationsByAddr(datapath)
		if err != nil {
			panic(err)
		}
		results, totalVotingPower := tally(votesByAddr, valsByAddr, delegsByAddr)
		printTallyResults(results, totalVotingPower, parseProp(datapath))

	case "accounts":
		votesByAddr, err := parseVotesByAddr(datapath)
		if err != nil {
			panic(err)
		}
		valsByAddr, err := parseValidatorsByAddr(datapath, votesByAddr)
		if err != nil {
			panic(err)
		}
		delegsByAddr, err := parseDelegationsByAddr(datapath)
		if err != nil {
			panic(err)
		}
		balancesByAddr, err := parseBalancesByAddr(datapath, "uatom")
		if err != nil {
			panic(err)
		}
		accountTypesByAddr, err := parseAccountTypesPerAddr(datapath)
		if err != nil {
			panic(err)
		}

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

const M = 1_000_000 // 1 million

func human(i sdk.Int) string {
	M := sdk.NewInt(M)
	return h.Comma(i.Quo(M).Int64())
}

func humani(i int64) string {
	return h.Comma(i / M)
}

func humand(d sdk.Dec) string {
	M := sdk.NewDec(1_000_000)
	return h.Comma(d.Quo(M).RoundInt64())
}

func humanPercent(d sdk.Dec) string {
	return fmt.Sprintf("%d%%", d.Mul(sdk.NewDec(100)).RoundInt64())
}
