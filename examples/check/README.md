# check

Connects to the Firefox container over WebDriver BiDi,
loads a page and demonstrates basic interactions:

- prints the page title and URL
- prints the size of the page HTML
- saves a screenshot to a PNG file

## Usage

The container must be running first:

    cd .. && make compose-up

Run the example:

    go run ./check -addr http://localhost:9222 https://www.wikipedia.org

With a custom screenshot path:

    go run ./check -addr http://localhost:9222 -output wikipedia.png https://example.com

## Flags

| Flag      | Default                 | Description                                        |
| --------- | ----------------------- | -------------------------------------------------- |
| `-addr`   | `http://localhost:9222` | WebDriver BiDi endpoint of the Firefox container   |
| `-output` | `check.png`             | Screenshot output file                             |

Run `go run ./check -h` for the full list.
