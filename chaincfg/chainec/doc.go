// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
Package chainec provides wrapper functions to abstract the ec functions.

Overview

This package provides thin wrappers around the ec or crypto function used
to make it easier to go from btcec (btcd) to ed25519 (endurio) for example
without changing the main body of the code.

*/
package chainec
