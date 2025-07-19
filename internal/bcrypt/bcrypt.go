package bcrypt

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", fmt.Errorf("error hashing password: %v", err)
	}

	return string(hash), nil
}

func ComparePassword(input_password, user_password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(user_password), []byte(input_password)); err != nil {
		return fmt.Errorf("password does not match: %v", err)
	}
	return nil
}
