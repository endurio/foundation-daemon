fees
=======


[![Build Status](http://img.shields.io/travis/endurio/ndrd.svg)](https://travis-ci.org/endurio/ndrd)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/endurio/ndrd/fees)

Package fees provides endurio-specific methods for tracking and estimating fee
rates for new transactions to be mined into the network. Fee rate estimation has
two main goals:

- Ensuring transactions are mined within a target _confirmation range_
  (expressed in blocks);
- Attempting to minimize fees while maintaining be above restriction.

This package was started in order to resolve issue endurio/ndrd#1412 and related.
See that issue for discussion of the selected approach.

This package was developed for ndrd, a full-node implementation of Decred which
is under active development.  Although it was primarily written for
ndrd, this package has intentionally been designed so it can be used as a
standalone package for any projects needing the functionality provided.

## Installation and Updating

```bash
$ go get -u github.com/endurio/ndrd/fees
```

## License

Package dcrutil is licensed under the [copyfree](http://copyfree.org) ISC
License.
