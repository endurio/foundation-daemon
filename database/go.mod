module github.com/endurio/ndrd/database

require (
	github.com/btcsuite/goleveldb v1.0.0
	github.com/endurio/ndrd/chaincfg v1.2.0
	github.com/endurio/ndrd/chaincfg/chainhash v1.0.1
	github.com/endurio/ndrd/dcrutil v1.1.1
	github.com/endurio/ndrd/wire v1.2.0
	github.com/decred/slog v1.0.0
	github.com/fsnotify/fsnotify v1.4.7 // indirect
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.4.0
	github.com/kr/pretty v0.1.0 // indirect
	golang.org/x/net v0.0.0-20180808004115-f9ce57c11b24 // indirect
	golang.org/x/sync v0.0.0-20181108010431-42b317875d0f // indirect
	golang.org/x/sys v0.0.0-20181206074257-70b957f3b65e // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

replace (
	github.com/endurio/ndrd/chaincfg => ../chaincfg
	github.com/endurio/ndrd/chaincfg/chainhash => ../chaincfg/chainhash
	github.com/endurio/ndrd/dcrec => ../dcrec
	github.com/endurio/ndrd/dcrec/edwards => ../dcrec/edwards
	github.com/endurio/ndrd/dcrec/secp256k1 => ../dcrec/secp256k1
	github.com/endurio/ndrd/dcrutil => ../dcrutil
	github.com/endurio/ndrd/wire => ../wire
)
