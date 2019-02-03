// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC websocket notifications that are
// supported by a wallet server.

package ndrjson

const (
	// AccountBalanceNtfnMethod is the method used for account balance
	// notifications.
	AccountBalanceNtfnMethod = "accountbalance"

	// DcrdConnectedNtfnMethod is the method used for notifications when
	// a wallet server is connected to a chain server.
	DcrdConnectedNtfnMethod = "ndrdconnected"

	// NewTxNtfnMethod is the method used to notify that a wallet server has
	// added a new transaction to the transaction store.
	NewTxNtfnMethod = "newtx"

	// WalletLockStateNtfnMethod is the method used to notify the lock state
	// of a wallet has changed.
	WalletLockStateNtfnMethod = "walletlockstate"
)

// AccountBalanceNtfn defines the accountbalance JSON-RPC notification.
type AccountBalanceNtfn struct {
	Account   string
	Balance   float64 // In DCR
	Confirmed bool    // Whether Balance is confirmed or unconfirmed.
}

// NewAccountBalanceNtfn returns a new instance which can be used to issue an
// accountbalance JSON-RPC notification.
func NewAccountBalanceNtfn(account string, balance float64, confirmed bool) *AccountBalanceNtfn {
	return &AccountBalanceNtfn{
		Account:   account,
		Balance:   balance,
		Confirmed: confirmed,
	}
}

// DcrdConnectedNtfn defines the ndrddconnected JSON-RPC notification.
type DcrdConnectedNtfn struct {
	Connected bool
}

// NewDcrdConnectedNtfn returns a new instance which can be used to issue a
// ndrddconnected JSON-RPC notification.
func NewDcrdConnectedNtfn(connected bool) *DcrdConnectedNtfn {
	return &DcrdConnectedNtfn{
		Connected: connected,
	}
}

// NewTxNtfn defines the newtx JSON-RPC notification.
type NewTxNtfn struct {
	Account string
	Details ListTransactionsResult
}

// NewNewTxNtfn returns a new instance which can be used to issue a newtx
// JSON-RPC notification.
func NewNewTxNtfn(account string, details ListTransactionsResult) *NewTxNtfn {
	return &NewTxNtfn{
		Account: account,
		Details: details,
	}
}

// WalletLockStateNtfn defines the walletlockstate JSON-RPC notification.
type WalletLockStateNtfn struct {
	Locked bool
}

// NewWalletLockStateNtfn returns a new instance which can be used to issue a
// walletlockstate JSON-RPC notification.
func NewWalletLockStateNtfn(locked bool) *WalletLockStateNtfn {
	return &WalletLockStateNtfn{
		Locked: locked,
	}
}

func init() {
	// The commands in this file are only usable with a wallet server via
	// websockets and are notifications.
	flags := UFWalletOnly | UFWebsocketOnly | UFNotification

	MustRegisterCmd(AccountBalanceNtfnMethod, (*AccountBalanceNtfn)(nil), flags)
	MustRegisterCmd(DcrdConnectedNtfnMethod, (*DcrdConnectedNtfn)(nil), flags)
	MustRegisterCmd(NewTxNtfnMethod, (*NewTxNtfn)(nil), flags)
	MustRegisterCmd(WalletLockStateNtfnMethod, (*WalletLockStateNtfn)(nil), flags)
}
