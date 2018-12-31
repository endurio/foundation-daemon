module github.com/endurio/ndrd/chaincfg

require (
	github.com/davecgh/go-spew v1.1.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/dcrec/edwards v0.0.0-20181208004914-a0816cf4301f
	github.com/endurio/ndrd/dcrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/wire v1.2.0
)

replace (
	github.com/endurio/ndrd/chaincfg => ./
	github.com/endurio/ndrd/chaincfg/chainhash => ./chainhash
	github.com/endurio/ndrd/dcrec/edwards => ../dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/wire => ../wire
)
