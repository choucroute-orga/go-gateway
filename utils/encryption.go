package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"

	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("context", "utils/encryption")

func GenerateSecretKey() (string, error) {
	l := logger.WithField("request", "generateSecretKey")
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		l.WithError(err).Error("Error generating key")
		return "", err
	}

	key := string(secret[:])
	return key, nil
}

func encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func Encrypt(plaintext, secretKey string) (string, error) {
	l := logger.WithField("request", "encrypt")
	aes, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		l.WithError(err).Error("Error creating cipher")
		return "", err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		l.WithError(err).Error("Error creating GHASH cipher")
		return "", err
	}

	// We need a 12-byte nonce for GCM (modifiable if you use cipher.NewGCMWithNonceSize())
	// A nonce should always be randomly generated for every encryption.
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		l.WithError(err).Error("Error generating nonce")
		return "", err
	}

	// ciphertext here is actually nonce+ciphertext
	// So that when we decrypt, just knowing the nonce size
	// is enough to separate it from the ciphertext.
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return encode(ciphertext), nil
}

func Decrypt(ciphertext, secretKey string) (string, error) {
	l := logger.WithField("request", "decrypt")
	aes, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		l.WithError(err).Error("Error creating cipher")
		return "", err
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		l.WithError(err).Error("Error creating GHASH cipher")
		return "", err
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	nonceSize := gcm.NonceSize()
	cypherTextBytes, err := decode(ciphertext)
	if err != nil {
		l.WithError(err).Error("Error decoding ciphertext")
		return "", err
	}
	ciphertext = string(cypherTextBytes)
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		l.WithError(err).Error("Error decrypting ciphertext")
		return "", err
	}

	return string(plaintext), nil
}

/*
func main() {
	secretKey, err := GenerateSecretKey()
	if err != nil {
		panic(err)
	}

	// This will successfully encrypt & decrypt
	ciphertext1, _ := encrypt("This is some sensitive information", secretKey)
	fmt.Printf("Encrypted ciphertext 1: %x \n", ciphertext1)

	plaintext1, _ := decrypt(ciphertext1, secretKey)
	fmt.Printf("Decrypted plaintext 1: %s \n", plaintext1)

	// GEnerate a new secret key
	secretKey2, err := GenerateSecretKey()
	if err != nil {
		panic(err)
	}

	// This will successfully encrypt & decrypt as well.
	ciphertext2, _ := encrypt("Hello", secretKey2)
	fmt.Printf("Encrypted ciphertext 2: %x \n", ciphertext2)

	plaintext2, _ := decrypt(ciphertext2, secretKey2)
	fmt.Printf("Decrypted plaintext 2: %s \n", plaintext2)

	// Try to decrypt ciphertext1 with secretKey2
	// This will panic because the secret key is different
	// and the decryption will fail.
	plaintext3, err := decrypt(ciphertext1, secretKey2)
	if err != nil {
		fmt.Println("Error decrypting ciphertext1 with secretKey2")
	}
	if plaintext3 == "" {
		fmt.Println("Decryption failed")
	}

}
*/
