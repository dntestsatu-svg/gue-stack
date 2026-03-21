package password

import "golang.org/x/crypto/bcrypt"

func Hash(raw string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func Compare(hash, raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw))
}
