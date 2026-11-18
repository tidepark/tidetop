# testdata

These files check that the Go code draws exactly what the reference implementation draws.
The reference is tidetop's first version, written in JavaScript (it is not part of this repository).

- `frames.json.gz`: 716 screens (every scene, sizes, values, times, states, skins, split and joined windows) as the reference wrote them to the terminal. `internal/scenes` renders the same inputs and compares cell by cell.
- `feeds.json`: tidepic feeds and the values the reference computed from them (`internal/feed`).
- `math.json`: inputs and outputs of the math functions the scenes use, as 64-bit patterns. Go and JavaScript may differ in the last bit; `internal/jsmath` reports the count without failing.
- `sample-60.txt`: one picture written by `tidetop init`, compared with the reference's.
