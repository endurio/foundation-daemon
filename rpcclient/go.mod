module github.com/endurio/ndrd/rpcclient

require (
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/davecgh/go-spew v1.1.0
	github.com/decred/slog v1.0.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/gcs v1.0.1
	github.com/endurio/ndrd/ndrjson v1.0.0
	github.com/endurio/ndrd/ndrutil v1.1.1
	github.com/endurio/ndrd/wire v1.2.0
	github.com/gorilla/websocket v1.2.0
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
	github.com/endurio/ndrd/ndrjson => ../ndrjson
	github.com/endurio/ndrd/ndrutil => ../ndrutil
	github.com/endurio/ndrd/txscript => ../txscript
	github.com/endurio/ndrd/wire => ../wire
)
