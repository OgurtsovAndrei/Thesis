# Third-Party Notices

This product (the `Thesis` module) is licensed under the Apache License 2.0
(see `LICENSE`). It includes the following third-party code:

## Inlined forks (vendored, with modifications)

- **`succinct_bit_vector/rsdic`** — fork of
  [github.com/hillbig/rsdic](https://github.com/hillbig/rsdic), MIT License,
  Copyright (c) 2014 Daisuke Okanohara. The upstream `LICENSE` is retained in
  the package directory. Modifications: `MarshalBinary`/`UnmarshalBinary`
  reimplemented with `encoding/binary` (the `github.com/ugorji/go/codec`
  dependency was removed).

## Go module dependencies

All other third-party code is consumed as normal Go module dependencies
declared in `go.mod`; their licenses apply as published by their respective
authors.
