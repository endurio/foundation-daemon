// Copyright (c) 2018 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"math"
	"time"

	"github.com/endurio/ndrd/wire"
)

// RegNetParams defines the network parameters for the regression test network.
// This should not be confused with the public test network or the simulation
// test network.  The purpose of this network is primarily for unit tests and
// RPC server tests.  On the other hand, the simulation test network is intended
// for full integration tests between different applications such as wallets,
// voting service providers, mining pools, block explorers, and other services
// that build on Decred.
//
// Since this network is only intended for unit testing, its values are subject
// to change even if it would cause a hard fork.
var RegNetParams = Params{
	Name:        "regnet",
	Net:         wire.RegNet,
	DefaultPort: "18655",
	DNSSeeds:    nil, // NOTE: There must NOT be any seeds.

	// Chain parameters
	GenesisBlock:             &regNetGenesisBlock,
	GenesisHash:              &regNetGenesisHash,
	PowLimit:                 regNetPowLimit,
	PowLimitBits:             0x207fffff,
	ReduceMinDifficulty:      false,
	MinDiffReductionTime:     0, // Does not apply since ReduceMinDifficulty false
	GenerateSupported:        true,
	MaximumBlockSizes:        []int{1000000, 1310720},
	MaxTxSize:                1000000,
	TargetTimePerBlock:       time.Second,
	WorkDiffAlpha:            1,
	WorkDiffWindowSize:       8,
	WorkDiffWindows:          4,
	TargetTimespan:           time.Second * 8, // TimePerBlock * WindowSize
	RetargetAdjustmentFactor: 4,

	// Subsidy parameters.
	BaseSubsidy:              50000000000,
	MulSubsidy:               100,
	DivSubsidy:               101,
	SubsidyReductionInterval: 128,
	WorkRewardProportion:     6,
	StakeRewardProportion:    3,
	BlockTaxProportion:       1,

	// Checkpoints ordered from oldest to newest.
	Checkpoints: nil,

	// Consensus rule change deployments.
	//
	// The miner confirmation window is defined as:
	//   target proof of work timespan / target proof of work spacing
	RuleChangeActivationThreshold: 1916, // 95% of MinerConfirmationWindow
	MinerConfirmationWindow:       2016, //
	Deployments: [DefinedDeployments]ConsensusDeployment{
		DeploymentTestDummy: {
			BitNumber:  28,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentCSV: {
			BitNumber:  0,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
		DeploymentSegwit: {
			BitNumber:  1,
			StartTime:  0,             // Always available for vote
			ExpireTime: math.MaxInt64, // Never expires
		},
	},

	// Enforce current block version once majority of the network has
	// upgraded.
	// 51% (51 / 100)
	// Reject previous block versions once a majority of the network has
	// upgraded.
	// 75% (75 / 100)
	BlockEnforceNumRequired: 51,
	BlockRejectNumRequired:  75,
	BlockUpgradeNumToCheck:  100,

	// AcceptNonStdTxs is a mempool param to either accept and relay
	// non standard txs to the network or reject them
	AcceptNonStdTxs: true,

	// Address encoding magics
	NetworkAddressPrefix: "R",
	PubKeyAddrID:         [2]byte{0x25, 0xe5}, // starts with Rk
	PubKeyHashAddrID:     [2]byte{0x0e, 0x00}, // starts with Rs
	PKHEdwardsAddrID:     [2]byte{0x0d, 0xe0}, // starts with Re
	PKHSchnorrAddrID:     [2]byte{0x0d, 0xc2}, // starts with RS
	ScriptHashAddrID:     [2]byte{0x0d, 0xdb}, // starts with Rc
	PrivateKeyID:         [2]byte{0x22, 0xfe}, // starts with Pr

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID: [4]byte{0xea, 0xb4, 0x04, 0x48}, // starts with rprv
	HDPublicKeyID:  [4]byte{0xea, 0xb4, 0xf9, 0x87}, // starts with rpub

	// BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	SLIP0044CoinType: 1, // SLIP0044, Testnet (all coins)
	LegacyCoinType:   1,

	// Decred PoS parameters
	CoinbaseMaturity: 16,

	// Decred organization related parameters
	//
	// Treasury address is a 3-of-3 P2SH going to a wallet with seed:
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// aardvark adroitness aardvark adroitness
	// briefcase
	// (seed 0x0000000000000000000000000000000000000000000000000000000000000000)
	//
	// This same wallet owns the three ledger outputs for regnet.
	//
	// P2SH details for regnet treasury:
	//
	// redeemScript: 53210323c1b9aa4facca85df363fb4abd5c52fe2af4746fbb5f99a6d
	// cc2edb633fe2a62103c2d8a61a2800092ddaf04ba30dfc7cf1ab4130ac1d2398ba15fc
	// 795b11bc690621035fe97a7b2d6b98242f4bfc33d86a564158b44634b93cdefa155909
	// 5d4bf6167853ae
	//   (3-of-3 multisig)
	// Pubkeys used:
	//   Rk8J2ZY5CkDLaBbAYqU7fb1Tr6nSwEACJ1j2oWAwuFZ26PyPeMXiB
	//   Rk8KEdGMGJiF27CZ8rw2gDPD7MkVGSPjHinXjtTZhoH8ZQ6UhJvhV
	//   Rk8JV484ePPX6vWZCfBX2Scme5XriXhzwmyaKSYQT64HTbkkfyzL3
	//
	// Organization address is RcQR65gasxuzf7mUeBXeAux6Z37joPuUwUN
	OrganizationPkScript:        hexDecode("a9146913bcc838bd0087fb3f6b3c868423d5e300078d87"),
	OrganizationPkScriptVersion: 0,
	BlockOneLedger:              BlockOneLedgerRegNet,
}
