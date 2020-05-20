package cryptopals

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

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

func TestS5C33(t *testing.T) {
	// Implement Diffe-Hellman key exchange
	rand.Seed(time.Now().UnixNano())

	// Shared by Alice and Bob (non secret, sent in clear)
	p := uint64(37)
	g := uint64(5)

	a := uint64(rand.Intn(10)) % p                    // Alice's secret
	A := uint64(math.Pow(float64(g), float64(a))) % p // Public: sent to Bob in the clear

	b := uint64(rand.Intn(10)) % p                    // Bob's secret
	B := uint64(math.Pow(float64(g), float64(b))) % p // Public: sent to Alice in the clear

	// Now, Bob and Alice can generate the shared secret 's', which can be hashed to
	// generate a symmetric key.
	//
	// Below s1 and s2 should compute to the same value == 's'.

	// Shared secret generated by Alice using her secret (a) and Bob's public key (B)
	s1 := uint64(math.Pow(float64(B), float64(a))) % p
	// Shared secret generated by Bob using his secret (b) and Alice's public key (A)
	s2 := uint64(math.Pow(float64(A), float64(b))) % p

	fmt.Printf("a = %d, b = %d, p = %d, g = %d\n", a, b, p, g)
	fmt.Printf("verifying s1 (%d) == s2 (%d)\n", s1, s2)
	assertEquals(t, s1, s2)

	// Validate that modExp works with small numbers
	fmt.Printf("verifying (g ^ a) %% p = A\n")
	fmt.Printf("i.e., (%d ^ %d) %% %d = %d\n", g, a, p, A)
	assertEquals(t, int64(A), bigModExp(big.NewInt(int64(g)), big.NewInt(int64(a)), big.NewInt(int64(p))).Int64())

	// Now do the same thing with huge numbers. We use the modular exponentiation function
	// above to calculate (a ^ b) % p.

	bpStr := "ffffffffffffffffc90fdaa22168c234c4c6628b80dc1cd129024e088a67cc74020bbea63b139b22514a08798e3404ddef9519b3cd3a431b302b0a6df25f14374fe1356d6d51c245e485b576625e7ec6f44c42e9a637ed6b0bff5cb6f406b7edee386bfb5a899fa5ae9f24117c4b1fe649286651ece45b3dc2007cb8a163bf0598da48361c55d39a69163fa8fd24cf5f83655d23dca3ad961c62f356208552bb9ed529077096966d670c354e4abc9804f1746c08ca237327ffffffffffffffff"
	bpBytes, err := hex.DecodeString(bpStr)
	assertNoError(t, err)

	bp := new(big.Int).SetBytes(bpBytes)
	bg := big.NewInt(2)

	ba := new(big.Int).Mod(big.NewInt(rand.Int63()), bp)
	bA := bigModExp(bg, ba, bp)

	bb := new(big.Int).Mod(big.NewInt(rand.Int63()), bp)
	bB := bigModExp(bg, bb, bp)

	bs1 := bigModExp(bB, ba, bp)
	bs2 := bigModExp(bA, bb, bp)

	fmt.Println("verifying s1 == s2 with big nums")
	assertTrue(t, bs1.Cmp(bs2) == 0)
}

func TestS5C34(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	plainText := []byte("Hi, I'm Alice!")

	type RPC struct {
		// We have the following message codes:
		//  DH, DHReply
		//  ECHO, ECHOReply
		Code   string
		Params map[string]string
	}

	genKeys := func(pHex string, gHex string, publicHex string) ([]byte, string, string) {
		p, ok := new(big.Int).SetString(pHex, 16)
		assertTrue(t, ok)
		g, ok := new(big.Int).SetString(gHex, 16)
		assertTrue(t, ok)
		A, ok := new(big.Int).SetString(publicHex, 16)
		assertTrue(t, ok)

		b := new(big.Int).Mod(big.NewInt(rand.Int63()), p)
		B := bigModExp(g, b, p)

		s := bigModExp(A, b, p)
		md := sha1.Sum(s.Bytes())
		key := md[:16]

		return key, b.Text(16), B.Text(16)
	}

	// echoBot is an encrypted echo server that uses Diffie-Hellman
	// to negotiate a symmetric key, and then return echo replies
	// encrypted with the key.
	echoBot := func() chan RPC {
		c := make(chan RPC)

		var key []byte
		authenticated := false

		// Start server
		go func() {
			for msg := range c {
				if msg.Code == "DH" {
					var B string
					key, _, B = genKeys(msg.Params["p"], msg.Params["g"], msg.Params["A"])
					fmt.Println("echoBot key:", hex.EncodeToString(key))

					c <- RPC{
						Code: "DHReply",
						Params: map[string]string{
							"B": B,
						},
					}

					authenticated = true
				} else if msg.Code == "ECHO" {
					// Echo message. Validate that we have keys.
					assertTrue(t, authenticated)

					// Decode encrypted message and IV
					cipherTextHex, ok := msg.Params["message"]
					assertTrue(t, ok)

					cipherText, err := hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok := msg.Params["iv"]
					assertTrue(t, ok)

					iv, err := hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err := decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err := unpadPKCS7(message)
					assertNoError(t, err)
					fmt.Println("echoBot (recv:decrypted):", string(unpaddedMessage))

					// Encrypt with new IV and return to client
					_, err = rand.Read(iv)
					assertNoError(t, err)

					cipherText, err = encryptAESCBC(message, key, iv)
					assertNoError(t, err)

					c <- RPC{
						Code: "ECHOReply",
						Params: map[string]string{
							"message": hex.EncodeToString(cipherText),
							"iv":      hex.EncodeToString(iv),
						},
					}
				}
			}
		}()

		return c
	}

	// middleMan performs a man-in-the-middle attack between aliceBot and echoBot
	// using DH parameter injection.
	middleMan := func(echoChan chan RPC) chan RPC {
		c := make(chan RPC)

		var key []byte
		authenticated := false

		// Start server
		go func() {
			for msg := range c {
				if msg.Code == "DH" {
					// Parameter injection. Send "p" instead of Alice's public key "A"
					echoChan <- RPC{
						Code: "DH",
						Params: map[string]string{
							"p": msg.Params["p"],
							"g": msg.Params["g"],
							"A": msg.Params["p"],
						},
					}

					// Throw away the reply
					<-echoChan

					// Send "p" to Alice as echoServer's public key
					c <- RPC{
						Code: "DHReply",
						Params: map[string]string{
							"B": msg.Params["p"],
						},
					}

					// Since (p ^ anything) % p == 0, we've fooled the client
					// and server into selecting zero-byte private keys. Turn
					// it into a symmetric key for encryption/decryption.
					md := sha1.Sum(new(big.Int).Bytes())
					key = md[:16]
					authenticated = true
				} else if msg.Code == "ECHO" {
					// Echo message. Validate that we have keys.
					assertTrue(t, authenticated)

					// Decode encrypted message and IV
					cipherTextHex, ok := msg.Params["message"]
					assertTrue(t, ok)

					cipherText, err := hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok := msg.Params["iv"]
					assertTrue(t, ok)

					iv, err := hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err := decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err := unpadPKCS7(message)
					assertNoError(t, err)
					fmt.Println("middleMan (decrypted:client):", string(unpaddedMessage))
					assertTrue(t, bytes.Equal(plainText, unpaddedMessage))

					// Passthrough original message
					echoChan <- msg

					// Decrypt echoServer reply
					reply := <-echoChan

					// Decode encrypted message and IV
					cipherTextHex, ok = reply.Params["message"]
					assertTrue(t, ok)

					cipherText, err = hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok = reply.Params["iv"]
					assertTrue(t, ok)

					iv, err = hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err = decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err = unpadPKCS7(message)
					assertNoError(t, err)
					fmt.Println("middleMan (decrypted:echoBot):", string(unpaddedMessage))
					assertTrue(t, bytes.Equal(plainText, unpaddedMessage))

					c <- reply
				}
			}
		}()

		return c
	}

	aliceBot := func(echoChan chan RPC) {
		// Generate private and public keys
		p, ok := new(big.Int).SetString("ffffffffffffffffc90fdaa22168c234c4c6628b80dc1cd129024e088a67cc74020bbea63b139b22514a08798e3404ddef9519b3cd3a431b302b0a6df25f14374fe1356d6d51c245e485b576625e7ec6f44c42e9a637ed6b0bff5cb6f406b7edee386bfb5a899fa5ae9f24117c4b1fe649286651ece45b3dc2007cb8a163bf0598da48361c55d39a69163fa8fd24cf5f83655d23dca3ad961c62f356208552bb9ed529077096966d670c354e4abc9804f1746c08ca237327ffffffffffffffff", 16)
		assertTrue(t, ok)
		g := big.NewInt(2)
		a := new(big.Int).Mod(big.NewInt(rand.Int63()), p)
		A := bigModExp(g, a, p)

		// Send public key (along with p and g) to Echo Server
		echoChan <- RPC{
			Code: "DH",
			Params: map[string]string{
				"p": p.Text(16),
				"g": g.Text(16),
				"A": A.Text(16),
			},
		}

		// Get server's public key from response
		reply := <-echoChan
		assertEquals(t, reply.Code, "DHReply")

		B, ok := new(big.Int).SetString(reply.Params["B"], 16)
		assertTrue(t, ok)

		// Generate shared secret
		s := bigModExp(B, a, p)
		md := sha1.Sum(s.Bytes())
		key := md[:16]
		fmt.Println("aliceBot key:", hex.EncodeToString(key))

		// Pad message and send to Echo Server
		message, err := padPKCS7ToBlockSize(plainText, 16)
		assertNoError(t, err)
		iv := make([]byte, 16)
		_, err = rand.Read(iv)
		assertNoError(t, err)

		cipherText, err := encryptAESCBC(message, key, iv)
		assertNoError(t, err)

		echoChan <- RPC{
			Code: "ECHO",
			Params: map[string]string{
				"message": hex.EncodeToString(cipherText),
				"iv":      hex.EncodeToString(iv),
			},
		}

		reply = <-echoChan
		assertEquals(t, reply.Code, "ECHOReply")

		// Decode encrypted reply and IV
		cipherTextHex, ok := reply.Params["message"]
		assertTrue(t, ok)

		cipherText, err = hex.DecodeString(cipherTextHex)
		assertNoError(t, err)

		ivHex, ok := reply.Params["iv"]
		assertTrue(t, ok)

		iv, err = hex.DecodeString(ivHex)
		assertNoError(t, err)
		assertEquals(t, 16, len(iv))

		message, err = decryptAESCBC(cipherText, key, iv)
		assertNoError(t, err)
		unpaddedMessage, err := unpadPKCS7(message)
		assertNoError(t, err)

		fmt.Println("aliceBot (recv:decrypted):", string(unpaddedMessage))

		assertTrue(t, bytes.Equal(unpaddedMessage, plainText))
		close(echoChan)
	}

	// Verify that echoBot works alone
	echoChan := echoBot()
	aliceBot(echoChan)

	// Verify that echoBot works via middleman
	echoChan = echoBot()
	mmChan := middleMan(echoChan)
	aliceBot(mmChan)
}

func TestS5C35(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	plainText := []byte("Hi, I'm Alice!")

	type RPC struct {
		// We have the following message codes:
		//  Handshake, HandshakeReply
		//  Exchange, ExchangeReply
		//  Echo, EchoReply
		Code   string
		Params map[string]string
	}

	// echoBot is an encrypted echo server that uses Diffie-Hellman
	// to negotiate a symmetric key, and then return echo replies
	// encrypted with the key.
	echoBot := func() chan RPC {
		c := make(chan RPC)

		// Start server
		go func() {
			var p *big.Int
			var g *big.Int
			var A *big.Int
			var b *big.Int
			var B *big.Int
			var s *big.Int
			var key []byte
			authenticated := false

			for msg := range c {
				fmt.Println("echoBot: Received", msg.Code)
				if msg.Code == "Handshake" {
					fmt.Println("echoBot: Handshake", msg.Params)
					pBytes, err := hex.DecodeString(msg.Params["p"])
					assertNoError(t, err)

					gBytes, err := hex.DecodeString(msg.Params["g"])
					assertNoError(t, err)

					p = new(big.Int).SetBytes(pBytes)
					g = new(big.Int).SetBytes(gBytes)

					c <- RPC{
						Code: "HandshakeReply",
					}
				} else if msg.Code == "Exchange" {
					ABytes, err := hex.DecodeString(msg.Params["A"])
					assertNoError(t, err)

					A = new(big.Int).SetBytes(ABytes)

					b = new(big.Int).Mod(big.NewInt(rand.Int63()), p)
					B = bigModExp(g, b, p)

					s = bigModExp(A, b, p)
					md := sha1.Sum(s.Bytes())
					key = md[:16]

					fmt.Println("echoBot key:", hex.EncodeToString(key))

					c <- RPC{
						Code: "ExchangeReply",
						Params: map[string]string{
							"B": zeroPad(B.Text(16)),
						},
					}

					authenticated = true
				} else if msg.Code == "Echo" {
					// Echo message. Validate that we have keys.
					assertTrue(t, authenticated)

					// Decode encrypted message and IV
					cipherTextHex, ok := msg.Params["message"]
					assertTrue(t, ok)

					cipherText, err := hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok := msg.Params["iv"]
					assertTrue(t, ok)

					iv, err := hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err := decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err := unpadPKCS7(message)
					if err != nil {
						unpaddedMessage = []byte("PADDING ERROR")
					}
					fmt.Println("echoBot (recv:decrypted):", string(unpaddedMessage))

					// Encrypt with new IV and return to client
					_, err = rand.Read(iv)
					assertNoError(t, err)

					cipherText, err = encryptAESCBC(message, key, iv)
					assertNoError(t, err)

					c <- RPC{
						Code: "EchoReply",
						Params: map[string]string{
							"message": hex.EncodeToString(cipherText),
							"iv":      hex.EncodeToString(iv),
						},
					}

				}
			}
		}()

		return c
	}

	// Middleman to inject g:
	//
	//   if g == 1, then B == 1, and alice's s == 1
	//   if g == p, then B == 0, and alice's s == 0
	//   if g == p - 1, then B == 1, and alice's s == 1

	middleMan := func(echoChan chan RPC) chan RPC {
		c := make(chan RPC)

		var key []byte
		authenticated := false

		// Start server
		go func() {
			for msg := range c {
				if msg.Code == "Handshake" {
					// Parameter injection. Send "p" instead of Alice's public key "A"
					pMinus1, ok := new(big.Int).SetString(msg.Params["p"], 16)
					assertTrue(t, ok)
					pMinus1 = pMinus1.Sub(pMinus1, big.NewInt(1))
					fmt.Println("p - 1 =", len(pMinus1.Text(16)))

					echoChan <- RPC{
						Code: "Handshake",
						Params: map[string]string{
							"p": msg.Params["p"],
							"g": zeroPad(pMinus1.Text(16)),
						},
					}

					// Pass through reply
					reply := <-echoChan
					c <- reply

					md := sha1.Sum(big.NewInt(1).Bytes())
					key = md[:16]
					authenticated = true
				} else if msg.Code == "Exchange" {
					// Pass through key exchange. Assume server has calculated key using
					// injected g
					echoChan <- msg
					reply := <-echoChan
					c <- reply
				} else if msg.Code == "Echo" {
					// Echo message. Validate that we have keys.
					assertTrue(t, authenticated)

					// Decode encrypted message and IV
					cipherTextHex, ok := msg.Params["message"]
					assertTrue(t, ok)

					cipherText, err := hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok := msg.Params["iv"]
					assertTrue(t, ok)

					iv, err := hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err := decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err := unpadPKCS7(message)
					assertNoError(t, err)
					fmt.Println("middleMan (decrypted:client):", string(unpaddedMessage))
					assertTrue(t, bytes.Equal(plainText, unpaddedMessage))

					// Passthrough original message
					echoChan <- msg

					// Decrypt echoServer reply
					reply := <-echoChan

					// Decode encrypted message and IV
					cipherTextHex, ok = reply.Params["message"]
					assertTrue(t, ok)

					cipherText, err = hex.DecodeString(cipherTextHex)
					assertNoError(t, err)

					ivHex, ok = reply.Params["iv"]
					assertTrue(t, ok)

					iv, err = hex.DecodeString(ivHex)
					assertNoError(t, err)
					assertEquals(t, 16, len(iv))

					// Decrypt message
					message, err = decryptAESCBC(cipherText, key, iv)
					assertNoError(t, err)
					unpaddedMessage, err = unpadPKCS7(message)
					if err != nil {
						unpaddedMessage = []byte("PADDING ERROR")
					}
					fmt.Println("middleMan (decrypted:echoBot):", string(unpaddedMessage))

					c <- reply
				}
			}
		}()

		return c
	}

	aliceBot := func(echoChan chan RPC) {
		// Generate private and public keys
		p, ok := new(big.Int).SetString("ffffffffffffffffc90fdaa22168c234c4c6628b80dc1cd129024e088a67cc74020bbea63b139b22514a08798e3404ddef9519b3cd3a431b302b0a6df25f14374fe1356d6d51c245e485b576625e7ec6f44c42e9a637ed6b0bff5cb6f406b7edee386bfb5a899fa5ae9f24117c4b1fe649286651ece45b3dc2007cb8a163bf0598da48361c55d39a69163fa8fd24cf5f83655d23dca3ad961c62f356208552bb9ed529077096966d670c354e4abc9804f1746c08ca237327ffffffffffffffff", 16)
		assertTrue(t, ok)
		g := big.NewInt(2)
		a := new(big.Int).Mod(big.NewInt(rand.Int63()), p)
		A := bigModExp(g, a, p)

		// Send public key (along with p and g) to Echo Server
		echoChan <- RPC{
			Code: "Handshake",
			Params: map[string]string{
				"p": p.Text(16),
				"g": fmt.Sprintf("%02x", g.Int64()),
			},
		}

		// Get server's public key from response
		reply := <-echoChan
		assertEquals(t, reply.Code, "HandshakeReply")

		echoChan <- RPC{
			Code: "Exchange",
			Params: map[string]string{
				"A": zeroPad(A.Text(16)),
			},
		}

		reply = <-echoChan
		assertEquals(t, reply.Code, "ExchangeReply")

		B, ok := new(big.Int).SetString(reply.Params["B"], 16)
		assertTrue(t, ok)

		// Generate shared secret
		s := bigModExp(B, a, p)
		md := sha1.Sum(s.Bytes())
		key := md[:16]
		fmt.Println("aliceBot key:", hex.EncodeToString(key))

		// Pad message and send to Echo Server
		message, err := padPKCS7ToBlockSize(plainText, 16)
		assertNoError(t, err)
		iv := make([]byte, 16)
		_, err = rand.Read(iv)
		assertNoError(t, err)

		cipherText, err := encryptAESCBC(message, key, iv)
		assertNoError(t, err)

		echoChan <- RPC{
			Code: "Echo",
			Params: map[string]string{
				"message": hex.EncodeToString(cipherText),
				"iv":      hex.EncodeToString(iv),
			},
		}

		reply = <-echoChan
		assertEquals(t, reply.Code, "EchoReply")

		// Decode encrypted reply and IV
		cipherTextHex, ok := reply.Params["message"]
		assertTrue(t, ok)

		cipherText, err = hex.DecodeString(cipherTextHex)
		assertNoError(t, err)

		ivHex, ok := reply.Params["iv"]
		assertTrue(t, ok)

		iv, err = hex.DecodeString(ivHex)
		assertNoError(t, err)
		assertEquals(t, 16, len(iv))

		message, err = decryptAESCBC(cipherText, key, iv)
		assertNoError(t, err)
		unpaddedMessage, err := unpadPKCS7(message)
		if err != nil {
			unpaddedMessage = []byte("PADDING ERROR")
		}

		fmt.Println("aliceBot (recv:decrypted):", string(unpaddedMessage))

		// assertTrue(t, bytes.Equal(unpaddedMessage, plainText))
		close(echoChan)
	}

	// Verify that echoBot works alone
	echoChan := echoBot()
	aliceBot(echoChan)

	// Verify that the middleMan can decrypt traffic
	echoChan = echoBot()
	mm := middleMan(echoChan)
	aliceBot(mm)
}
