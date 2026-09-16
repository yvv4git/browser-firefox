# connect

Connects to the Firefox container over WebDriver BiDi,
prints the session status and lists the currently open
browser contexts.

## Usage

The container must be running first:

    cd .. && make compose-up

Run the example:

    go run ./connect -addr http://localhost:9222

## Flags

| Flag    | Default                 | Description                                        |
| ------- | ----------------------- | -------------------------------------------------- |
| `-addr` | `http://localhost:9222` | WebDriver BiDi endpoint of the Firefox container   |

Run `go run ./connect -h` for the full list.
