module github.com/endurio/ndrd/connmgr

require (
	github.com/endurio/ndrd/chaincfg v1.1.1
	github.com/endurio/ndrd/wire v1.2.0
	github.com/decred/slog v1.0.0
)

replace (
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/wire => ../wire
)
