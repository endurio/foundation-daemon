// Copyright (c) 2017-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errDuplicateBitNumber = errors.New("duplicate bit number")
	errInvalidMask        = errors.New("invalid mask")
	errNotConsecutive     = errors.New("choices not consecutive")
	errTooManyChoices     = errors.New("too many choices")
	errInvalidAbstain     = errors.New("invalid abstain bits")
	errInvalidBits        = errors.New("invalid vote bits")
	errInvalidIsAbstain   = errors.New("one and only one IsAbstain rule " +
		"violation")
	errInvalidIsNo      = errors.New("one and only one IsNo rule violation")
	errInvalidBothFlags = errors.New("IsNo and IsAbstain may not be both " +
		"set to true")
	errDuplicateChoiceId = errors.New("duplicate choice ID")
)

// bitsSet counts number of bits set.
// Proudly stolen from:
// https://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetKernighan
func bitsSet(bits uint16) uint {
	c := uint(0)
	for v := bits; v != 0; c++ {
		v &= v - 1
	}
	return c
}

// consecOnes count consecutive 1 bits set.
func consecOnes(bits uint16) uint {
	c := uint(0)
	for v := bits; v != 0; c++ {
		v = v & (v << 1)
	}
	return c
}

// shift calculates the number of bits that need shifting to get to an index.
func shift(mask uint16) uint {
	shift := uint(0)
	for {
		if mask&0x0001 == 0x0001 {
			break
		}
		shift++
		mask >>= 1
	}
	return shift
}

func validateChoices(mask uint16, choices []Choice) error {
	var (
		numAbstain, numNo int
	)

	// Check that mask is consecutive.
	if consecOnes(mask) != bitsSet(mask) {
		return errInvalidMask
	}

	// Check bits and choice bounds.
	if len(choices) > 1<<bitsSet(mask) {
		return errTooManyChoices
	}

	dups := make(map[string]struct{})
	s := shift(mask)
	for index, choice := range choices {
		// Check that choice 0 is the abstain vote.
		if mask&choice.Bits == 0 && !choice.IsAbstain {
			return errInvalidAbstain
		}

		// Check mask bits.
		if mask&choice.Bits != choice.Bits {
			return errInvalidBits
		}

		// Check that index is consecutive.  This test is below the
		// Check mask bits one for testing reasons.  Leave it here.
		if uint16(index) != choice.Bits>>s {
			return errNotConsecutive
		}

		// Check that both flags aren't set to true.
		if choice.IsAbstain && choice.IsNo {
			return errInvalidBothFlags
		}

		// Count flags.
		if choice.IsAbstain {
			numAbstain++
		}
		if choice.IsNo {
			numNo++
		}

		// Check for duplicates.
		id := strings.ToLower(choice.Id)
		_, found := dups[id]
		if found {
			return errDuplicateChoiceId
		}
		dups[id] = struct{}{}
	}

	// Check that there is only one IsNo and IsAbstain flag set to true.
	if numAbstain != 1 {
		return errInvalidIsAbstain
	}
	if numNo != 1 {
		return errInvalidIsNo
	}

	return nil
}

func validateAgenda(vote Vote) error {
	return validateChoices(vote.Mask, vote.Choices)
}

func validateDeployments(deployments [DefinedDeployments]ConsensusDeployment) (int, error) {
	var bits uint8
	for index, deployment := range deployments {
		// Check for duplicates.
		if bits&deployment.BitNumber != 0 {
			return index, errDuplicateBitNumber
		}
		bits |= deployment.BitNumber
	}
	return -1, nil
}

func validateAgendas() {
	allParams := []*Params{&MainNetParams, &TestNet3Params, &SimNetParams,
		&RegNetParams}
	for _, params := range allParams {
		index, err := validateDeployments(params.Deployments)
		if err != nil {
			e := fmt.Sprintf("invalid agenda on %v "+
				"bit number %v: %v", params.Name,
				params.Deployments[index].BitNumber,
				err)
			panic(e)
		}
	}
}

func init() {
	validateAgendas()
}
