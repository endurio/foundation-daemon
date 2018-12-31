txscript
========

[![Build Status](http://img.shields.io/travis/endurio/ndrd.svg)](https://travis-ci.org/endurio/ndrd)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/endurio/ndrd/txscript)

Package txscript implements the Decred transaction script language.  There is
a comprehensive test suite.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing to use or validate Decred transaction scripts.

## Decred Scripts

Decred provides a stack-based, FORTH-like language for the scripts in
the Decred transactions.  This language is not turing complete
although it is still fairly powerful.

## Installation and Updating

```bash
$ go get -u github.com/endurio/ndrd/txscript
```

## Examples

* [Standard Pay-to-pubkey-hash Script](http://godoc.org/github.com/endurio/ndrd/txscript#example-PayToAddrScript)  
  Demonstrates creating a script which pays to a Decred address.  It also
  prints the created script hex and uses the DisasmString function to display
  the disassembled script.

* [Extracting Details from Standard Scripts](http://godoc.org/github.com/endurio/ndrd/txscript#example-ExtractPkScriptAddrs)  
  Demonstrates extracting information from a standard public key script.

* [Manually Signing a Transaction Output](http://godoc.org/github.com/endurio/ndrd/txscript#example-SignTxOutput)  
  Demonstrates manually creating and signing a redeem transaction.

## License

Package txscript is licensed under the [copyfree](http://copyfree.org) ISC
License.
