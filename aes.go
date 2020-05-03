package cryptopals

import (
	"bytes"
	"crypto/aes"
	"fmt"
	"log"
	"math/rand"
)

func encryptAESECB(plainText []byte, key []byte, blockSize int) ([]byte, error) {
	cipherText := make([]byte, len(plainText))
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AES: %w", err)
	}

	if len(plainText)%blockSize > 0 {
		log.Printf("WARN: plainText (%d) is not a multiple of blockSize (%d)", len(plainText), blockSize)
	}

	for i := 0; i < (len(plainText) / blockSize); i++ {
		start := i * blockSize
		end := (i + 1) * blockSize
		cipher.Encrypt(cipherText[start:end], plainText[start:end])
	}

	return cipherText, nil
}

func decryptAESECB(cipherText []byte, key []byte, blockSize int) ([]byte, error) {
	plainText := make([]byte, len(cipherText))
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AES: %w", err)
	}

	for i := 0; i < (len(plainText) / blockSize); i++ {
		start := i * blockSize
		end := (i + 1) * blockSize
		cipher.Decrypt(plainText[start:end], cipherText[start:end])
	}

	return plainText, nil
}

func encryptAESCBC(plainText []byte, key []byte, iv []byte) ([]byte, error) {
	blockSize := 16
	if len(key) < blockSize {
		return nil, fmt.Errorf("key size must be %d", blockSize)
	}

	if len(iv) < blockSize {
		return nil, fmt.Errorf("iv size must be %d", blockSize)
	}

	cipherText := make([]byte, len(plainText))
	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AES: %w", err)
	}

	buffer := make([]byte, blockSize)
	lastCipherText := make([]byte, blockSize)
	copy(lastCipherText, iv)
	for i := 0; i < (len(plainText) / blockSize); i++ {
		start := i * blockSize
		end := (i + 1) * blockSize

		for j := 0; j < blockSize; j++ {
			buffer[j] = lastCipherText[j] ^ plainText[start:end][j]
		}

		cipher.Encrypt(lastCipherText, buffer)
		copy(cipherText[start:end], lastCipherText)
	}

	return cipherText, nil
}

func decryptAESCBC(cipherText []byte, key []byte, iv []byte) ([]byte, error) {
	blockSize := 16
	if len(key) < blockSize {
		return nil, fmt.Errorf("key size must be %d", blockSize)
	}

	if len(iv) < blockSize {
		return nil, fmt.Errorf("iv size must be %d", blockSize)
	}

	cipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("could not initialize AES: %w", err)
	}

	plainText := make([]byte, len(cipherText))
	buffer := make([]byte, blockSize)
	plainTextBuffer := make([]byte, blockSize)
	lastCipherText := make([]byte, blockSize)
	copy(lastCipherText, iv)
	for i := 0; i < (len(plainText) / blockSize); i++ {
		start := i * blockSize
		end := (i + 1) * blockSize

		cipher.Decrypt(buffer, cipherText[start:end])
		for j := 0; j < blockSize; j++ {
			plainTextBuffer[j] = lastCipherText[j] ^ buffer[j]
		}

		copy(plainText[start:end], plainTextBuffer)
		copy(lastCipherText, cipherText[start:end])
	}

	return plainText, nil
}

// Encrypts plainText under an unknown key, using ECB 50% of the time
// and CBC (with a random IV) 50% of the time (randomly.)
func encryptAESRandom(plainText []byte) ([]byte, error) {
	key := make([]byte, 16)
	_, err := rand.Read(key)

	if err != nil {
		return nil, fmt.Errorf("Can't generate random key: %w", err)
	}

	iv := make([]byte, 16)
	_, err = rand.Read(iv)

	if err != nil {
		return nil, fmt.Errorf("Can't generate random IV: %w", err)
	}

	beforeData := make([]byte, rand.Intn(5)+5)
	_, err = rand.Read(beforeData)
	if err != nil {
		return nil, fmt.Errorf("Can't generate random prefix data: %w", err)
	}

	afterData := make([]byte, rand.Intn(5)+5)
	_, err = rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("Can't generate random suffix data: %w", err)
	}

	newPlainText := append(append(beforeData, plainText...), afterData...)

	toss := rand.Intn(2)
	var cipherText []byte

	if toss == 0 {
		fmt.Println("ECB")
		// Mode ECB
		cipherText, err = encryptAESECB(newPlainText, key, 16)
		if err != nil {
			return nil, fmt.Errorf("Could not perform ECB encryption: %w", err)
		}
	} else {
		fmt.Println("CBC")
		// Mode CBC
		cipherText, err = encryptAESCBC(newPlainText, key, iv)
		if err != nil {
			return nil, fmt.Errorf("Could not perform CBC encryption: %w", err)
		}
	}

	return cipherText, nil
}

type encryptor func([]byte) ([]byte, error)

// This function determines if "func f encryptor" below is an ECB encryptor, and
// returns the ECB block size, if true. Expects that encryptor f uses a stable key
// and pads input.
func detectAESECB(f encryptor) (bool, int, error) {
	plainText := []byte("0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDE")
	cipherText, err := f(plainText)
	if err != nil {
		return false, 0, fmt.Errorf("could not encrypt: %w", err)
	}

	length := len(cipherText)

	newPlainText := []byte(plainText)
	for blockSize := 1; blockSize <= 64; blockSize++ {
		newPlainText = append([]byte("A"), newPlainText...)
		newCipherText, err := f(newPlainText)
		if err != nil {
			return false, 0, fmt.Errorf("could not encrypt: %w", err)
		}

		if bytes.Equal(cipherText[length-blockSize:], newCipherText[len(newCipherText)-blockSize:]) {
			return true, blockSize, nil
		}
	}

	return false, 0, nil
}

// Perform byte-at-a-time cracking on ECB function "encrypt" up to
// maxLen bytes. maxLen must be a multiple of the block size.
func crackAESECB(encrypt encryptor, maxLen int) ([]byte, error) {
	// Crack ECB byte-at-a-time
	crackedSecret := []byte{}
	// Allocate enough room to crack up to maxLen bytes
	prefix := make([]byte, maxLen)

	// Crack secret one byte at a time and stop when no more plainText
	// can be recovered.
	for match := true; match; {
		match = false
		if len(crackedSecret) > maxLen {
			return crackedSecret, nil
		}

		// Prefix should be just one byte less than the length
		prefixLen := maxLen - (len(crackedSecret) % maxLen) - 1
		prefix = prefix[:prefixLen]

		// Encrypt data prefixed by 1-fewer byte than needed
		cipherPrefix, err := encrypt(prefix)
		if err != nil {
			return nil, fmt.Errorf("could not encrypt prefix: %w", err)
		}

		// Append what we've cracked so far
		prefix = append(prefix, crackedSecret...)

		// Lengthen prefix to make it maxLen bytes
		prefix = append(prefix, '\x00')
		for testByte := byte(0); testByte < 255; testByte++ {
			prefix[maxLen-1] = testByte
			cipherText, err := encrypt(prefix)
			if err != nil {
				return nil, fmt.Errorf("could not encrypt prefix: %w", err)
			}

			if bytes.Equal(cipherPrefix[:maxLen], cipherText[:maxLen]) {
				crackedSecret = append(crackedSecret, testByte)
				match = true
				break
			}
		}
	}

	return crackedSecret, nil
}
