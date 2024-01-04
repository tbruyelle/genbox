package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	h "github.com/dustin/go-humanize"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/olekukonko/tablewriter"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	proposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const ticker = "govno"

var unmarshaler jsonpb.Unmarshaler

func init() {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	govtypes.RegisterInterfaces(registry)
	sdk.RegisterInterfaces(registry)
	proposaltypes.RegisterInterfaces(registry)
	unmarshaler = jsonpb.Unmarshaler{AnyResolver: registry}
}

func main() {
	// Read data from files
	datapath := os.Args[1]
	votes, err := parseVotes(datapath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s votes\n", h.Comma(int64(len(votes))))
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

	// Tally from snapshot data
	results, totalVotingPower := tally(votes, valsByAddr, delegsByAddr)
	// Optionnaly print and compare tally with prop data
	printTallyResults(results, totalVotingPower, parseProp(datapath))

	var (
		// balances will receive the new token distribution
		balances []banktypes.Balance
		// balanceFactors maps vote option and airdrop/slash functions
		balanceFactors = map[govtypes.VoteOption]func(sdk.Dec) sdk.Dec{
			// XXX these are basic raw examples of airdrop/slash functions
			govtypes.OptionYes:        func(d sdk.Dec) sdk.Dec { return sdk.ZeroDec() },
			govtypes.OptionAbstain:    func(d sdk.Dec) sdk.Dec { return d.QuoInt64(2) },
			govtypes.OptionNo:         func(d sdk.Dec) sdk.Dec { return d },
			govtypes.OptionNoWithVeto: func(d sdk.Dec) sdk.Dec { return d.MulInt64(2) },
		}
	)
	_ = balanceFactors
	// TODO build balances:
	// Compute votesByAddr map, iterate over delegeations, if no vote found,
	// use validator vote to compute balance

	// Write bank genesis
	err = writeBankGenesis(balances)
	if err != nil {
		panic(err)
	}
}

func parseVotes(path string) (govtypes.Votes, error) {
	f, err := os.Open(filepath.Join(path, "votes.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// XXX workaround to unmarshal votes because proto doesn't support top-level array
	dec := json.NewDecoder(f)
	_, err = dec.Token()
	if err != nil {
		return nil, err
	}
	var votes govtypes.Votes
	for dec.More() {
		var vote govtypes.Vote
		err := unmarshaler.UnmarshalNext(dec, &vote)
		if err != nil {
			return nil, err
		}
		votes = append(votes, vote)
	}
	return votes, nil
}

func parseDelegationsByAddr(path string) (map[string][]stakingtypes.Delegation, error) {
	f, err := os.Open(filepath.Join(path, "delegations.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var delegs []stakingtypes.Delegation
	err = json.NewDecoder(f).Decode(&delegs)
	if err != nil {
		return nil, err
	}
	delegsByAddr := make(map[string][]stakingtypes.Delegation)
	for _, d := range delegs {
		delegsByAddr[d.DelegatorAddress] = append(delegsByAddr[d.DelegatorAddress], d)
	}
	return delegsByAddr, nil
}

func parseValidatorsByAddr(path string) (map[string]govtypes.ValidatorGovInfo, error) {
	f, err := os.Open(filepath.Join(path, "active_validators.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// XXX workaround to unmarshal validators because proto doesn't support top-level array
	dec := json.NewDecoder(f)
	_, err = dec.Token()
	if err != nil {
		return nil, err
	}
	valsByAddr := make(map[string]govtypes.ValidatorGovInfo)
	for dec.More() {
		var val stakingtypes.Validator
		err := unmarshaler.UnmarshalNext(dec, &val)
		if err != nil {
			return nil, err
		}
		valsByAddr[val.OperatorAddress] = govtypes.NewValidatorGovInfo(
			val.GetOperator(),
			val.GetBondedTokens(),
			val.GetDelegatorShares(),
			sdk.ZeroDec(),
			govtypes.WeightedVoteOptions{},
		)
	}
	return valsByAddr, nil
}

func parseProp(path string) govtypes.Proposal {
	f, err := os.Open(filepath.Join(path, "prop.json"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var prop govtypes.Proposal
	err = unmarshaler.Unmarshal(f, &prop)
	if err != nil {
		panic(err)
	}
	return prop
}

func tally(
	votes []govtypes.Vote, valsByAddr map[string]govtypes.ValidatorGovInfo,
	delegsByAddr map[string][]stakingtypes.Delegation,
) (map[govtypes.VoteOption]sdk.Dec, sdk.Dec) {
	var (
		results = map[govtypes.VoteOption]sdk.Dec{
			govtypes.OptionYes:        sdk.ZeroDec(),
			govtypes.OptionAbstain:    sdk.ZeroDec(),
			govtypes.OptionNo:         sdk.ZeroDec(),
			govtypes.OptionNoWithVeto: sdk.ZeroDec(),
		}
		totalVotingPower = sdk.ZeroDec()
	)
	for _, vote := range votes {
		// Check if it's a validator vote
		voter := sdk.MustAccAddressFromBech32(vote.Voter)
		valAddrStr := sdk.ValAddress(voter.Bytes()).String()
		if val, ok := valsByAddr[valAddrStr]; ok {
			// It's a validator vote
			val.Vote = vote.Options
			valsByAddr[valAddrStr] = val
		}

		// Check voter delegations
		dels := delegsByAddr[vote.Voter]
		// Initialize voter balance
		// balance := sdk.NewDec(0)
		for _, del := range dels {
			val, ok := valsByAddr[del.ValidatorAddress]
			if !ok {
				// Validator isn't in active set or jailed, ignore
				continue
			}
			// Reduce validator voting power with delegation that has voted
			val.DelegatorDeductions = val.DelegatorDeductions.Add(del.GetShares())
			valsByAddr[del.ValidatorAddress] = val

			// delegation shares * bonded / total shares
			votingPower := del.GetShares().MulInt(val.BondedTokens).Quo(val.DelegatorShares)
			// Iterate over vote options
			for _, option := range vote.Options {
				subPower := votingPower.Mul(option.Weight)
				results[option.Option] = results[option.Option].Add(subPower)
				// update balance according to vote
				// balance = balance.Add(balanceFactors[option.Option](subPower))
			}
			totalVotingPower = totalVotingPower.Add(votingPower)
		}
		// Append voter balance to bank genesis
		// balances = append(balances, banktypes.Balance{
		// Address: vote.Voter,
		// Coins:   sdk.NewCoins(sdk.NewCoin("u"+ticker, balance.TruncateInt())),
		// })
	}
	// iterate over the validators again to tally their voting power
	for _, val := range valsByAddr {
		if len(val.Vote) == 0 {
			continue
		}
		sharesAfterDeductions := val.DelegatorShares.Sub(val.DelegatorDeductions)
		votingPower := sharesAfterDeductions.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

		for _, option := range val.Vote {
			subPower := votingPower.Mul(option.Weight)
			results[option.Option] = results[option.Option].Add(subPower)
		}
		totalVotingPower = totalVotingPower.Add(votingPower)
	}
	return results, totalVotingPower
}

func printTallyResults(results map[govtypes.VoteOption]sdk.Dec, totalVotingPower sdk.Dec, prop govtypes.Proposal) {
	fmt.Println("Computed total voting power", h.Comma(totalVotingPower.TruncateInt64()))
	yesPercent := results[govtypes.OptionYes].
		Quo(totalVotingPower.Sub(results[govtypes.OptionAbstain]))
	fmt.Println("Yes percent:", yesPercent)
	tallyResult := govtypes.NewTallyResultFromMap(results)

	fmt.Println("--- TALLY RESULT ---")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", "Yes", "No", "NoWithVeto", "Abstain", "Total"})
	M := sdk.NewInt(1_000_000)
	appendTable := func(source string, t govtypes.TallyResult) {
		total := t.Yes.Add(t.No).Add(t.Abstain).Add(t.NoWithVeto)
		table.Append([]string{
			source,
			h.Comma(t.Yes.Quo(M).Int64()),
			h.Comma(t.No.Quo(M).Int64()),
			h.Comma(t.NoWithVeto.Quo(M).Int64()),
			h.Comma(t.Abstain.Quo(M).Int64()),
			h.Comma(total.Quo(M).Int64()),
		})
	}
	appendTable("computed", tallyResult)
	appendTable("from prop", prop.FinalTallyResult)
	diff := govtypes.NewTallyResult(
		tallyResult.Yes.Sub(prop.FinalTallyResult.Yes),
		tallyResult.Abstain.Sub(prop.FinalTallyResult.Abstain),
		tallyResult.No.Sub(prop.FinalTallyResult.No),
		tallyResult.NoWithVeto.Sub(prop.FinalTallyResult.NoWithVeto),
	)
	appendTable("diff", diff)
	table.Render()
}

func writeBankGenesis(balances []banktypes.Balance) error {
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
