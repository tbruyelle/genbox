package main

import (
	"encoding/json"
	"os"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func writeBankGenesis(accountVotes []AccountVote) error {
	var balances []banktypes.Balance
	for _, a := range accountVotes {
		balance := sdk.ZeroDec()
		for _, option := range a.Vote.Options {
			subPower := a.Power.Mul(option.Weight)
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
