package utils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

)
func a(){
	email := "chitkenkhoi@gmail.com"
	hash := sha256.New()
	hash.Write([]byte(email + os.Getenv("KEY_FOR_REGISTER")))
	// hashBytes := hash.Sum(nil)
	a := os.Getenv("KEY_FOR_REGISTER")
	fmt.Println(a)
	
}
func TestGenerateToken(t *testing.T) {
	a()
	
}