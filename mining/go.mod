module github.com/endurio/ndrd/mining

require (
	github.com/endurio/ndrd/blockchain v1.0.1
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/ndrutil v1.2.0
	github.com/endurio/ndrd/wire v1.2.0
)

replace (
	github.com/endurio/ndrd/blockchain => ../blockchain
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/database => ../database
	github.com/endurio/ndrd/gcs => ../gcs
	github.com/endurio/ndrd/ndrec => ../ndrec
	github.com/endurio/ndrd/ndrec/edwards => ../ndrec/edwards
	github.com/endurio/ndrd/ndrec/secp256k1 => ../ndrec/secp256k1
	github.com/endurio/ndrd/ndrutil => ../ndrutil
	github.com/endurio/ndrd/txscript => ../txscript
	github.com/endurio/ndrd/wire => ../wire
)
