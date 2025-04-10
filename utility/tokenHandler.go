package utility

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt"
)

// checks if the token is expired
func CheckTokenExpiry(tokenStr string, PubtokenKey *ecdsa.PublicKey) (bool, error) {

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return PubtokenKey, nil
	})

	if err != nil {
		fmt.Printf("There was an error parsing the token string, %v\n", err)
		return false, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println("Token is valid.")
		exp := int64(claims["exp"].(float64))
		currentTime := time.Now().Unix()
		difference := exp - currentTime

		if difference < 86400 {
			fmt.Println("The token's expiration is less than 24 hours away.")
			return true, nil
		} else {
			return false, nil
		}

	} else {
		fmt.Println(err)
		return false, err
	}

}

// converts the permissiondata to a the ECDSA public key
func PemToECDSAPub(pemData string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	ecdsaPub, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not of type ECDSA")
	}

	return ecdsaPub, nil
}

// checks if the JWT token is valid
func IsValidJWT(token string) bool {
	// This regex pattern checks for three base64url-encoded strings separated by dots.
	// It does not check the validity of the base64url encoding itself in-depth, nor does it validate the token's integrity.
	pattern := `^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]*$`
	r := regexp.MustCompile(pattern)
	return r.MatchString(token)
}