package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	key := aesKey("foo")
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	enc, err := aesEncrypt(key, data)
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 {
		os.WriteFile(os.Args[1], enc, 0644)
	} else {
		os.Stdout.Write(enc)
	}
}

func newTestIV() io.Reader {
	return strings.NewReader("1234567812345678")
}

// This will usually receive a JWT token as an input, but since the token has more than 32 bytes, we'll hash it so it
// can be used as a key for AES encryption
func aesKey(input string) []byte {
	h := md5.New()
	h.Write([]byte(input))
	key := hex.EncodeToString(h.Sum(nil))
	return []byte(key)
}

func aesEncrypt(key []byte, input []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	payload, _ := pkcs7Padding(input, aes.BlockSize)
	output := make([]byte, aes.BlockSize+len(payload))

	// CBC mode works on blocks, so plain text may need to be padded to the next whole block
	// https://www.rfc-editor.org/rfc/rfc5246#section-6.2.3.2
	iv := output[:aes.BlockSize]
	if _, err := io.ReadFull(newTestIV(), iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(output[aes.BlockSize:], payload)

	return output, nil
}

func withPadding(payload []byte, blockSize int) []byte {
	if len(payload)%aes.BlockSize == 0 {
		return payload
	}
	data := make([]byte, int(len(payload)/blockSize+1)*blockSize)
	copy(data, payload)
	return data
}

// Right-pads the data string with 1 to n bytes according to PKCS#7 where n is the block size. The size of the result
// is x times n, where x is at least 1. The version of PKCS#7 padding used is the one defined in RFC 5652 chapter 6.3.
// This padding is identical to PKCS#5 padding for 8 byte block ciphers such as DES
func pkcs7Padding(payload []byte, blockSize int) ([]byte, uint8) {
	padding := blockSize - len(payload)%blockSize
	text := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(payload, text...), uint8(padding)
}

func zeroPad(payload []byte, blockSize int) []byte {
	padding := blockSize - len(payload)%blockSize
	text := bytes.Repeat([]byte{byte('0')}, padding)
	return append(payload, text...)
}
