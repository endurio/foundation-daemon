module github.com/endurio/ndrd/fees

require (
	github.com/btcsuite/goleveldb v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/ndrutil v1.2.0
	github.com/decred/slog v1.0.0
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/jessevdk/go-flags v1.4.0
	golang.org/x/sys v0.0.0-20180831094639-fa5fdf94c789 // indirect
)

replace (
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/ndrec => ../ndrec
	github.com/endurio/ndrd/ndrec/edwards => ../ndrec/edwards
	github.com/endurio/ndrd/ndrec/secp256k1 => ../ndrec/secp256k1
	github.com/endurio/ndrd/ndrutil => ../ndrutil
	github.com/endurio/ndrd/wire => ../wire
)
