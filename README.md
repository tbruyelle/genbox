# govbox

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
- `votes.json`
- `delegations.json`
- `active_validators.json`
- `prop.json`
- `balances.json`
- `auth_genesis.json`

The way the data was extracted is documented [here](SNAPSHOT-EXTRACT.md).

See [PROP-001](PROP-001.md) to have an usage demonstration for the GovGen
Proposal 001.

