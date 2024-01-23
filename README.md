# govgenesis

Tool to validate governance data from a snapshot and turn data into a genesis.

```
Usage: go run . [tally|genesis] PATH
```

Where PATH is a directory containing the following files:
- `votes.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/votes.json
- `delegations.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/delegations.json
- `active_validators.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/active_validators.json
- `prop.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/prop.json

Considering all these files downloaded in the `data/prop848` direcory, you can
compute the tally and compare it to the prop `FinalTallyResult` field.

```
$ go run . tally data/prop848

173,165 votes
180 validators
1,061,423 delegations for 765,656 delegators
Computed total voting power 177,825,601,877,018
Yes percent: 0.517062127500689774
--- TALLY RESULT ---
+-----------+------------+------------+------------+------------+-------------+
|           |    YES     |     NO     | NOWITHVETO |  ABSTAIN   |    TOTAL    |
+-----------+------------+------------+------------+------------+-------------+
| computed  | 73,165,203 | 56,667,011 | 11,669,549 | 36,323,836 | 177,825,601 |
| from prop | 73,165,203 | 56,667,011 | 11,669,549 | 36,323,836 | 177,825,601 |
| diff      |          0 |          0 |          0 |          0 |           0 |
+-----------+------------+------------+------------+------------+-------------+
```

which shows that the tally calculated from these files is exactly the same as
the tally from the prop stored in the blockchain data.

# Data extraction

To extract the data, 2 snapshots are needed, the one where the tally happened,
to fetch the validators and the delegations, and the one just before, to get
the votes (because votes are removed during the tally). Let's call these files
- snapshot.json (where the tally happened)
- snaphost-1.json (the block just before)

## Get direct & indirect voters

While direct voters are easy to extract, indirect voters must be determined by
iterating over delegations and correlating them with validator votes.

#### Get all direct voters

```sh
$ jq '[.app_state.gov.votes[] | select(.proposal_id == "848")]' snapshot-1.json > votes.json
```

We need to manually add the last votes from block where the tally takes place,
for instance:

```sh
$ jq '. += [{
  "option": "VOTE_OPTION_YES",
  "options": [
    {
      "option": "VOTE_OPTION_YES",
      "weight": "1.000000000000000000"
    }
  ],
  "proposal_id": "848",
  "voter": "cosmos1jq6rpkf233jq9h98tlarzk8w3pl3lx87sv3t28"
}]' votes.json > votes_final.json
```

If the final votes have duplicates, because the user have voted more than one 
time, we need to eliminate the first votes and keep only the last ones (maybe
this is something that should be hanlded in the code).

#### Get all delegations

```sh
$ jq '.app_state.staking.delegations' snapshot.json > delegations.json
```

#### Get active bonded validators

```sh
$ jq '.app_state.staking.validators' snapshot.json > validators.json
```

To have the active set, we need to:
- Get the `max_validator` parameters:
```sh
$ jq '.app_state.staking.params.max_validators' snapshot.json
180
```
- Filter out bonded validators
- Sort by the `tokens` field (descending)
- Limit to `max_validators`

```sh
$ jq '[.[] | select(.status == "BOND_STATUS_BONDED")] | sort_by(.tokens|tonumber) | reverse | .[:180]' validators.json > active_validators.json
```

Now we have only the active validators.

This procedures follows the code of the [`staking.Keeper.IterateBondedValidatorsByPower()`][code-validators]
function, which is used in the [`x/gov.Keeper.Tally()`][code-tally] function.

#### Get proposal

```sh
jq '.app_state.gov.proposals[] | select(.proposal_id == "82") '  snapshot.json > prop.json
```

### Get balances

```sh
jq '.app_state.bank.balances' snapshot.json > balances.json
```
