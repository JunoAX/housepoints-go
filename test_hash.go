package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash := "$2a$10$sTGx0JjVBbv3kgr3Z3Nv7.qr6xULdWup/oXlrkxZJ3nbQOsV7qiZe"
	password := "tom"

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Println("FAIL:", err)
	} else {
		fmt.Println("PASS - hash matches password")
	}
}
