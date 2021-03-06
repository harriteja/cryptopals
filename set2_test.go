package cryptopals

/*
## Cryptopals Solutions by Mohit Muthanna Cheppudira 2020.

This file consists of solutions to Set 2. Run with:

  $ go test -v --run S2
*/

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestS2C9(t *testing.T) {
	plainText := "YELLOW SUBMARINE"
	want := "YELLOW SUBMARINE\x04\x04\x04\x04"

	paddedText, err := padPKCS7(plainText, 20)
	assertNoError(t, err)
	assertEquals(t, want, paddedText)
}

func TestS2C10(t *testing.T) {
	plainText := []byte("YELLOW SUBMARINEYELLOW SUBMARINEYELLOW SUBMARINEYELLOW SUBMARINE")
	key := []byte("1234567890ABCDEF")
	iv := make([]byte, 16)

	cipherText, err := encryptAESCBC(plainText, key, iv)
	assertNoError(t, err)

	newPlainText, err := decryptAESCBC(cipherText, key, iv)
	assertNoError(t, err)
	assertEquals(t, string(plainText), string(newPlainText))

	data, err := ioutil.ReadFile("data/10.txt")
	assertNoError(t, err)

	cipherText, err = base64.StdEncoding.DecodeString(string(data))
	plainText, err = decryptAESCBC(cipherText, []byte("YELLOW SUBMARINE"), iv)
	assertNoError(t, err)

	trimmedPlaintext := strings.Trim(string(plainText), "\x04\n ")
	re := regexp.MustCompile(`Play that funky music$`)
	assertEquals(t, true, re.MatchString(trimmedPlaintext))
}

func TestECBEncryptDecrypt(t *testing.T) {
	plainText := make([]byte, 16*100)
	_, err := rand.Read(plainText)
	assertNoError(t, err)
	key := make([]byte, 16)
	_, err = rand.Read(key)
	assertNoError(t, err)

	cipherText, err := encryptAESECB(plainText, key, 16)
	assertNoError(t, err)

	newPlainText, err := decryptAESECB(cipherText, key, 16)
	assertNoError(t, err)

	assertEquals(t, true, bytes.Equal(plainText, newPlainText))
}

func TestS2C11(t *testing.T) {
	detectBlockSize := func(data []byte) (int, error) {
		bestBlockSize := 0

		for i := 4; i <= 40; i++ {
			distance, err := numSimilarBlocks(data, i, 4)
			fmt.Println(i, distance)
			if err != nil {
				return 0, fmt.Errorf("could not calculate block distance: %w", err)
			}

			// Pick the largest block size with similar blocks. This is due to aliasing
			// effects of similarity. E.g., block size 16 with 1 similar block will have
			// block size 8 with 2 similar blocks.
			//
			// Fixme: use square for similar block size
			if distance > 0 {
				bestBlockSize = i
			}
		}

		return bestBlockSize, nil
	}

	rand.Seed(time.Now().UnixNano())
	plainText, err := ioutil.ReadFile("data/11.txt")
	assertNoError(t, err)

	cipherText, err := encryptAESRandom(plainText)
	assertNoError(t, err)

	similarity, err := numSimilarBlocks(cipherText, 16, 0)
	assertNoError(t, err)

	detectedBlockSize, err := detectBlockSize(cipherText)
	assertNoError(t, err)

	// If there are any similar blocks, then this is ECB
	if similarity > 0 {
		fmt.Println(similarity, "ECB", detectedBlockSize)
	} else {
		fmt.Println(similarity, "CBC", detectedBlockSize)
	}
}

func TestS2C12(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 16)
	_, err := rand.Read(key)
	assertNoError(t, err)

	secretData, err := ioutil.ReadFile("data/12.txt")
	assertNoError(t, err)

	secret, err := base64.StdEncoding.DecodeString(string(secretData))
	assertNoError(t, err)

	// Encryption oracle
	encrypt := func(plainText []byte) ([]byte, error) {
		newPlainText := append(plainText, secret...)
		newPlainText, err = padPKCS7ToBlockSize(newPlainText, 16)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 pad: %w", err)
		}

		cipherText, err := encryptAESECB(newPlainText, key, 16)
		if err != nil {
			return nil, fmt.Errorf("could not ECB encrypt: %w", err)
		}

		return cipherText, nil
	}

	isECB, blockSize, err := detectAESECB(encrypt)
	assertNoError(t, err)
	assertEquals(t, true, isECB)
	assertEquals(t, 16, blockSize)

	// Figure out length of secret
	cipherText, err := encrypt([]byte{})
	assertNoError(t, err)

	crackedSecret, err := crackAESECB(encrypt, len(cipherText))
	assertNoError(t, err)

	fmt.Println(string(crackedSecret))
	assertEquals(t, bytes.Equal(secret, crackedSecret[:len(secret)]), true)
}

// parseCookie extracts the "key=value&key=value" pairs from 'cookie' and
// returns a map. This is used by a number of challenges in set 2.
func parseCookie(cookie string) (map[string]string, error) {
	parts := strings.Split(cookie, "&")
	cookieMap := map[string]string{}
	for _, part := range parts {
		subParts := strings.Split(part, "=")
		if len(subParts) != 2 {
			return nil, fmt.Errorf("Invalid cookie part %s in %s", part, cookie)
		}

		cookieMap[strings.TrimSpace(subParts[0])] = subParts[1]
	}

	return cookieMap, nil
}

func TestCookieParsing(t *testing.T) {
	cookie, err := parseCookie("foo=bar&baz=qux&zap=zazzle")
	assertNoError(t, err)
	assertEquals(t, "qux", cookie["baz"])

	cookie, err = parseCookie("foo=bar&baz&=qux&zap=zazzle")
	assertHasError(t, err)

	cookie, err = parseCookie("foo =bar&baz=qux&zap=zazzle")
	assertNoError(t, err)
	assertEquals(t, "bar", cookie["foo"])
}

func TestS2C13(t *testing.T) {
	// Remove cookie control characters '&' and '&' from val.
	sanitizeCookieValue := func(val string) string {
		sanitizedString := ""
		for _, c := range val {
			if c != '&' && c != '=' {
				sanitizedString += string(c)
			}
		}

		return sanitizedString
	}

	// Verify that sanitization works
	sanitized := sanitizeCookieValue("fo==obar&&boo=baz&hello")
	assertEquals(t, "foobarboobazhello", sanitized)

	// Encoodes cookie map into a string
	encodeCookie := func(cookie map[string]string, order []string) string {
		cookies := []string{}
		for _, k := range order {
			cookies = append(cookies, fmt.Sprintf("%s=%s", sanitizeCookieValue(k), sanitizeCookieValue(cookie[k])))
		}

		return strings.Join(cookies, "&")
	}

	// Returns the encoded profile for email
	profileFor := func(email string) string {
		profile := map[string]string{
			"email": email,
			"uid":   "10",
			"role":  "user",
		}

		return encodeCookie(profile, []string{"email", "uid", "role"})
	}
	assertEquals(t, "email=mo@mo.town&uid=10&role=user", profileFor("mo@mo.town"))

	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 16)
	_, err := rand.Read(key)
	assertNoError(t, err)

	encrypt := func(plainText []byte) ([]byte, error) {
		newPlainText, err := padPKCS7ToBlockSize(plainText, 16)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 pad: %w", err)
		}

		cipherText, err := encryptAESECB(newPlainText, key, 16)
		if err != nil {
			return nil, fmt.Errorf("could not ECB encrypt: %w", err)
		}

		return cipherText, nil
	}

	decrypt := func(cipherText []byte) ([]byte, error) {
		plainText, err := decryptAESECB(cipherText, key, 16)
		if err != nil {
			return nil, fmt.Errorf("could not ECB decrypt: %w", err)
		}

		newPlainText, err := unpadPKCS7(plainText)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 pad: %w", err)
		}

		return newPlainText, nil
	}

	// Verify that encrypt/decrypt works correctly
	plainText := []byte(profileFor("foo@bar.com"))
	cipherText, err := encrypt(plainText)
	assertNoError(t, err)

	newPlainText, err := decrypt(cipherText)
	assertNoError(t, err)
	assertTrue(t, bytes.Equal(plainText, newPlainText))

	// Verify that parseCookie works correctly
	profile, err := parseCookie(string(newPlainText))
	assertNoError(t, err)

	assertEquals(t, profile["email"], "foo@bar.com")
	assertEquals(t, profile["uid"], "10")
	assertEquals(t, profile["role"], "user")

	// Construct email to push role to new independent block
	plainText = []byte(profileFor("foo@bar.com012345678901234567"))
	cipherText, err = encrypt(plainText)
	assertNoError(t, err)
	assertEquals(t, 64, len(cipherText))

	// Verify that last block is just "user" (role)
	userRoleCipherText, err := encrypt([]byte("user"))
	assertNoError(t, err)
	assertTrue(t, bytes.Equal(userRoleCipherText, cipherText[48:]))

	// Create an encrypted block with the contents "admin"
	adminRoleCipherText, err := encrypt([]byte("admin"))
	assertNoError(t, err)

	// Finally, replace the "user" block with the "admin" block
	finalCipherText := append(cipherText[:48], adminRoleCipherText...)
	finalPlainText, err := decrypt(finalCipherText)

	// Verify that the final cipher text is a user with the admin role
	finalProfile, err := parseCookie(string(finalPlainText))
	assertNoError(t, err)
	fmt.Println(finalProfile)
	assertEquals(t, "admin", finalProfile["role"])
}

func TestS2C14(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 16)
	_, err := rand.Read(key)
	assertNoError(t, err)

	// Use the same target bytes as C12
	secretData, err := ioutil.ReadFile("data/12.txt")
	assertNoError(t, err)

	secret, err := base64.StdEncoding.DecodeString(string(secretData))
	assertNoError(t, err)

	numBytes := rand.Intn(10) + 1
	randomPrefix := make([]byte, numBytes)
	_, err = rand.Read(randomPrefix)
	assertNoError(t, err)

	// Encryption oracle
	encrypt := func(plainText []byte) ([]byte, error) {
		newPlainText := append(randomPrefix, append(plainText, secret...)...)
		newPlainText, err = padPKCS7ToBlockSize(newPlainText, 16)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 pad: %w", err)
		}

		cipherText, err := encryptAESECB(newPlainText, key, 16)
		if err != nil {
			return nil, fmt.Errorf("could not ECB encrypt: %w", err)
		}

		return cipherText, nil
	}

	isECB, blockSize, err := detectAESECB(encrypt)
	assertNoError(t, err)
	assertEquals(t, true, isECB)
	assertEquals(t, 16, blockSize)

	// Determine length of random text
	randomTextLen := 0
	plainText := make([]byte, 0, blockSize)
	cipherText, err := encrypt(plainText)
	assertNoError(t, err)
	for i := 1; i <= blockSize; i++ {
		plainText = append(plainText, '\x00')
		cipherBlock, err := encrypt(plainText)
		assertNoError(t, err)
		if bytes.Equal(cipherBlock[:blockSize], cipherText[:blockSize]) {
			randomTextLen = blockSize - (i - 1)
			break
		}

		cipherText = cipherBlock
	}

	spacing := blockSize - randomTextLen

	// Crack ECB byte-at-a-time
	crackedSecret, err := crackAESECB(encrypt, (blockSize*11)+spacing)
	assertNoError(t, err)
	fmt.Println(string(crackedSecret))
}

func TestS2C15(t *testing.T) {
	text, err := unpadPKCS7([]byte("ICE ICE BABY\x04\x04\x04\x04"))
	assertNoError(t, err)
	assertTrue(t, bytes.Equal([]byte("ICE ICE BABY"), text))

	text, err = unpadPKCS7([]byte("ICE ICE BABY\x05\x05\x05\x05"))
	assertHasError(t, err)

	text, err = unpadPKCS7([]byte("ICE ICE BABY\x01\x02\x03\x04"))
	assertHasError(t, err)

	text, err = unpadPKCS7([]byte("ICE ICE BABY\x03\x03\x03\x03"))
	assertHasError(t, err)

	// Should fail because PKCS7 requires that text of blockSize length have
	// an extra block added.
	text, err = unpadPKCS7([]byte("0123456789ABCDEF"))
	assertHasError(t, err)

	// Pad to verify that unpadding works correctly
	paddedText, err := padPKCS7ToBlockSize([]byte("0123456789ABCDEF"), 16)
	assertNoError(t, err)
	assertEquals(t, 32, len(paddedText))
	text, err = unpadPKCS7(paddedText)
	assertNoError(t, err)
}

func TestS2C16(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 16)
	_, err := rand.Read(key)
	assertNoError(t, err)

	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	assertNoError(t, err)

	encrypt := func(input []byte) ([]byte, error) {
		pre := []byte("comment1=cooking%20MCs;userdata=")
		post := []byte(";comment2=%20like%20a%20pound%20of%20bacon")

		sanitizedInput := []byte{}
		for _, c := range input {
			if c != ';' && c != '=' {
				sanitizedInput = append(sanitizedInput, c)
			}
		}

		plainText := append(pre, append(sanitizedInput, post...)...)
		plainText, err = padPKCS7ToBlockSize(plainText, 16)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 pad: %w", err)
		}

		cipherText, err := encryptAESCBC(plainText, key, iv)
		if err != nil {
			return nil, fmt.Errorf("could not CBC encrypt: %w", err)
		}

		return cipherText, nil
	}

	decrypt := func(cipherText []byte) ([]byte, error) {
		plainText, err := decryptAESCBC(cipherText, key, iv)
		if err != nil {
			return nil, fmt.Errorf("could not CBC decrypt: %w", err)
		}

		plainText, err = unpadPKCS7(plainText)
		if err != nil {
			return nil, fmt.Errorf("could not PKCS7 unpad: %w", err)
		}

		return plainText, nil
	}

	isCracked := func(cipherText []byte) bool {
		plainText, err := decrypt(cipherText)
		assertNoError(t, err)

		match, err := regexp.MatchString(";admin=true;", string(plainText))
		assertNoError(t, err)

		return match
	}

	cipherText, err := encrypt([]byte(";admin=true;"))
	assertNoError(t, err)
	assertFalse(t, isCracked(cipherText))

	zeroBlock := make([]byte, 16)
	adminBlock := []byte(";admin=true;")

	// Flips a single bit in data, indexed by byteIndex and bitIndex
	flipBit := func(data []byte, byteIndex int, bitIndex int) {
		data[byteIndex] ^= byte((1 << 7) >> bitIndex)
	}

	// Flip the bits for ; and = so that the sanitizer passes them through.
	flipBit(adminBlock, 0, 7)
	flipBit(adminBlock, 6, 7)
	flipBit(adminBlock, 11, 7)

	// Construct cipher text with zero block and our custom admin block. The idea
	// is to flip bits in the zero block, such that they are carried over to the
	// admin block.
	cipherText, err = encrypt(append(zeroBlock, adminBlock...))
	assertNoError(t, err)

	// Flip bits in the zero block of cipherText. This completely scrambles the zero block
	// but only propagates single-bit errors to the next block(s).
	flipBit(cipherText, 32, 7)
	flipBit(cipherText, 32+6, 7)
	flipBit(cipherText, 32+11, 7)

	// Verify that we now have ";admin=true" in our cipherText
	assertTrue(t, isCracked(cipherText))
}
