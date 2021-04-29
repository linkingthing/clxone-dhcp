package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

//fill
func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}
	return src[:(length - unpadding)], nil
}

func Encrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	msg := pad([]byte(text))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := make([]byte, aes.BlockSize)
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], msg)
	finalMsg := hex.EncodeToString([]byte(ciphertext[aes.BlockSize:]))
	return finalMsg, nil
}

func Decrypt(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	decodedMsg, _ := hex.DecodeString(text)
	iv := make([]byte, aes.BlockSize)
	msg := decodedMsg
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(msg, msg)
	unpadMsg, err := unpad(msg)
	if err != nil {
		return "", err
	}
	return string(unpadMsg), nil
}

func Md5(data string, salt string) string {
	h := md5.New()
	_, _ = io.WriteString(h, data)
	_, _ = io.WriteString(h, salt)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func RSAEncrypt(publicKeyStr []byte, text string) ([]byte, error) {
	block, _ := pem.Decode(publicKeyStr)
	if block == nil {
		return nil, fmt.Errorf("pem failed")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unknown keyType")
	}
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(text))
	if err != nil {
		return nil, err
	}

	return cipherText, nil
}

func RSADecrypt(privateKeyStr []byte, text []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKeyStr)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	plainText, _ := rsa.DecryptPKCS1v15(rand.Reader, privateKey, text)
	return plainText, nil
}

func GenerateRSAKey(bits int, filePath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	X509PrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	privateFile, err := os.Create(path.Join(filePath, "private.pem"))
	if err != nil {
		return err
	}
	defer privateFile.Close()
	privateBlock := pem.Block{Type: "RSA Private Key", Bytes: X509PrivateKey}
	if err := pem.Encode(privateFile, &privateBlock); err != nil {
		return err
	}
	publicKey := privateKey.PublicKey
	X509PublicKey, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return err
	}
	publicFile, err := os.Create(path.Join(filePath, "public.pem"))
	if err != nil {
		return err
	}
	defer publicFile.Close()
	publicBlock := pem.Block{Type: "RSA Public Key", Bytes: X509PublicKey}

	return pem.Encode(publicFile, &publicBlock)
}
