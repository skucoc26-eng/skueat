package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	output := flag.String("output", filepath.Join(configDir, "skueat", "admin-credentials.json"), "관리자 자격 증명을 저장할 로컬 파일 (기존 파일은 덮어쓰지 않음)")
	username := flag.String("username", "admin", "관리자 아이디")
	flag.Parse()
	var random [24]byte
	if _, err := rand.Read(random[:]); err != nil {
		log.Fatal(err)
	}
	password := base64.RawURLEncoding.EncodeToString(random[:])
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0700); err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Fatal(err)
	}
	err = json.NewEncoder(file).Encode(map[string]string{"username": *username, "password": password, "password_hash": string(hash), "url": "https://skueat.fly.dev/admin"})
	closeErr := file.Close()
	if err != nil {
		log.Fatal(err)
	}
	if closeErr != nil {
		log.Fatal(closeErr)
	}
	fmt.Printf("관리자 자격 증명 저장: %s\n", *output)
}
