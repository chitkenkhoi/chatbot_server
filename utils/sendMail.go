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
	
	body := `
	<html>
	<body style="font-family: Arial, sans-serif;">
		<p>Your email has been registered to our website, this is the code for verification:</p>
		<p style="font-size: 24px; font-weight: bold; color: green; text-align: center;">` + otp + `</p>
		<p>This code will be expired after <span style="color: red;">15 minutes</span>, if you did not register, please ignore this email.</p>
		<p style="color: red;">Do not give this code to anybody, including us!</p>
	</body>
	</html>
	`
	
	m.SetBody("text/html", body)
	
	d := gomail.NewDialer("smtp.gmail.com", 587, from, "zhpm jmci jnkm nlxf")
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
