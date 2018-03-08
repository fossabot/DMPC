package core

import (
	"crypto/rand"
	"crypto/rsa"
	"reflect"
	"testing"
)

/*
	Test helpers
*/

func generatePrivateKey() *rsa.PrivateKey {
	priv, _ := rsa.GenerateKey(rand.Reader, AsymmetricKeySizeBits)
	return priv
}

func generateRandomBytes(nbBytes int) (bytes []byte) {
	bytes = make([]byte, nbBytes)
	rand.Read(bytes)
	return
}

func generateTemporaryEncryptedOperation(
	encrypted bool,
	challenges map[string]string,
	nonce []byte,
	payload []byte,
) *TemporaryEncryptedOperation {
	return &TemporaryEncryptedOperation{
		Version: 0.1,
		Encryption: TemporaryEncryptionFields{
			Encrypted:  encrypted,
			Challenges: challenges,
			Nonce:      Base64EncodeToString(nonce),
		},
		Payload: Base64EncodeToString(payload),
	}
}

func generateTemporaryEncryptedOperationWithEncryption(
	plainPayload []byte,
) (*TemporaryEncryptedOperation, *rsa.PrivateKey) {
	// Make temporary key and nonce
	temporaryNonce := generateRandomBytes(SymmetricNonceSize)
	temporaryKey := generateRandomBytes(SymmetricKeySize)

	// Encrypt challenge string and payload using temporary symmetric key
	aead, _ := NewAead(temporaryKey)
	payloadCiphertext := SymmetricEncrypt(
		aead,
		[]byte{},
		temporaryNonce,
		plainPayload,
	)
	challengeCiphertext := SymmetricEncrypt(
		aead,
		[]byte{},
		temporaryNonce,
		[]byte(correctChallenge),
	)

	// Make RSA key and use it to encrypt temporary key
	recipientKey := generatePrivateKey()
	symKeyEncrypted, _ := AsymmetricEncrypt(&recipientKey.PublicKey, temporaryKey[:])

	// Make challenges map
	challengeCiphertextBase64 := Base64EncodeToString(challengeCiphertext)
	symKeyEncryptedBase64 := Base64EncodeToString(symKeyEncrypted)
	challenges := map[string]string{
		"random":              "test",
		symKeyEncryptedBase64: challengeCiphertextBase64,
		"random2":             "test2",
	}

	return generateTemporaryEncryptedOperation(
		true,
		challenges,
		temporaryNonce,
		payloadCiphertext,
	), recipientKey
}

func generatePermanentEncryptedOperation(
	encrypted bool,
	keyId string,
	nonce []byte,
	issuerSignature []byte,
	certifierSignature []byte,
	requestType int,
	payload []byte,
) *PermanentEncryptedOperation {
	return &PermanentEncryptedOperation{
		Encryption: PermanentEncryptionFields{
			Encrypted: encrypted,
			KeyId:     keyId,
			Nonce:     Base64EncodeToString(nonce),
		},
		Issue: PermanentAuthenticationFields{
			Signature: Base64EncodeToString(issuerSignature),
		},
		Certification: PermanentAuthenticationFields{
			Signature: Base64EncodeToString(certifierSignature),
		},
		Meta: PermanentMetaFields{
			RequestType: requestType,
		},
		Payload: Base64EncodeToString(payload),
	}
}

func generatePermanentEncryptedOperationWithEncryption(
	keyId string,
	permanentKey []byte,
	requestType int,
	plainPayload []byte,
) *PermanentEncryptedOperation {
	// Encrypt payload with symmetric permanent key
	permanentNonce := generateRandomBytes(SymmetricNonceSize)
	aead, _ := NewAead(permanentKey)
	ciphertextPayload := SymmetricEncrypt(
		aead,
		[]byte{},
		permanentNonce,
		plainPayload,
	)

	// Hash and sign plaintext payload with new RSA keys
	plainPayloadHashed := Hash(plainPayload)
	issuerKey := generatePrivateKey()
	certifierKey := generatePrivateKey()
	issuerSignature, _ := Sign(issuerKey, plainPayloadHashed[:])
	certifierSignature, _ := Sign(certifierKey, plainPayloadHashed[:])

	return generatePermanentEncryptedOperation(
		true,
		keyId,
		permanentNonce,
		issuerSignature,
		certifierSignature,
		requestType,
		ciphertextPayload,
	)
}

/*
	Temporary decryption
*/

func TestValidOperation(t *testing.T) {
	// Make encrypted
	encryptedInnerOperation := generatePermanentEncryptedOperationWithEncryption(
		"KEY_ID",
		generateRandomBytes(SymmetricKeySize),
		1,
		[]byte("REQUEST_PAYLOAD"),
	)
	innerOperationJson, _ := encryptedInnerOperation.Encode()
	temporaryEncryptedOperation, recipientKey := generateTemporaryEncryptedOperationWithEncryption(
		innerOperationJson,
	)

	decryptedTemporaryEncryptedOperation, err := temporaryEncryptedOperation.Decrypt(recipientKey)
	if err != nil ||
		!reflect.DeepEqual(encryptedInnerOperation, decryptedTemporaryEncryptedOperation) {
		t.Errorf("Temporary decryption failed.")
		t.Errorf("encryptedInnerOperation=%v", encryptedInnerOperation)
		t.Errorf("decryptedTemporaryEncryptedOperation=%v", decryptedTemporaryEncryptedOperation)
		t.Errorf("err=%v", err)
	}
}
