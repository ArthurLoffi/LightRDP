# Invalid arguments
**Messages:**
- `Error: accepts 1 arg(s), received 0` (or `received 2`, etc.)
- `Error: unknown command "<NAME>" for "lightrdp"`
- `Error: unknown flag: --<NAME>`
- `Error: unknown shorthand flag: '<LETTER>' in -<FLAG>`

Each message is followed by the usage of the command.

## Description
`lightrdp` was started with a missing or invalid command, argument or option.

## Cause
- `lightrdp join` was used without the host ID, or with more than one argument.
- The command name was typed wrong.
- An option was typed wrong, or an old option with a single dash was used, such as `-bind`, `-mode` or `-peer`. Options now take two dashes, and `-mode` and `-peer` no longer exist.

## Possible solution
- Run `lightrdp join <HOST_ID>` with the host ID as the only argument.
- Run `lightrdp host` on the computer that will be accessed (replaces `-mode host`).
- Run `lightrdp --help` to see the commands, and `lightrdp <COMMAND> --help` to see the options of a command.
- Use two dashes for options, for example `--bind`, `--server` and `--ca`.

_Documented on: 2026-11-13 (updated 2026-11-21)_
