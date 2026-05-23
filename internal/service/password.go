package service

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	BcryptCost     = 12
	MinPasswordLen = 8
	MaxPasswordLen = 72 // bcrypt limit
)

var commonPasswords = map[string]bool{
	"password": true, "12345678": true, "123456789": true, "1234567890": true,
	"qwerty123": true, "password1": true, "iloveyou": true, "admin123": true,
	"welcome1": true, "monkey123": true, "dragon12": true, "master12": true,
	"letmein1": true, "football": true, "baseball": true, "shadow12": true,
	"trustno1": true, "sunshine": true, "princess": true, "starwars": true,
	"whatever": true, "qwertyui": true, "passw0rd": true, "abcdefgh": true,
	"12341234": true, "11111111": true, "00000000": true, "password123": true,
	"admin1234": true, "changeme": true, "testtest": true, "qwerty12": true,
}

func ValidatePassword(password string) error {
	if len(password) < MinPasswordLen {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	if len(password) > MaxPasswordLen {
		return fmt.Errorf("password must be at most %d characters", MaxPasswordLen)
	}
	if commonPasswords[strings.ToLower(password)] {
		return fmt.Errorf("password is too common, please choose a stronger one")
	}
	return nil
}

func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false
	}
	return cost < BcryptCost
}
