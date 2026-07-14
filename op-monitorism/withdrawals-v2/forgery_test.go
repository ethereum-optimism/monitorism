package withdrawalsv2

import (
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals-v2/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// Real mainnet withdrawal proven at L1 block 25,520,219
// (tx 0x7d44ddbcde36ba9c81974e55e0e5fa8d2abdbe9d2d5839c46c09c41da0a16074),
// captured via debug_traceTransaction against an archive node. This is a genuine,
// portal-accepted proof, so a correct verifier MUST accept it — and any tampering
// with it (a forgery) MUST be rejected.
var (
	realWdHash    = common.HexToHash("0x6c4b24cb42c6bbd933b8ef5d29345cda96ded944d9f3b245adf119a9c5c703ac")
	realStateRoot = common.HexToHash("0xb2ba7e144e9688bf97f28d3ad6d13bd96779bb73be3d634cadf2f774d329d221")
	realMPSR      = common.HexToHash("0x56a533b14262680a2337c5cb2ae0d867a1fc6ee3999bf1c986cbd6bf55a90b99")
	realBlockHash = common.HexToHash("0x396309986f34e07084c96cd788a51d86ebaa574322589f7dcd69d15330d4d2ad")

	realWithdrawalProof = [][]byte{
		common.FromHex("0xf90211a075c3090d10f25958630bfbcbd700fe87816e1c7d8d403949c0316238a095d3bda0461b9d4a18b28fee0351759434931fc002dccd2654820f81380598ad7fcc8e0fa0fe5d697d7612d57debcfda15834fdd641f08c501997b331d0f4656896ff48859a0e1004c4eb53638edfbfb4375656091bb195c826610bf4860b25874f7b856d5dfa060212c58825d131b4102908edafc2cd5c941409c42e13c0d25205a9c3bc636c1a09764732371b630d6c84697d6f172db40925a3c5ad94e9e303e6c01cae8047ac2a0e1b0c88e10ac042c5c4f0ffc22e77e5ec99ff003969aa495f281632321ef6c67a0119b1691d4c0f595ac99e052622d672a5399dd8449ebc37860c292998eb8c4c3a0e7abaa70fa86de2e2e7904cd643c0d052f4bfd764fb5c7ba64eb9b2b8de080c8a025a8b300c83fb77baad1492800bc08a34675049992beff1ef257e797c17778dfa033d7a1be29d86afb535a54635dd41f0aeefbc1a35d9c3a44554b3ad7dafa7c17a029fb0d6d255749fda504e116177898a80d0f3cd9ba8b94dcfa0b43ae39e38c36a0df856e5fcda1d2c501adbd98ee74dbe898306f8b5918a56a4c86b8dfb9770320a0513d44966a983dcf52bdfef4029ef8c9fb7dc0072176970572ddd727a3e2dffda02b4ad59521b5922a309604f10a1cec63a2bd01d191d733c94e0a0dd357f9bcb5a0af2c53f5220ea2feadc2bf6fb4ddda24c34feb72bd3a25f8ce1fe69816c1c94d80"),
		common.FromHex("0xf90211a081b4cad14f1ca191940566ebf3ac09273f8e09ba4aad40af7cce421a6d907bbaa056b8a722dc30ce0bc7c6bb80ef1e52af52a3f80968e5ab82716e0a4c746b838ba0673fcc5ff670d0842447d19f90f9e52776c5b9116bd429d55cf2f26e5ed78e72a0bc74ee01b8c7682de3bb927ccba2e05671691ae064d35a31813b4882866f94e0a0da7f97866fb0f4710942069bb9eb942feb9b465ed0037b9630b841b248e3df13a0bd79742faf4e49c756cabfa71d62f4db950c5969398574f0017b62aa5cc7edcea0d09bdba5a3f21c65179583e3ee67d0c166c6f97c55ddc8758f24cf8f719dd2e5a0a218398ec2076dd94f095b51b084855d75dca2d1d82222a10949e0323c38c65ea0a83689d2ad7e8edcb3e5156447f91ee3336378278db4d0883594dc1f158c85e5a04647b1d9b0e495cd230f2b4af7f660e9a5db42f6c334e11b4a8c89e7f228ac2ba09cbee4101b0d744be06468cf3ca957dfa2889b8f00ebf99c13261b1c638994b7a08f72e31df60c815aeb6c5d5b9982e12f1f5c2f68f1d7e33f2806d770a3b19eaaa07b9cc51d072a77d8931284de735953409a3e6451c33076111c5ffe139695729ba0b652761e8d443666c0cfe0921f684461601efa8415621754899568ee1bb22b15a08bdd56ea407f8f89834382318d0cb5fe40ba9d052e1955fe5a9265b0600dbaa7a0622939ef79500fdf40aeaa06884edfed4fa4221e7544d1ce1908a0fe038e08f880"),
		common.FromHex("0xf90211a0a2f7fe9d1a66f587a6897dd036bc432e9f77da46c02cfab060ee92b782d55477a06966b4af611012c945ef58ed5042cd2f77054768e22c7f85720beec41581b5e8a0ef5192938277e788dcd22eba0a539aa2b10514dd86c26efb670ef31b1d8dc3cca0851423f5ddba14caeedc248f020b58241cd288ed7597417eb116921402b039c6a038a8e76c9bb72ce14ff28521818838f00e06aff8e14059b4d934ea3240161af8a0e442ed139239ae4ecf7d62d41885789e719c1b568385f6572bef1af51bdcceb8a03797ddee9bed8d10dc685504dc6f2acd14f6061db3f3a4e86214a17512dc1967a08172519e2b2380c54a11b283c67de842e9895eac5f8170fb8381040eaf25cea2a0208fc763bf2894bd345521c3b0d30d33b8a2a1657b1ad65ac70ab1edd0a6946fa0a63c3a8145bf395df49a5ec2c1d3a3577af73f59fe9c380118b8a0c127fa5373a0eb1dc53c626ab2c4d38bb2817db0c6afabcd1ad6acdf3fd8becbd05e1b86f9f7a0239788b294e8d8bfb02155214d84515505baba4274da9f7fcdc801736353faa6a004f78c3cda6ecb7a44e4d685e36cfc0bba3a78d72ce030009a26ea69f6e1d8c5a05fb6161c572bd84badb6b1124db59d691e8d462bb0b4c0bffe6f8191cf3a940aa0291503c5998cd0f4be679503db1c83ab6de1a318e778bcecdccf8d2ce2859135a083b229b2d1a7f972f11a85d247be6317d4b5022b2be4fd82fef723d03321d61d80"),
		common.FromHex("0xf901b1a0444a9578abf2501101d78bb9637a3625b3f2efeca34943ade72906c7bc0fb964a0710615c5d5fe1542d6a67a5081d39db581bfa1bc7fd6f75351f5b785c699da808080a05a715533892fff2b5089d3f4c321bd7edf3e1f0079bb47e6d047bf681ec9b97ea0f4b7e954b2c718e4678472d6510a2bc5fe14fa1a5a47a561940b1b698f75210fa0b6e56d34e7088dd901cc793af38b0c879627242e672c809fe7eeece964682f29a00ff81e038237937dc6ca5944abb3ae4a6242dd0b28294de6568d45fbfd092d79a03ee4b1820bc857be2998b230f3411c218ffa618497e701253a6fc15e0dbb3a91a02d71a912e6d71a3d0d76c9511051f6bdbdc6e729417be103c9d89ea4abdf1e94a0c9c42bc9745ee342e28a32045e55968eb71a848197f0eaba2e87e8e4a0bffd8ea0fee177f6cf1afb19d6078f6604c507a4b832f09e3814ded6220cd4fdccbed768a019d788b75b2a02656f870f8f22654069b5cf8d6edffd4d972b95b9f6ec1e383aa0ae8a05783006fe9595041f03bef22811fe7f489582c5e5e1f2a9cc7a99c8471e80a075f4fed404578d2717bf9d0e78528a3cd11cbd6dce8b9da29393d45673b6310b80"),
		common.FromHex("0xf851808080808080a0ad5346d167d94ba9d5e1e3cf58180f4d8e8c5cee47ab562353adf11936fcd7718080808080a0e946df05aeccb5e96494164511ace5bc7896ac15f46ab0f291b7665ac8233c2f80808080"),
		common.FromHex("0xe09e33a29cc25676665dc7846dd15416ccfaba5f2d99e39e0a4e84584e28879f01"),
	}
)

// realProof builds the decodedProof + matching root claim from the captured fixture.
func realProof() (*decodedProof, [32]byte) {
	orp := bindings.TypesOutputRootProof{
		Version:                  [32]byte{},
		StateRoot:                realStateRoot,
		MessagePasserStorageRoot: realMPSR,
		LatestBlockhash:          realBlockHash,
	}
	rootClaim := crypto.Keccak256Hash(
		orp.Version[:], orp.StateRoot[:], orp.MessagePasserStorageRoot[:], orp.LatestBlockhash[:],
	)
	proof := make([][]byte, len(realWithdrawalProof))
	for i, n := range realWithdrawalProof {
		proof[i] = append([]byte(nil), n...)
	}
	return &decodedProof{outputRootProof: orp, withdrawalProof: proof}, [32]byte(rootClaim)
}

// TestForgeryDetection uses a real, portal-accepted mainnet proof and confirms that
// (1) a correct verifier accepts it, and (2) every way of forging it is caught as P0.
func TestForgeryDetection(t *testing.T) {
	t.Run("genuine proof re-verifies", func(t *testing.T) {
		dp, rootClaim := realProof()
		assert.Equal(t, "", verifyProof([32]byte(realWdHash), rootClaim, dp),
			"a real portal-accepted proof must verify")
	})

	t.Run("FORGERY: corrupted storage-proof leaf -> P0", func(t *testing.T) {
		dp, rootClaim := realProof()
		// Flip the last byte of the leaf node (the sentinel value / value hash).
		leaf := dp.withdrawalProof[len(dp.withdrawalProof)-1]
		leaf[len(leaf)-1] ^= 0xff
		assert.Equal(t, reasonBadWithdrawalProof, verifyProof([32]byte(realWdHash), rootClaim, dp),
			"a tampered storage proof must be rejected as a bad withdrawal proof (P0)")
	})

	t.Run("FORGERY: corrupted interior proof node -> P0", func(t *testing.T) {
		dp, rootClaim := realProof()
		dp.withdrawalProof[0][10] ^= 0xff
		assert.Equal(t, reasonBadWithdrawalProof, verifyProof([32]byte(realWdHash), rootClaim, dp))
	})

	t.Run("FORGERY: withdrawal hash not in the trie -> P0", func(t *testing.T) {
		dp, rootClaim := realProof()
		var otherWd [32]byte
		copy(otherWd[:], realWdHash.Bytes())
		otherWd[0] ^= 0xff // a different (never-initiated) withdrawal hash
		assert.Equal(t, reasonBadWithdrawalProof, verifyProof(otherWd, rootClaim, dp),
			"proving a hash absent from the trie must be rejected (P0)")
	})

	t.Run("FORGERY: output-root proof not bound to the game -> P0", func(t *testing.T) {
		dp, rootClaim := realProof()
		rootClaim[0] ^= 0xff // pretend the game's root claim differs
		assert.Equal(t, reasonBadOutputRootBinding, verifyProof([32]byte(realWdHash), rootClaim, dp),
			"an output-root proof that doesn't hash to the game root must be rejected (P0)")
	})
}
