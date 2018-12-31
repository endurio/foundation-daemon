module github.com/endurio/ndrd/hdkeychain

require (
	github.com/decred/base58 v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/dcrec v0.0.0-20180721005914-d26200ec716b
	github.com/endurio/ndrd/dcrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/dcrutil v1.1.1
)

replace (
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/dcrec => ../dcrec
	github.com/endurio/ndrd/dcrec/edwards => ../dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/dcrutil => ../dcrutil
	github.com/endurio/ndrd/wire => ../wire
)
