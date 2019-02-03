module github.com/endurio/ndrd/txscript

require (
	github.com/decred/slog v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/ndrec v0.0.0-20180721031028-5369a485acf6
	github.com/endurio/ndrd/ndrec/edwards v0.0.0-20181208004914-a0816cf4301f
	github.com/endurio/ndrd/ndrec/secp256k1 v1.0.1
	github.com/endurio/ndrd/ndrutil v1.1.1
	github.com/endurio/ndrd/wire v1.2.0
	golang.org/x/crypto v0.0.0-20180718160520-a2144134853f
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
