package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func applyVoteOptions(vote govtypes.WeightedVoteOptions, amount sdk.Dec) sdk.Dec {
	balance := sdk.ZeroDec()
	for _, option := range vote {
		subPower := amount.Mul(option.Weight)
		// TODO apply bonus or slash function according to option
		switch option.Option {
		case govtypes.OptionYes:
			// ??
		case govtypes.OptionNo:
			// ??
		case govtypes.OptionAbstain:
			// ??
		case govtypes.OptionNoWithVeto:
			// ??
		}
		balance = balance.Add(subPower)
	}
	return balance
}

// TODO add tests
func writeBankGenesis() error {
	accountsFile := os.Args[2]
	f, err := os.Open(accountsFile)
	if err != nil {
		return fmt.Errorf("cannot read %s file, run `%s accounts` to generate it: %w", accountsFile, os.Args[0], err)
	}
	defer f.Close()
	var accounts []Account
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		return fmt.Errorf("cannot json decode accounts from file %s: %w", accountsFile, err)
	}

	const ticker = "govno"
	var balances []banktypes.Balance
	for _, a := range accounts {
		balance := sdk.ZeroDec()
		if len(a.Vote) > 0 {
			// Direct vote
			balance = applyVoteOptions(a.Vote, a.StakedAmount)
		} else {
			// Inherited votes
			for _, deleg := range a.Delegations {
				balance = balance.Add(applyVoteOptions(deleg.Vote, deleg.Amount))
			}
		}
		balances = append(balances, banktypes.Balance{
			Address: a.Address,
			Coins:   sdk.NewCoins(sdk.NewInt64Coin("u"+ticker, balance.TruncateInt64())),
		})
	}
	g := banktypes.GenesisState{
		DenomMetadata: []banktypes.Metadata{
			{
				Display:     ticker,
				Symbol:      strings.ToUpper(ticker),
				Base:        "u" + ticker,
				Name:        "Atom One Govno",
				Description: "The governance token of Atom One Hub",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Aliases:  []string{"micro" + ticker},
						Denom:    "u" + ticker,
						Exponent: 0,
					},
					{
						Aliases:  []string{"milli" + ticker},
						Denom:    "m" + ticker,
						Exponent: 3,
					},
					{
						Aliases:  []string{ticker},
						Denom:    ticker,
						Exponent: 6,
					},
				},
			},
		},
		Balances: balances,
	}
	bz, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("bank.genesis", bz, 0o666)
}
