module github.com/endurio/ndrd/mempool

require (
	github.com/endurio/ndrd/blockchain v1.0.1
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/ndrec v0.0.0-20180801202239-0761de129164
	github.com/endurio/ndrd/ndrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/ndrjson v1.0.0
	github.com/endurio/ndrd/ndrutil v1.2.0
	github.com/endurio/ndrd/mining v1.1.0
	github.com/endurio/ndrd/txscript v1.0.2
	github.com/endurio/ndrd/wire v1.2.0
	github.com/decred/slog v1.0.0
)

replace (
	github.com/endurio/ndrd/blockchain => ../blockchain
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/database => ../database
	github.com/endurio/ndrd/ndrec/secp256k1 => ../ndrec/secp256k1
	github.com/endurio/ndrd/ndrjson => ../ndrjson
	github.com/endurio/ndrd/ndrutil => ../ndrutil
	github.com/endurio/ndrd/gcs => ../gcs
	github.com/endurio/ndrd/mining => ../mining
	github.com/endurio/ndrd/txscript => ../txscript
	github.com/endurio/ndrd/wire => ../wire
)
