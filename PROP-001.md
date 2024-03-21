# Govgen Proposal 001 methodology

This page will describe the methodology to apply the $ATONE distribution
detailed in [proposal 001][001].

> [!NOTE]
> While this documentation is related to [proposal 848][prop848], you can
> easily use it for any other proposal.
> The code itself isn't related to proposal 848.

## Get the snapshot data

Create a directory `data/prop848` and download the following files in that
directory:
- `votes.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/votes.json
- `delegations.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/delegations.json
- `active_validators.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/active_validators.json
- `prop.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/prop.json
- `balances.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/balances.json 
- `auth_genesis.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/auth_genesis.json

The way those files were extracted from the snaphots is available
[here](SNAPSHOT-EXTRACT.md). If you prefer you can start from a Gaia blockchain
node and extract all the data, everything is detailed in the link above.

## Verify tally

A good way to check the correctness of the data is to compute the prop 848
tally from the data and compare it with the proposal `TallyResult` field stored
in the blockchain's store.

You can achieve that by running this command:
```
$ go run . tally data/prop848/
173,165 votes
180 validators
1,061,423 delegations for 765,656 delegators
1,061,762 account balances
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

As printed in the output of the command, the diff is always 0, meaning there's
no difference between the tally computed from the data and the `TallyResult`
field of the proposal.

## Consolidate accounts

The program allows you to create a `data/prop848/accounts.json` file that
consolidates all the interesting data from an account. This file will be used
by the following command to compute other things.

```
$ go run . accounts data/prop848/
173,165 votes
180 validators
1,061,423 delegations for 765,656 delegators
1,061,762 account balances
1,948,588 accounts
data/prop848/accounts.json file created.
```

For example, here is the representation of an account in this file:

```json
{
    "Address": "cosmos1zujjhe8j7fe0fzkxf4addzudx0s2j0zrwuyl2z",
    "Type": "/cosmos.auth.v1beta1.BaseAccount",
    "LiquidAmount": "155159.000000000000000000",
    "StakedAmount": "12404000.482447507703078623",
    "Vote": [
      {
        "option": 1,
        "weight": "1.000000000000000000"
      }
    ],
    "Delegations": [
      {
        "Amount": "12404000.481949488362528991",
        "ValidatorAddress": "cosmosvaloper130mdu9a0etmeuw52qfxk73pn0ga6gawkxsrlwf",
        "Vote": [
          {
            "option": 1,
            "weight": "1.000000000000000000"
          }
        ]
      },
      {
        "Amount": "0.000498019340549632",
        "ValidatorAddress": "cosmosvaloper16k579jk6yt2cwmqx9dz5xvq9fug2tekvlu9qdv",
        "Vote": [
          {
            "option": 1,
            "weight": "1.000000000000000000"
          }
        ]
      }
    ],
}
```

It gives access to the liquid and staked amount, the vote, the delegations and
their relative vote.

> [!NOTE]
> `ModuleAccount` and `InterchainAccount` are skipped.

## Compute $ATONE distribution

Finally, let's compute the $ATONE distribution:

```
$ go run . distribution data/prop848/
```

The command above will output the resulting data and a table, which shows the distribution.

But more importantly, the command will create a file
`data/prop848/airdrop.json` which you can find [here][airdrop]. The file lists
all accounts and their relative future $ATONE balance.

The resulting supply will be of 485,031,369 $ATONE, distributed as follows:

|                       | DID NOT VOTE |    YES     |     NO      | NOWITHVETO |  ABSTAIN   | NOT STAKED |
|-----------------------|--------------|------------|-------------|------------|------------|------------|
| Distributed           |   52,479,607 | 63,746,761 | 213,404,392 | 47,911,135 | 28,498,638 | 78,990,836 |
| Percentage over total | 11%          | 13%        | 44%         | 10%        | 6%         | 16%        |

A specific effort is made to ensure that non-voters (DID NOT VOTE, ABSTRAIN and
NOT STAKED) don't hold more than 1/3 of the supply.

As a comparison, here is the $ATOM distribution for [prop848] ($ATOM supply was
342,834,268):

|                       | DID NOT VOTE |    YES     |     NO     | NOWITHVETO |  ABSTAIN   | NOT STAKED  |
|-----------------------|--------------|------------|------------|------------|------------|-------------|
| Distributed           |   66,855,758 | 70,428,501 | 55,519,213 | 11,664,818 | 35,679,919 | 102,686,059 |
| Percentage over total | 20%          | 21%        | 16%        | 3%         | 10%        | 30%         |

[001]: https://github.com/giunatale/govgen-proposals/blob/giunatale/atone_distribution/001_ATONE_DISTRIBUTION.md
[airdrop]: https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/airdrop.json
[prop848]: https://www.mintscan.io/cosmos/proposals/848
