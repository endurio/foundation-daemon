rpctest
=======

[![Build Status](http://img.shields.io/travis/endurio/ndrd.svg)](https://travis-ci.org/endurio/ndrd)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/endurio/ndrd/rpctest)

Package rpctest provides a ndrd-specific RPC testing harness crafting and
executing integration tests by driving a `ndrd` instance via the `RPC`
interface. Each instance of an active harness comes equipped with a simple
in-memory HD wallet capable of properly syncing to the generated chain,
creating new addresses, and crafting fully signed transactions paying to an
arbitrary set of outputs. 

This package was designed specifically to act as an RPC testing harness for
`ndrd`. However, the constructs presented are general enough to be adapted to
any project wishing to programmatically drive a `ndrd` instance of its
systems/integration tests. 

## Installation and Updating

```bash
$ go get -u github.com/endurio/ndrd/rpctest
```

## License


Package rpctest is licensed under the [copyfree](http://copyfree.org) ISC
License.

