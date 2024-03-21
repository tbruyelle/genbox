# Methodology for Applying the $ATONE Distribution as Detailed in Govgen Proposal 001

This page will describe the methodology to apply the $ATONE distribution
detailed in GovGen [proposal 001][001].

> [!NOTE]
> While this documentation is related to Cosmos Hub [proposal 848][prop848] and
> GovGen [proposal 001][001], the code itself is supposed to be portable and
> reusable in other scenarios.

## Get the Cosmos Hub Proposal 848 Snapshot Data

Create a directory `data/prop848` and download the following files in that
directory:
- `votes.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/votes.json
- `delegations.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/delegations.json
- `active_validators.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/active_validators.json
- `prop.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/prop.json
- `balances.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/balances.json 
- `auth_genesis.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/auth_genesis.json

The way these files were extracted from Cosmos HUb [proposal 848][prop848]
snaphot is available [here](SNAPSHOT-EXTRACT.md). If you prefer you can start from a 
[Cosmos Hub node][gaia] and extract all the data, everything is detailed
[here](SNAPSHOT-EXTRACT.md) as already mentioned.

## Verify Cosmos Hub Proposal 848 Tally

A good way to verify correctness of the data is to use it to compute the
prop 848 tally and compare the results with the `TallyResult` field of the
proposal object stored on-chain.

You can achieve this by running the following command:
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

As printed in the output of the command, the diff is 0, meaning there is
no difference between the tally computed from the data and the `TallyResult`
field of the proposal.

## Consolidate accounts

The program allows you to create a `data/prop848/accounts.json` file that
consolidates all the relevant data for accounts. This file will be the starting
point for the following steps, and will be fed to this very same program
but using a different command.

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

It gives access to the unbonded and bonded amounts (here *liquid* and *staked*
amounts), the direct vote if present, the delegations and their relative vote.

> [!NOTE]
> `ModuleAccount` and `InterchainAccount` are skipped.

## Compute $ATONE distribution

Finally, let's compute the $ATONE distribution:

```
$ go run . distribution data/prop848/
```

The command above will output a table which provides an overview of the
resulting distribution.

But more importantly, the command will create a file
`data/prop848/airdrop.json` which for convenience you can find [here][airdrop]
already generated. The file lists all accounts and their relative proposed
$ATONE balance.

The following table is also provided for a quick recap of the employed
[methodology][001]:

|                     |  DNV      | YES | ABSTAIN | NO |    NWV    |
|---------------------|-----------|-----|---------|----|-----------|
| Bonded multiplier   | C x malus |  1  |    C    | 4  | 4 x bonus |
| Unbonded multiplier | C x malus |  -  |    -    | -  |     -     |

A specific effort is made to ensure that *non-voting* categories 
(*Did not Vote*, *Abstain* and *Not Staked (or Unbonded)*) do not end up holding
more than 1/3 of the supply. A tailored `C` multiplier is introduced to achieve
this. See the following section for details on how this is achieved.

To obtain the final distribution, we also apply a decimation factor `K = 0.1`
when computing balances. 

> [!IMPORTANT]
> The recap table above does not explicitly account for the `K = 0.1`
> decimation factor. It is considered implicit as it will be applied
> indiscriminately alongside any other category-specific multiplier

According to the current calculations -- which **may** change -- the potential
$ATONE distribution will be of around ~48.5 Millions.

|                       |   TOTAL    | DID NOT VOTE |    YES    |     NO     | NOWITHVETO |  ABSTAIN  | NOT STAKED |
|-----------------------|------------|--------------|-----------|------------|------------|-----------|------------|
| Distributed $ATONE    | 48,503,137 |    5,247,961 | 6,374,676 | 21,340,439 |  4,791,114 | 2,849,864 |  7,899,084 |
| Percentage over total |            | 11%          | 13%       | 44%        | 10%        | 6%        | 16%        |

As a comparison, here is the $ATOM distribution for [prop848]:

|                       |    TOTAL    | DID NOT VOTE |    YES     |     NO     | NOWITHVETO |  ABSTAIN   | NOT STAKED  |
|-----------------------|-------------|--------------|------------|------------|------------|------------|-------------|
| Total $ATOM           | 342,834,268 | 66,855,758   | 70,428,501 | 55,519,213 | 11,664,818 | 35,679,919 | 102,686,059 |
| Percentage over total |             | 20%          | 21%        | 16%        | 3%         | 10%        | 30%         |


## Multiplier Formula

This section details how the multiplier `C` for the *non-voting* $ATOM 
(*Abstain*, *Did Not Vote*, *Not Staked*) is calculated to result in them having
less than or equal to 1/3 of the final $ATONE supply, or in general a fixed
target percentage `t`.

Let's define the following variables:
- `C` is the multiplier we want to compute to be applied to non-voting
  categories, i.e. *Not Staked*, *Abstain* and *Did Not Vote*
- `t` the target relative percentage of $ATONE supply distributed to non-voting
  categories we want to achieve (known, 33%)
- `X` is the supply in $ATOM (known)
- `Y` is the supply in $ATONE
- both `X` and `Y` will have an annotation indicating the portion of the supply:
    - `Y` voted Yes
    - `A` voted Abstain
    - `N` voted No
    - `NWV` voted No With Veto
    - `DNV` Did Not Vote
    - `U` Unbonded (Not Staked)

  For example, $X_{A}$ is the number of $ATOM that has voted ABSTAIN.

Using the above notation, we can express what we want to achieve mathemacally as:
```math
\frac{Y_{A} + Y_{DNV} + Y_{U}}{Y_{A} + Y_{DNV} + Y_{U} + Y_{Y} + Y_{N} + Y_{NWV}} \leq t
```
It is know from the specifications of the distribution mechanism, *disregarding
any additional bonus or malus* for simplicity and also because they are meant to
be applied *additionally* to the *C* multiplier:
```math
\left\{
\begin{aligned}
& Y_{Y} = X_{Y} \\
& Y_{N} = 4 \cdot X_{N} \\
& Y_{NWV} = 4 \cdot X_{NWV} \\
& Y_{A} + Y_{DNV} + Y_{U} = C \cdot X_{A} + C \cdot X_{DNV} + C \cdot X_{U} = C \cdot (X_{A} + X_{DNV} + X_{U})
\end{aligned}
\right.
```

Which if plugged in the above equation gives:
```math
\frac{C \cdot (X_{A} + X_{DNV} + X_{U})}{C \cdot (X_{A} + X_{DNV} + X_{U}) + X_{Y} + 4 \cdot X_{N} + 4 \cdot X_{NWV}} \leq t
```

Finally, let's isolate `C`:
```math
\begin{align}
C \cdot (X_{A} + X_{DNV} + X_{U}) &\leq t \cdot C \cdot (X_{A} + X_{DNV} + X_{U}) + t \cdot (X_{Y} + 4 \cdot X_{N} + 4 \cdot X_{NWV}) \\[10pt]
(1 - t) \cdot C \cdot (X_{A} + X_{DNV} + X_{U}) &\leq  t \cdot (X_{Y} + 4 \cdot X_{N} + 4 \cdot X_{NWV}) \\[10pt]
C  &\leq  \frac{t}{1-t} \cdot \frac{(X_{Y} + 4 \cdot X_{N} + 4 \cdot X_{NWV})}{(X_{A} + X_{DNV} + X_{U})}
\end{align}
```
Which gives the final formula described in the [proposal 001][001].


[001]: https://github.com/giunatale/govgen-proposals/blob/giunatale/atone_distribution/001_ATONE_DISTRIBUTION.md
[airdrop]: https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/airdrop.json
[prop848]: https://www.mintscan.io/cosmos/proposals/848
[gaia]: https://github.com/cosmos/gaia
