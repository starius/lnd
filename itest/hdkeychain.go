package itest

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/lightningnetwork/lnd/keychain"
)

// deriveChildren derives a key from master key using derivation path.
func deriveChildren(key *hdkeychain.ExtendedKey, path []uint32) (
	*hdkeychain.ExtendedKey, error) {

	var currentKey = key
	for idx, pathPart := range path {
		derivedKey, err := currentKey.DeriveNonStandard(pathPart)
		if err != nil {
			return nil, err
		}

		// There's this special case in lnd's wallet (btcwallet) where
		// the coin type and account keys are always serialized as a
		// string and encrypted, which actually fixes the key padding
		// issue that makes the difference between DeriveNonStandard and
		// Derive. To replicate lnd's behavior exactly, we need to
		// serialize and de-serialize the extended key at the coin type
		// and account level (depth = 2 or depth = 3). This does not
		// apply to the default account (id = 0) because that is always
		// derived directly.
		depth := derivedKey.Depth()
		keyID := pathPart - hdkeychain.HardenedKeyStart
		nextID := uint32(0)
		if depth == 2 && len(path) > 2 {
			nextID = path[idx+1] - hdkeychain.HardenedKeyStart
		}
		if (depth == 2 && nextID != 0) || (depth == 3 && keyID != 0) {
			currentKey, err = hdkeychain.NewKeyFromString(
				derivedKey.String(),
			)
			if err != nil {
				return nil, err
			}
		} else {
			currentKey = derivedKey
		}
	}

	return currentKey, nil
}

// hdKeyRing performs public derivation of various keys used within the
// peer-to-peer network, and also within any created contracts. All derivation
// required by the hdKeyRing is based off of public derivation, so a system with
// only an extended public key (for the particular purpose+family) can derive
// this set of keys.
type hdKeyRing struct {
	ExtendedKey *hdkeychain.ExtendedKey
	ChainParams *chaincfg.Params
}

// DeriveNextKey attempts to derive the *next* key within the key family
// (account in BIP43) specified. This method should return the next external
// child within this branch.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (r *hdKeyRing) DeriveNextKey(_ keychain.KeyFamily) (
	keychain.KeyDescriptor, error) {

	return keychain.KeyDescriptor{}, nil
}

// DeriveKey attempts to derive an arbitrary key specified by the passed
// KeyLocator. This may be used in several recovery scenarios, or when manually
// rotating something like our current default node key.
//
// NOTE: This is part of the keychain.KeyRing interface.
func (r *hdKeyRing) DeriveKey(keyLoc keychain.KeyLocator) (
	keychain.KeyDescriptor, error) {

	var empty = keychain.KeyDescriptor{}
	const HardenedKeyStart = uint32(hdkeychain.HardenedKeyStart)
	derivedKey, err := deriveChildren(r.ExtendedKey, []uint32{
		HardenedKeyStart + uint32(keychain.BIP0043Purpose),
		HardenedKeyStart + r.ChainParams.HDCoinType,
		HardenedKeyStart + uint32(keyLoc.Family),
		0,
		keyLoc.Index,
	})
	if err != nil {
		return empty, err
	}

	derivedPubKey, err := derivedKey.ECPubKey()
	if err != nil {
		return empty, err
	}

	return keychain.KeyDescriptor{
		KeyLocator: keychain.KeyLocator{
			Family: keyLoc.Family,
			Index:  keyLoc.Index,
		},
		PubKey: derivedPubKey,
	}, nil
}

var _ keychain.KeyRing = (*hdKeyRing)(nil)
