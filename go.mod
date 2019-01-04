module github.com/endurio/ndrd

require (
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/btcsuite/winsvc v1.0.0
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/decred/base58 v1.0.0
	github.com/decred/slog v1.0.0
	github.com/endurio/ndrd/addrmgr v1.0.2
	github.com/endurio/ndrd/blockchain v1.1.1
	github.com/endurio/ndrd/certgen v1.0.2
	github.com/endurio/ndrd/chaincfg v1.2.1
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/connmgr v1.0.2
	github.com/endurio/ndrd/database v1.0.3
	github.com/endurio/ndrd/dcrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/dcrjson v1.1.0
	github.com/endurio/ndrd/dcrutil v1.2.0
	github.com/endurio/ndrd/fees v1.0.0
	github.com/endurio/ndrd/gcs v1.0.2
	github.com/endurio/ndrd/hdkeychain v1.1.1
	github.com/endurio/ndrd/mempool v1.1.0
	github.com/endurio/ndrd/mining v1.1.0
	github.com/endurio/ndrd/peer v1.1.0
	github.com/endurio/ndrd/rpcclient v1.1.0
	github.com/endurio/ndrd/txscript v1.0.2
	github.com/endurio/ndrd/wire v1.2.0
	github.com/gorilla/websocket v1.2.0
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/bitset v1.0.0
	github.com/jrick/logrotate v1.0.0
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9
)

replace (
	github.com/endurio/ndrd => ./
	github.com/endurio/ndrd/addrmgr => ./addrmgr
	github.com/endurio/ndrd/blockchain => ./blockchain
	github.com/endurio/ndrd/certgen => ./certgen
	github.com/endurio/ndrd/chaincfg => ./chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ./chaincfg/chainhash
	github.com/endurio/ndrd/connmgr => ./connmgr
	github.com/endurio/ndrd/database => ./database
	github.com/endurio/ndrd/dcrec => ./dcrec
	github.com/endurio/ndrd/dcrec/edwards => ./dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ./dcrec/secp256k1
	github.com/endurio/ndrd/dcrjson => ./dcrjson
	github.com/endurio/ndrd/dcrutil => ./dcrutil
	github.com/endurio/ndrd/fees => ./fees
	github.com/endurio/ndrd/gcs => ./gcs
	github.com/endurio/ndrd/hdkeychain => ./hdkeychain
	github.com/endurio/ndrd/limits => ./limits
	github.com/endurio/ndrd/mempool => ./mempool
	github.com/endurio/ndrd/mining => ./mining
	github.com/endurio/ndrd/peer => ./peer
	github.com/endurio/ndrd/rpcclient => ./rpcclient
	github.com/endurio/ndrd/txscript => ./txscript
	github.com/endurio/ndrd/wire => ./wire
)
