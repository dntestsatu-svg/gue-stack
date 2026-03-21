package queue

import "encoding/json"

const (
	TypeSendWelcomeEmail = "email:send_welcome"
)

type WelcomeEmailPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewWelcomeEmailPayload(email, name string) ([]byte, error) {
	return json.Marshal(WelcomeEmailPayload{
		Email: email,
		Name:  name,
	})
}
