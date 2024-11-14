package utils

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"golang.org/x/crypto/bcrypt"
)

// OTP expiration time in minutes
const OTPExpirationTime = 1 * time.Minute

// Email configuration
var smtpHost = "smtp.gmail.com"
var smtpPort = "587"
var mailUser = os.Getenv("MAIL_USER")
var mailPass = os.Getenv("MAIL_PASS")

// Mocked databases for OTPs
var OTPStore = make(map[string]models.OTP)       // Regular OTP storage
var ForgotOTPStore = make(map[string]models.OTP) // Forgot Password OTP storage

// GenerateOTP generates a 6-digit OTP
func GenerateOTP() string {
	return strconv.Itoa(100000 + int(time.Now().UnixNano()%900000))
}

// SendOTPByEmail sends OTP to the provided email
func SendOTPByEmail(email, otp, text, subject string) error {
	auth := smtp.PlainAuth("", mailUser, mailPass, smtpHost)
	msg := []byte("Subject: " + subject + "\r\n" + text + ": " + otp + "\r\n")
	to := []string{email}

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, mailUser, to, msg)
	if err != nil {
		return err
	}
	fmt.Printf("Sent OTP %s to %s\n", otp, email)
	return nil
}

// StoreOTP stores OTP for an email in OTPStore
func StoreOTP(email, otp string) error {
	// Check if OTP already exists for the email
	if _, exists := OTPStore[email]; exists {
		return fmt.Errorf("OTP already sent, please resend to get a new OTP")
	}

	// Hash the OTP
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %v", err)
	}

	// Store OTP in OTPStore with expiration
	OTPStore[email] = models.OTP{Email: email, OTP: string(hashedOTP), CreatedAt: time.Now()}
	fmt.Printf("Stored OTP %s for email %s\n", otp, email)

	// Set a timer to delete OTP after expiration
	time.AfterFunc(OTPExpirationTime, func() {
		delete(OTPStore, email)
		fmt.Println("OTP deleted after expiration for email:", email)
	})

	return nil
}

// ResendOTP resends a new OTP, replacing the existing one
func ResendOTP(email, otp string) error {
	// Delete existing OTP, if any
	delete(OTPStore, email)

	// Hash the new OTP
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %v", err)
	}

	// Store the new OTP
	OTPStore[email] = models.OTP{Email: email, OTP: string(hashedOTP), CreatedAt: time.Now()}
	fmt.Printf("Resent OTP %s for email %s\n", otp, email)

	// Set a timer to delete OTP after expiration
	time.AfterFunc(OTPExpirationTime, func() {
		delete(OTPStore, email)
		fmt.Println("OTP deleted after expiration for email:", email)
	})

	return nil
}

// StoreForgotOTP stores OTP for password reset
func StoreForgotOTP(email, otp string) error {
	// Check if OTP already exists
	if _, exists := ForgotOTPStore[email]; exists {
		return fmt.Errorf("OTP already sent, please resend to get a new OTP")
	}

	// Hash the OTP
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %v", err)
	}

	// Store OTP in ForgotOTPStore with expiration
	ForgotOTPStore[email] = models.OTP{Email: email, OTP: string(hashedOTP), CreatedAt: time.Now()}
	fmt.Printf("Stored Forgot OTP %s for email %s\n", otp, email)

	// Set a timer to delete OTP after expiration
	time.AfterFunc(OTPExpirationTime, func() {
		delete(ForgotOTPStore, email)
		fmt.Println("Forgot OTP deleted after expiration for email:", email)
	})

	return nil
}

// ResendForgotOTP resends OTP for password reset
func ResendForgotOTP(email, otp string) error {
	// Delete existing OTP if any
	delete(ForgotOTPStore, email)

	// Hash the new OTP
	hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash OTP: %v", err)
	}

	// Store the new OTP
	ForgotOTPStore[email] = models.OTP{Email: email, OTP: string(hashedOTP), CreatedAt: time.Now()}
	fmt.Printf("Resent Forgot OTP %s for email %s\n", otp, email)

	// Set a timer to delete OTP after expiration
	time.AfterFunc(OTPExpirationTime, func() {
		delete(ForgotOTPStore, email)
		fmt.Println("Forgot OTP deleted after expiration for email:", email)
	})

	return nil
}
