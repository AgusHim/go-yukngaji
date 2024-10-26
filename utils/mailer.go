package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendEmail(to string, from string, subject string, body string) error {
	// SMTP server configuration.
	smtpServer := "smtp-relay.brevo.com"
	smtpPort := "587"
	username := os.Getenv("SMTP_LOGIN") // Set this in your environment
	password := os.Getenv("SMTP_PASSWORD")
	// Set this in your environment

	// Message.
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", from, to, subject, body)

	// Authentication.
	auth := smtp.PlainAuth("", username, password, smtpServer)

	// Sending email.
	err := smtp.SendMail(smtpServer+":"+smtpPort, auth, "no-reply@ynsolo.com", []string{to}, []byte(message))
	if err != nil {
		return err
	}
	return nil
}
