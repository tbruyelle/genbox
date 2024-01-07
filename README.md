# govgenesis

Tool to validate governance data from a snapshot and turn data into a genesis.

```
Usage: go run main.go [tally|genesis] PATH
```

Where PATH is a directory containing the following files:
- `votes.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/votes.json
- `delegations.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/delegations.json
- `active_validators.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/active_validators.json
- `prop.json` https://atomone.fra1.digitaloceanspaces.com/cosmoshub-4/prop848/prop.json

