module github.com/endurio/ndrd/dcrutil

require (
	github.com/davecgh/go-spew v1.1.0
	github.com/decred/base58 v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/dcrec v0.0.0-20180721005212-59fe2b293f69
	github.com/endurio/ndrd/dcrec/edwards v0.0.0-20181208004914-a0816cf4301f
	github.com/endurio/ndrd/dcrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/wire v1.2.0
	golang.org/x/crypto v0.0.0-20180718160520-a2144134853f
)

replace (
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/dcrec => ../dcrec
	github.com/endurio/ndrd/dcrec/edwards => ../dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/wire => ../wire
)
