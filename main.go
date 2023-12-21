package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/gogo/protobuf/jsonpb"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var unmarshaler jsonpb.Unmarshaler

func init() {
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	unmarshaler = jsonpb.Unmarshaler{AnyResolver: registry}
}

func main() {
	// Read data from files
	datapath := os.Args[1]
	votes, err := parseVotes(datapath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d votes\n", len(votes))
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
	fmt.Printf("%d delegations for %d delegators\n", numDeleg, len(delegsByAddr))

	// Tally votes
	results := make(map[govtypes.VoteOption]sdk.Dec)
	results[govtypes.OptionYes] = sdk.ZeroDec()
	results[govtypes.OptionAbstain] = sdk.ZeroDec()
	results[govtypes.OptionNo] = sdk.ZeroDec()
	results[govtypes.OptionNoWithVeto] = sdk.ZeroDec()
	totalVotingPower := sdk.ZeroDec()
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
		for _, del := range dels {
			val, ok := valsByAddr[del.ValidatorAddress]
			if !ok {
				// Validator isn't in active set or jailed, ignore
				continue
			}
			// Reduce validator voting power with delegation that has voted
			val.DelegatorDeductions = val.DelegatorDeductions.Add(del.Shares)
			valsByAddr[del.ValidatorAddress] = val

			// delegation shares * bonded / total shares
			votingPower := del.Shares.MulInt(val.BondedTokens).Quo(val.DelegatorShares)

			for _, option := range vote.Options {
				subPower := votingPower.Mul(option.Weight)
				results[option.Option] = results[option.Option].Add(subPower)
			}
			totalVotingPower = totalVotingPower.Add(votingPower)
		}
	}
	// iterate over the validators again to tally their voting power
	nonvoter := 0
	for _, val := range valsByAddr {
		if len(val.Vote) == 0 {
			nonvoter++
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
	fmt.Println("VALIDATOR DIDN'T VOTE", nonvoter)
	tallyResults := govtypes.NewTallyResultFromMap(results)

	fmt.Println("VOTING POWER", humanize.Comma(totalVotingPower.TruncateInt64()))
	fmt.Println("YES", humanize.Comma(tallyResults.Yes.Int64()))
	fmt.Println("NO", humanize.Comma(tallyResults.No.Int64()))
	fmt.Println("NWV", humanize.Comma(tallyResults.NoWithVeto.Int64()))
	fmt.Println("ABS", humanize.Comma(tallyResults.Abstain.Int64()))
}

func parseVotes(path string) (govtypes.Votes, error) {
	f, err := os.Open(filepath.Join(path, "votes_final.json"))
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
	// XXX workaround to unmarshal validators because proto doesn't support top-level array
	defer f.Close()
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
