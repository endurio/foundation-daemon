module github.com/endurio/ndrd/mining

require (
	github.com/endurio/ndrd/blockchain v1.0.1
	github.com/endurio/ndrd/blockchain/stake v1.1.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/dcrutil v1.2.0
	github.com/endurio/ndrd/wire v1.2.0
)

replace (
	github.com/endurio/ndrd/blockchain => ../blockchain
	github.com/endurio/ndrd/blockchain/stake => ../blockchain/stake
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/database => ../database
	github.com/endurio/ndrd/dcrec => ../dcrec
	github.com/endurio/ndrd/dcrec/edwards => ../dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/dcrutil => ../dcrutil
	github.com/endurio/ndrd/gcs => ../gcs
	github.com/endurio/ndrd/txscript => ../txscript
	github.com/endurio/ndrd/wire => ../wire
)
