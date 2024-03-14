# genbox

Tools, Scripts and Code snippets for GovGen proposals.
- validate governance data from a snapshot
- turn data into a genesis
- balanced auto staking genesis
- distribution analysis
- etc...

```
Usage: go run . COMMAND PATH
```

Where PATH is a directory containing the following files:
- `votes.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/votes.json
- `delegations.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/delegations.json
- `active_validators.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/active_validators.json
- `prop.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/prop.json
- `balances.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/balances.json 
- `auth_genesis.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/auth_genesis.json

The way the data was extracted is documented [here](SNAPSHOT-EXTRACT.md).

## Verify the tally

Considering all these files downloaded in the `data/prop848` diretcory, you can
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
