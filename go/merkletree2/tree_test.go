package merkletree2

import (
	"context"
	"fmt"
	"testing"

	"github.com/keybase/client/go/logger"

	"github.com/stretchr/testify/require"
)

func TestEmptyTree(t *testing.T) {
	config1bit, config2bits, config3bits := getTreeCfgsWith1_2_3BitsPerIndex(t)
	kvps1_1bit, kvps2_1bit, kvps3_1bit := getSampleKVPS1bit()
	kvps1_3bits, kvps2_3bits, kvps3_3bits := getSampleKVPS3bits()

	tests := []struct {
		cfg   TreeConfig
		kvps1 []KeyValuePair
		kvps2 []KeyValuePair
		kvps3 []KeyValuePair
	}{
		{config1bit, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config2bits, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config3bits, kvps1_3bits, kvps2_3bits, kvps3_3bits},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v bits tree", test.cfg.bitsPerIndex), func(t *testing.T) {
			tree := NewTree(test.cfg, &InMemoryStorageEngine{}, logger.NewTestLogger(t))

			seq, root, hash, err := tree.GetLatestRoot(context.TODO(), nil)
			require.NoError(t, err)
			require.Equal(t, Seqno(0), seq, "Tree should have Seqno 0 as no insertions were made, got %v instead", seq)
			require.Nil(t, root.BareRootHash, "Tree root should not have a bareRootHash as no insertions were made")
			require.Nil(t, hash, "Tree root should not have a root hash as no insertions were made")

			require.NoError(t, err)
			for _, kvp := range test.kvps1 {
				_, _, err := tree.GetKeyValuePairUnsafe(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, err = tree.GetKeyValuePairUnsafe(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, KeyNotFoundError{}, err, "Expected KeyNotFoundError, but got %v", err)

				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
			}
		})
	}

}

func TestBuildTreeAndGetKeyValuePair(t *testing.T) {
	config1bit, config2bits, config3bits := getTreeCfgsWith1_2_3BitsPerIndex(t)
	kvps1_1bit, kvps2_1bit, kvps3_1bit := getSampleKVPS1bit()
	kvps1_3bits, kvps2_3bits, kvps3_3bits := getSampleKVPS3bits()

	tests := []struct {
		cfg   TreeConfig
		kvps1 []KeyValuePair
		kvps2 []KeyValuePair
		kvps3 []KeyValuePair
	}{
		{config1bit, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config2bits, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config3bits, kvps1_3bits, kvps2_3bits, kvps3_3bits},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v bits tree", test.cfg.bitsPerIndex), func(t *testing.T) {
			tree := NewTree(test.cfg, &InMemoryStorageEngine{}, logger.NewTestLogger(t))

			err := tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps1)
			require.NoError(t, err)

			for _, kvp := range test.kvps1 {
				_, _, err := tree.GetKeyValuePairUnsafe(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, err := tree.GetKeyValuePairUnsafe(context.TODO(), nil, 7, kvp.Key)
				require.NoError(t, err, "Unexpected error for key %v: %v", kvp.Key, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, err = tree.GetKeyValuePair(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err = tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

			}

			err = tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps2)
			require.NoError(t, err)

			for i, kvp := range test.kvps2 {
				_, _, err := tree.GetKeyValuePairUnsafe(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, err := tree.GetKeyValuePairUnsafe(context.TODO(), nil, 7, kvp.Key)
				require.NoError(t, err, "Unexpected error for key %v: %v", kvp.Key, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, err = tree.GetKeyValuePair(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, err = tree.GetKeyValuePair(context.TODO(), nil, 2, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err = tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

				kvpRet, kvpHashRet, err = tree.GetKeyValuePair(context.TODO(), nil, 2, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err = tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

				kvpRet, kvpHashRet, err = tree.GetKeyValuePair(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, test.kvps1[i].Value, kvpRet.Value)
				expHash, err = tree.EncodeAndHash(test.kvps1[i])
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

			}

		})
	}

}

func TestBuildTreeAndGetKeyValuePairWithProof(t *testing.T) {
	config1bit, config2bits, config3bits := getTreeCfgsWith1_2_3BitsPerIndex(t)
	kvps1_1bit, kvps2_1bit, kvps3_1bit := getSampleKVPS1bit()
	kvps1_3bits, kvps2_3bits, kvps3_3bits := getSampleKVPS3bits()

	tests := []struct {
		cfg   TreeConfig
		kvps1 []KeyValuePair
		kvps2 []KeyValuePair
		kvps3 []KeyValuePair
	}{
		{config1bit, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config2bits, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config3bits, kvps1_3bits, kvps2_3bits, kvps3_3bits},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v bits tree", test.cfg.bitsPerIndex), func(t *testing.T) {
			tree := NewTree(test.cfg, &InMemoryStorageEngine{}, logger.NewTestLogger(t))

			err := tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps1)
			require.NoError(t, err)

			for _, kvp := range test.kvps1 {
				_, _, _, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, _, err = tree.GetKeyValuePairWithProof(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, _, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))

			}

			err = tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps2)
			require.NoError(t, err)

			for i, kvp := range test.kvps2 {
				_, _, _, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 0, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				_, _, _, err = tree.GetKeyValuePairWithProof(context.TODO(), nil, 7, kvp.Key)
				require.Error(t, err)
				require.IsType(t, InvalidSeqnoError{}, err, "Expected InvalidSeqnoError, but got %v", err)
				kvpRet, kvpHashRet, _, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 2, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, test.kvps2[i].Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(test.kvps2[i])
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))
			}

		})
	}

}

func TestHonestMerkleProofsVerifySuccesfully(t *testing.T) {
	config1bit, config2bits, config3bits := getTreeCfgsWith1_2_3BitsPerIndex(t)
	kvps1_1bit, kvps2_1bit, kvps3_1bit := getSampleKVPS1bit()
	kvps1_3bits, kvps2_3bits, kvps3_3bits := getSampleKVPS3bits()

	config3bits2valsPerLeaf, err := NewConfig(IdentityHasher{}, 3, 2, 3)
	require.NoError(t, err)

	tests := []struct {
		cfg   TreeConfig
		kvps1 []KeyValuePair
		kvps2 []KeyValuePair
		kvps3 []KeyValuePair
	}{
		{config1bit, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config2bits, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config3bits, kvps1_3bits, kvps2_3bits, kvps3_3bits},
		{config3bits2valsPerLeaf, kvps1_3bits, kvps2_3bits, kvps3_3bits},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v bits %v values per leaf tree", test.cfg.bitsPerIndex, test.cfg.maxValuesPerLeaf), func(t *testing.T) {
			tree := NewTree(test.cfg, &InMemoryStorageEngine{}, logger.NewTestLogger(t))
			verifier := MerkleProofVerifier{TreeConfig: test.cfg}

			err := tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps1)
			require.NoError(t, err)

			_, _, rootHash1, err := tree.GetLatestRoot(context.TODO(), nil)
			require.NoError(t, err)

			for _, kvp := range test.kvps1 {
				kvpRet, kvpHashRet, proof, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.True(t, kvp.Key.Equal(kvpRet.Key))
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))
				require.NoError(t, verifier.VerifyInclusionProof(context.TODO(), kvp, proof, rootHash1))
			}

			err = tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps2)
			require.NoError(t, err)
			_, _, rootHash2, err := tree.GetLatestRoot(context.TODO(), nil)
			require.NoError(t, err)

			for i, kvp := range test.kvps2 {
				kvpRet, kvpHashRet, proof, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 2, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))
				require.NoError(t, verifier.VerifyInclusionProof(context.TODO(), kvp, proof, rootHash2))

				kvpRet, kvpHashRet, proof, err = tree.GetKeyValuePairWithProof(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, test.kvps1[i].Value, kvpRet.Value)
				expHash, err = tree.EncodeAndHash(test.kvps1[i])
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))
				require.NoError(t, verifier.VerifyInclusionProof(context.TODO(), test.kvps1[i], proof, rootHash1))
			}

		})
	}

}

func TestSomeMaliciousProofsFail(t *testing.T) {
	config1bit, config2bits, config3bits := getTreeCfgsWith1_2_3BitsPerIndex(t)
	kvps1_1bit, kvps2_1bit, kvps3_1bit := getSampleKVPS1bit()
	kvps1_3bits, kvps2_3bits, kvps3_3bits := getSampleKVPS3bits()

	config3bits2leaves, err := NewConfig(SHA512Hasher{}, 3, 2, 3)
	require.NoError(t, err)

	tests := []struct {
		cfg   TreeConfig
		kvps1 []KeyValuePair
		kvps2 []KeyValuePair
		kvps3 []KeyValuePair
	}{
		{config1bit, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config2bits, kvps1_1bit, kvps2_1bit, kvps3_1bit},
		{config3bits, kvps1_3bits, kvps2_3bits, kvps3_3bits},
		{config3bits2leaves, kvps1_3bits, kvps2_3bits, kvps3_3bits},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v bits tree", test.cfg.bitsPerIndex), func(t *testing.T) {
			tree := NewTree(test.cfg, &InMemoryStorageEngine{}, logger.NewTestLogger(t))
			verifier := MerkleProofVerifier{TreeConfig: test.cfg}

			err := tree.BuildNewTreeVersionFromAllKeys(context.TODO(), test.kvps1)
			require.NoError(t, err)

			_, _, rootHash1, err := tree.GetLatestRoot(context.TODO(), nil)
			require.NoError(t, err)

			for _, kvp := range test.kvps1 {
				// First, check honest proofs pass
				kvpRet, kvpHashRet, proof, err := tree.GetKeyValuePairWithProof(context.TODO(), nil, 1, kvp.Key)
				require.NoError(t, err)
				require.Equal(t, kvp.Value, kvpRet.Value)
				expHash, err := tree.EncodeAndHash(kvp)
				require.NoError(t, err)
				require.True(t, kvpHashRet.Equal(expHash))
				require.NoError(t, verifier.VerifyInclusionProof(context.TODO(), kvp, proof, rootHash1))

				// Change the value
				kvpFakeVal := KeyValuePair{Key: kvp.Key, Value: "ALTERED_VALUE"}
				err = verifier.VerifyInclusionProof(context.TODO(), kvpFakeVal, proof, rootHash1)
				require.Error(t, err)
				require.IsType(t, ProofVerificationFailedError{}, err)

				// Change the key
				keyFake := Key(append([]byte(nil), ([]byte(kvp.Key))...))
				([]byte(keyFake))[0] = 1 + ([]byte(keyFake))[0]
				kvpFakeKey := KeyValuePair{Key: keyFake, Value: kvp.Value}
				err = verifier.VerifyInclusionProof(context.TODO(), kvpFakeKey, proof, rootHash1)
				require.Error(t, err)
				require.IsType(t, ProofVerificationFailedError{}, err)

				// Change the root hash
				rootHashFake := Hash(append([]byte(nil), ([]byte(rootHash1))...))
				([]byte(rootHashFake))[0] = 1 + ([]byte(rootHashFake))[0]
				err = verifier.VerifyInclusionProof(context.TODO(), kvp, proof, rootHashFake)
				require.Error(t, err)
				require.IsType(t, ProofVerificationFailedError{}, err)

				// nil root hash
				err = verifier.VerifyInclusionProof(context.TODO(), kvp, proof, nil)
				require.Error(t, err)
				require.IsType(t, ProofVerificationFailedError{}, err)

				// empty proof
				err = verifier.VerifyInclusionProof(context.TODO(), kvp, MerkleInclusionProof{}, rootHash1)
				require.Error(t, err)
				require.IsType(t, ProofVerificationFailedError{}, err)
			}
		})
	}

}
