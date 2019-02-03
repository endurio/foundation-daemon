module github.com/endurio/ndrd/hdkeychain

require (
	github.com/decred/base58 v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/ndrec v0.0.0-20180721005914-d26200ec716b
	github.com/endurio/ndrd/ndrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/ndrutil v1.1.1
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
