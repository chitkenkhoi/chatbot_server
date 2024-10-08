package utils

import (
	"fmt"
	"math/rand"

	"gopkg.in/gomail.v2"
)

func GenerateOTP() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
func SendMail(to, otp string) error {
	m := gomail.NewMessage()
	from := "chatbot.newgen123@gmail.com"
	subject := "OTP code"
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	body := "This is your OTP code:\n" + otp
	m.SetBody("text/plain", body)
	d := gomail.NewDialer("smtp.gmail.com", 587, from, "zhpm jmci jnkm nlxf")
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
