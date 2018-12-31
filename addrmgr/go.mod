module github.com/endurio/ndrd/addrmgr

require (
	github.com/decred/slog v1.0.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/wire v1.1.0
)

replace (
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/wire => ../wire
)
