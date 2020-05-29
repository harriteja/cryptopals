package cryptopals

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

func decryptXORByte(data []byte, key byte) []byte {
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ key
	}

	return out
}

func getExpectedFreqForChar(char byte) float64 {
	value := float64(0.00001)

	freqMap := map[byte]float64{
		' ':  10,
		'\'': 0.1,
		'\n': 0.1,
		',':  0.1,
		'.':  0.1,
		'E':  12.02,
		'T':  9.1,
		'A':  8.12,
		'O':  7.68,
		'I':  7.31,
		'N':  6.95,
		'S':  6.28,
		'R':  6.02,
		'H':  5.92,
		'D':  4.32,
		'L':  3.98,
		'U':  2.88,
		'C':  2.71,
		'M':  2.61,
		'F':  2.3,
		'Y':  2.11,
		'W':  2.09,
		'G':  2.03,
		'P':  1.82,
		'B':  1.49,
		'V':  1.11,
		'K':  0.69,
		'X':  0.17,
		'Q':  0.11,
		'J':  0.10,
		'Z':  0.1,
		'0':  0.1,
		'1':  0.2,
		'2':  0.1,
		'3':  0.1,
		'4':  0.1,
		'5':  0.1,
		'6':  0.1,
		'7':  0.1,
		'8':  0.1,
		'9':  0.1,
	}

	if freq, ok := freqMap[strings.ToUpper(string(char))[0]]; ok {
		value = freq
	}

	return value
}

// Calculates the liklihood of str being an English string using chi-squared testing. Lower
// cost means higher liklihood.
func calcStringCost(str []byte) float64 {
	countMap := map[byte]int{}
	totalChars := len(str)

	for _, char := range str {
		key := strings.ToUpper(string(char))[0]
		if count, ok := countMap[key]; ok {
			countMap[key] = count + 1
		} else {
			countMap[key] = 1
		}
	}

	cost := float64(0)
	for k, v := range countMap {
		expectedCount := (getExpectedFreqForChar(k) / 100) * float64(totalChars)
		observedCount := float64(v)

		cost += math.Pow(expectedCount-observedCount, 2) / expectedCount
	}

	return cost
}

// Calculates the liklihood of str being an English string using correlation. Higher score
// means higher liklihood.
func calcStringScore(str []byte) float64 {
	score := float64(0)
	for _, char := range str {
		c := strings.ToUpper(string(char))[0]
		score += getExpectedFreqForChar(c)
	}

	return score
}

func crackXORByteCost(cipherText []byte) (key byte, cost float64, plainText string) {
	bestCost := float64(len(cipherText) * 100)
	var bestString string
	var bestKey byte
	for i := 0; i < 256; i++ {
		key := byte(i)
		plainText := decryptXORByte(cipherText, byte(key))
		cost := math.Sqrt(calcStringCost(plainText))

		if cost < bestCost {
			bestCost = cost
			bestString = string(plainText)
			bestKey = byte(key)
		}
	}

	return bestKey, bestCost, bestString
}

func crackXORByteScore(cipherText []byte) (key byte, cost float64, plainText string) {
	bestScore := float64(0)
	var bestString string
	var bestKey byte
	for _, key := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz 0123456789" {
		plainText := decryptXORByte(cipherText, byte(key))
		score := calcStringScore(plainText)

		if score > bestScore {
			bestScore = score
			bestString = string(plainText)
			bestKey = byte(key)
		}
	}

	return bestKey, bestScore, bestString
}

func hamming(a []byte, b []byte) (int, error) {
	if len(a) != len(b) {
		return -1, fmt.Errorf("strings not equal length")
	}

	length := len(a)
	if length == 0 {
		return 0, nil
	}

	count := 0
	for i := range a {
		// XOR the bytes, the number of remaining 1-bits represent
		// the differing bits.
		diff := a[i] ^ b[i]

		// Count the number of 1-bits in the result
		for j := 0; j < 8; j++ {
			count += int(diff & 1)
			diff >>= 1
		}
	}

	return count, nil
}

func decryptRepeatingKeyXOR(cipherText []byte, key []byte) []byte {
	plainText := make([]byte, len(cipherText))
	for i := 0; i < len(cipherText); i += len(key) {
		end := i + len(key)
		if end > len(cipherText) {
			end = len(cipherText)
		}

		for j := range key {
			if i+j < end {
				plainText[i+j] = cipherText[i+j] ^ key[j]
			}
		}
	}

	return plainText
}

// This function returns the mean hamming distance between blocks of size
// blockSize.
func meanBlockHammingDistance(data []byte, blockSize int, opts ...map[string]string) (float64, error) {
	// Get the average of maxBlocks blocks
	maxBlocks := 10

	if len(opts) > 0 {
		intBlocks, err := strconv.ParseInt(opts[0]["maxBlocks"], 10, 16)
		maxBlocks = int(intBlocks)
		if err != nil {
			return 0, fmt.Errorf("could not parse opts: %w", err)
		}
	}

	meanDistance := float64(0)
	for i := 0; i < maxBlocks; i++ {
		start := i * blockSize
		first := data[start : start+blockSize]
		second := data[start+blockSize : start+blockSize+blockSize]
		distance, err := hamming(first, second)

		if err != nil {
			return 0, fmt.Errorf("could not compute hamming distance: %w", err)
		}

		normalizedDistance := float64(distance) / float64(blockSize)
		meanDistance += normalizedDistance
		meanDistance /= 2
	}

	return meanDistance, nil
}

// Returns the total hamming distance between each block and every other
// block in data, given blockSize.
func blockDistance(data []byte, blockSize int) (float64, error) {
	numBlocks := len(data) / blockSize

	totalDistance := float64(0)
	for i := 0; i < numBlocks; i++ {
		for j := i; j < numBlocks; j++ {
			if i == j {
				continue
			}
			first := data[i*blockSize : (i+1)*blockSize]
			second := data[j*blockSize : (j+1)*blockSize]
			distance, err := hamming(first, second)
			if err != nil {
				return 0, fmt.Errorf("could not compute hamming distance: %w", err)
			}

			totalDistance += math.Pow(float64(distance)/float64(blockSize), 2)
		}
	}

	return math.Sqrt(totalDistance), nil
}

// Returns the number of blocks that have a similarity score under minSimilarity. The score
// is the hamming distance between the blocks.
func numSimilarBlocks(data []byte, blockSize int, minSimilarity int) (int, error) {
	numBlocks := len(data) / blockSize

	count := 0
	for i := 0; i < numBlocks; i++ {
		for j := i; j < numBlocks; j++ {
			if i == j {
				continue
			}
			first := data[i*blockSize : (i+1)*blockSize]
			second := data[j*blockSize : (j+1)*blockSize]
			distance, err := hamming(first, second)
			if err != nil {
				return 0, fmt.Errorf("could not compute hamming distance: %w", err)
			}

			if distance <= minSimilarity {
				count++
			}
		}
	}

	return count, nil
}

func padPKCS7Bytes(plainText []byte, length int) ([]byte, error) {
	if length > 256 {
		return nil, fmt.Errorf("cannot pad length > 256")
	}

	if length == 0 {
		return plainText, nil
	}

	textLength := len(plainText)
	diff := length - textLength

	if diff < 0 {
		return nil, fmt.Errorf("plainText longer than length")
	}

	if diff == 0 {
		return plainText, nil
	}

	padding := make([]byte, diff)
	for i := 0; i < diff; i++ {
		padding[i] = byte(diff)
	}

	return append(plainText, padding...), nil
}

func padPKCS7ToBlockSize(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length%blockSize == 0 {
		padding := make([]byte, blockSize)
		for i := 0; i < blockSize; i++ {
			padding[i] = byte(blockSize)
		}

		return append(data, padding...), nil
	}

	diff := blockSize - length%blockSize
	padding := make([]byte, diff)
	for i := 0; i < diff; i++ {
		padding[i] = byte(diff)
	}

	return append(data, padding...), nil
}

func unpadPKCS7(data []byte) ([]byte, error) {
	length := len(data)

	// Assume that the last byte was a padding byte
	paddingByte := data[length-1]
	count := byte(0)

	if paddingByte == 0 {
		return nil, fmt.Errorf("invalid zero padding byte: %d", paddingByte)
	}

	if paddingByte > 16 {
		return nil, fmt.Errorf("invalid padding byte: %d", paddingByte)
	}

	for i := length - 1; i >= 0; i-- {
		if data[i] == paddingByte {
			count++
			if count > paddingByte {
				return nil, fmt.Errorf("invalid padding byte: %d, count: %d", paddingByte, count)
			}
		} else {
			if count == paddingByte {
				return data[:i+1], nil
			}
			return nil, fmt.Errorf("invalid padding byte: %d, count: %d", paddingByte, count)
		}
	}

	return data, nil
}

func padPKCS7(plainText string, length int) (string, error) {
	padded, err := padPKCS7Bytes([]byte(plainText), length)
	return string(padded), err
}

func detectBlockSize(data []byte) (int, error) {
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

func encodeCookie(cookie map[string]string, order []string) string {
	cookies := []string{}
	for _, k := range order {
		cookies = append(cookies, fmt.Sprintf("%s=%s", sanitizeCookieValue(k), sanitizeCookieValue(cookie[k])))
	}

	return strings.Join(cookies, "&")
}

func sanitizeCookieValue(val string) string {
	sanitizedString := ""
	for _, c := range val {
		if c != '&' && c != '=' {
			sanitizedString += string(c)
		}
	}

	return sanitizedString
}

// Zero-pad hex strings to even-valued length
func zeroPad(s string) string {
	if len(s)%2 == 1 {
		return "0" + s
	}
	return s
}

// Modular exponentiation for Big Ints
func bigModExp(base *big.Int, exponent *big.Int, modulus *big.Int) *big.Int {
	if modulus.Cmp(big.NewInt(1)) == 0 {
		return big.NewInt(0)
	}

	if exponent.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(1)
	}

	result := bigModExp(base, new(big.Int).Div(exponent, big.NewInt(2)), modulus)
	result = new(big.Int).Mod(new(big.Int).Mul(result, result), modulus)

	// if exponent & 1 != 0, means, if exponent % 2 != 0, means, if exponent is not divisible by 2
	if new(big.Int).Mod(exponent, big.NewInt(2)).Int64() != 0 {
		return new(big.Int).Mod(new(big.Int).Mul(new(big.Int).Mod(base, modulus), result), modulus)
	}

	return new(big.Int).Mod(result, modulus)
}

func cubeRoot(i *big.Int) (cbrt *big.Int, rem *big.Int) {
	var (
		n0    = big.NewInt(0)
		n1    = big.NewInt(1)
		n2    = big.NewInt(2)
		n3    = big.NewInt(3)
		guess = new(big.Int).Div(i, n2)
		dx    = new(big.Int)
		absDx = new(big.Int)
		minDx = new(big.Int).Abs(i)
		step  = new(big.Int).Abs(new(big.Int).Div(guess, n2))
		cube  = new(big.Int)
	)

	for {
		cube.Exp(guess, n3, nil)
		dx.Sub(i, cube)
		cmp := dx.Cmp(n0)
		if cmp == 0 {
			return guess, n0
		}

		absDx.Abs(dx)
		switch absDx.Cmp(minDx) {
		case -1:
			minDx.Set(absDx)
		case 0:
			return guess, dx
		}

		switch cmp {
		case -1:
			guess.Sub(guess, step)
		case +1:
			guess.Add(guess, step)
		}

		step.Div(step, n2)
		if step.Cmp(n0) == 0 {
			step.Set(n1)
		}
	}
}
