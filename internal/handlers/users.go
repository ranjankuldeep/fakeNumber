package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// Allowed email domains
var allowedDomains = []string{
	"gmail.com",
	"outlook.com",
	"hotmail.com",
	"yahoo.com",
}

// Utility function to get email domain
func getEmailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Struct to parse the request payload
type SignupRequest struct {
	Email   string `json:"email"`
	Captcha string `json:"captcha"`
}

// Function to verify CAPTCHA
func verifyCaptcha(captcha string) error {
	recaptchaSecretKey := os.Getenv("RECAPTCHA_SECRET_KEY")
	url := fmt.Sprintf("https://www.google.com/recaptcha/api/siteverify?secret=%s&response=%s", recaptchaSecretKey, captcha)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.New("invalid CAPTCHA")
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if success, ok := result["success"].(bool); !ok || !success {
		return errors.New("CAPTCHA verification failed")
	}
	return nil
}

// Function to generate OTP (stubbed here for simplicity)
func generateOTP() string {
	return "123456" // Replace with actual OTP generation logic
}

// Handler function for signup
func signup(c echo.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Check if email domain is allowed
	emailDomain := getEmailDomain(req.Email)
	isAllowed := false
	for _, domain := range allowedDomains {
		if domain == emailDomain {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Use Valid Email"})
	}

	// Check for CAPTCHA
	if req.Captcha == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Please complete the CAPTCHA"})
	}

	// Verify CAPTCHA
	if err := verifyCaptcha(req.Captcha); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Check if the email already exists in the database (replace with actual DB lookup)
	existingUser, err := findUserByEmail(req.Email) // Stub function to simulate DB lookup
	if err == nil && existingUser != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email already exists"})
	}

	// Generate and send OTP (replace with actual email sending logic)
	otp := generateOTP()
	otpText := "Your OTP for registration is"
	subText := "OTP verification"
	if err := sendOTPByEmail(req.Email, otp, otpText, subText); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to send OTP"})
	}

	// Store OTP for the user in database (replace with actual DB storage logic)
	if err := storeOTP(req.Email, otp); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store OTP"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "PENDING",
		"message": "Verification OTP sent.",
		"email":   req.Email,
	})
}

// Helper functions (stubbed for demonstration)

// Replace with actual DB lookup logic
func findUserByEmail(email string) (interface{}, error) {
	// Return nil if user is not found
	return nil, nil
}

// Replace with actual email sending logic
func sendOTPByEmail(email, otp, otpText, subText string) error {
	// Simulate sending email
	fmt.Printf("Sent OTP %s to %s\n", otp, email)
	return nil
}

// Replace with actual OTP storage logic
func storeOTP(email, otp string) error {
	// Simulate storing OTP
	fmt.Printf("Stored OTP %s for email %s\n", otp, email)
	return nil
}
