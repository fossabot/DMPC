/*
	Cryptography utilities
*/

package core

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"bytes"
)

func AsymKeyToString(key *rsa.PublicKey) string {
	// Break into bytes
	keyBytes, _ := x509.MarshalPKIXPublicKey(key)

	// Build pem block containing public key
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: keyBytes,
	}

	// PEM encode block
	buf := new(bytes.Buffer)
	pem.Encode(buf, block)

	// Return string representing bytes
	return string(pem.EncodeToMemory(block))
}

func StringToAsymKey(rsaString string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(rsaString))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.New("failed to parse DER encoded public key: " + err.Error())
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		return nil, errors.New("unknown type of public key" + err.Error())
	}
}

func GeneratePrivateKey() *rsa.PrivateKey {
	priv, _ := rsa.GenerateKey(rand.Reader, AsymmetricKeySizeBits)
	return priv
}

func GeneratePublicKey() *rsa.PublicKey {
	priv := GeneratePrivateKey()
	return &priv.PublicKey
}

func GenerateTemporaryEncryptedOperation(
	encrypted bool,
	challenges map[string]string,
	nonce []byte,
	nonceEncoded bool,
	payload []byte,
	payloadEncoded bool,
) *TemporaryEncryptedOperation {
	nonceResult := string(nonce)
	payloadResult := string(payload)
	if !nonceEncoded {
		nonceResult = Base64EncodeToString(nonce)
	}
	if !payloadEncoded {
		payloadResult = Base64EncodeToString(payload)
	}

	return &TemporaryEncryptedOperation{
		Version: 0.1,
		Encryption: TemporaryEncryptionFields{
			Encrypted:  encrypted,
			Challenges: challenges,
			Nonce:      nonceResult,
		},
		Payload: payloadResult,
	}
}

func GenerateTemporaryEncryptedOperationWithEncryption(
	plainPayload []byte,
	plaintextChallenge []byte,
	modifyChallenges func(map[string]string),
	recipientKey *rsa.PrivateKey,
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
		[]byte(plaintextChallenge),
	)

	// Make RSA key if nil and use it to encrypt temporary key
	if recipientKey == nil {
		recipientKey = generatePrivateKey()
	}
	symKeyEncrypted, _ := AsymmetricEncrypt(&recipientKey.PublicKey, temporaryKey[:])

	// Make challenges map
	challengeCiphertextBase64 := Base64EncodeToString(challengeCiphertext)
	symKeyEncryptedBase64 := Base64EncodeToString(symKeyEncrypted)
	challenges := map[string]string{
		symKeyEncryptedBase64: challengeCiphertextBase64,
	}
	modifyChallenges(challenges)

	return GenerateTemporaryEncryptedOperation(
		true,
		challenges,
		temporaryNonce,
		false,
		payloadCiphertext,
		false,
	), recipientKey
}

func GeneratePermanentEncryptedOperation(
	encrypted bool,
	keyId string,
	nonce []byte,
	nonceEncoded bool,
	issuerSignature []byte,
	issuerSignatureEncoded bool,
	certifierSignature []byte,
	certifierSignatureEncoded bool,
	requestType int,
	payload []byte,
	payloadEncoded bool,
) *PermanentEncryptedOperation {
	// Encode or convert to string
	nonceResult := string(nonce)
	issuerSignatureResult := string(issuerSignature)
	certifierSignatureResult := string(certifierSignature)
	payloadResult := string(payload)
	if !nonceEncoded {
		nonceResult = Base64EncodeToString(nonce)
	}
	if !issuerSignatureEncoded {
		issuerSignatureResult = Base64EncodeToString(issuerSignature)
	}
	if !certifierSignatureEncoded {
		certifierSignatureResult = Base64EncodeToString(certifierSignature)
	}
	if !payloadEncoded {
		payloadResult = Base64EncodeToString(payload)
	}

	// Create operation
	return &PermanentEncryptedOperation{
		Encryption: PermanentEncryptionFields{
			Encrypted: encrypted,
			KeyId:     keyId,
			Nonce:     nonceResult,
		},
		Issue: PermanentAuthenticationFields{
			Signature: issuerSignatureResult,
		},
		Certification: PermanentAuthenticationFields{
			Signature: certifierSignatureResult,
		},
		Meta: PermanentMetaFields{
			RequestType: requestType,
		},
		Payload: payloadResult,
	}
}

func GeneratePermanentEncryptedOperationWithEncryption(
	keyId string,
	permanentKey []byte,
	permanentNonce []byte,
	requestType int,
	plainPayload []byte,
	modifyIssuerSignature func([]byte) ([]byte, bool),
	modifyCertifierSignature func([]byte) ([]byte, bool),
) (*PermanentEncryptedOperation, *rsa.PrivateKey, *rsa.PrivateKey) {
	// Encrypt payload with symmetric permanent key
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
	issuerSignature, issuerSignatureEncoded := modifyIssuerSignature(issuerSignature)

	certifierSignature, _ := Sign(certifierKey, plainPayloadHashed[:])
	certifierSignature, certifierSignatureEncoded := modifyCertifierSignature(certifierSignature)

	return GeneratePermanentEncryptedOperation(
		true,
		keyId,
		permanentNonce,
		false,
		issuerSignature,
		issuerSignatureEncoded,
		certifierSignature,
		certifierSignatureEncoded,
		requestType,
		ciphertextPayload,
		false,
	), issuerKey, certifierKey
}