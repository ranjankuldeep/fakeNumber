package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/lib"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func GoogleSignup(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	type RequestBody struct {
		Token string `json:"token"`
	}

	var body RequestBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Fetch user data from Google
	profile, err := fetchGoogleUserProfile(body.Token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Failed to fetch user data"})
	}

	// Check if user exists
	userCollection := models.InitializeUserCollection(db)
	filter := bson.M{"email": profile["email"]}
	var user models.User
	err = userCollection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		// Create a new user if not found
		now := time.Now()
		newUser := models.User{
			ID:          primitive.NewObjectID(),
			GoogleID:    profile["id"].(string),
			DisplayName: profile["name"].(string),
			Email:       profile["email"].(string),
			ProfileImg:  profile["picture"].(string),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		_, err = userCollection.InsertOne(context.TODO(), newUser)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		}

		// Generate API key and wallet
		apiKey := generateAPIKey()

		wallet, err := lib.GenerateTronAddress()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate TRON wallet"})
		}

		// todo
		trxAddress := wallet["address"]
		trxPrivateKey := wallet["privateKey"]

		apiWallet := models.ApiWalletUser{
			UserID:        newUser.ID,
			APIKey:        apiKey,
			Balance:       0,
			TRXAddress:    trxAddress,
			TRXPrivateKey: trxPrivateKey,
		}

		apiWalletColl := models.InitializeApiWalletuserCollection(db)
		_, err = apiWalletColl.InsertOne(context.TODO(), apiWallet)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create wallet"})
		}

		// Generate JWT token
		token := generateJWT(newUser.Email, newUser.ID.Hex(), "google", trxAddress)
		return c.JSON(http.StatusOK, map[string]string{"token": token})
	}

	return c.JSON(http.StatusBadRequest, map[string]string{"error": "User already exists, Please Login."})
}

func GoogleLogin(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	type RequestBody struct {
		Token string `json:"token"`
	}

	var body RequestBody
	if err := c.Bind(&body); err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Fetch user data from Google
	profile, err := fetchGoogleUserProfile(body.Token)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Failed to fetch user data"})
	}

	// Check if user exists
	userCollection := models.InitializeUserCollection(db)
	filter := bson.M{"email": profile["email"]}
	var user models.User
	err = userCollection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User not found, Please register."})
	}

	// Fetch wallet details
	apiWalletColl := models.InitializeApiWalletuserCollection(db)
	var apiWallet models.ApiWalletUser
	err = apiWalletColl.FindOne(context.TODO(), bson.M{"userId": user.ID}).Decode(&apiWallet)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch wallet details"})
	}

	// Generate JWT token
	token := generateJWT(user.Email, user.ID.Hex(), "google", apiWallet.TRXAddress)
	return c.JSON(http.StatusOK, map[string]string{"token": token})
}

// todo replace with env file
// JWT Secret Key
var jwtSecretKey = []byte("your_secret_key") // Replace with your secret key

// Claims defines the structure of the JWT claims
type Claims struct {
	Email      string `json:"email"`
	UserID     string `json:"userId"`
	LoginType  string `json:"logintype"`
	TRXAddress string `json:"trxAddress"`
	jwt.StandardClaims
}

// Generate a JWT token
func generateJWT(email, userID, loginType, trxAddress string) string {
	claims := &Claims{
		Email:      email,
		UserID:     userID,
		LoginType:  loginType,
		TRXAddress: trxAddress,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecretKey)
	if err != nil {
		fmt.Println("Error generating JWT:", err)
		return ""
	}
	return signedToken
}

// Fetch Google user profile using access token
func fetchGoogleUserProfile(token string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://www.googleapis.com/oauth2/v1/userinfo?access_token=%s", token)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch profile, status code: %d", resp.StatusCode)
	}

	var profile map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&profile)
	if err != nil {
		return nil, fmt.Errorf("error decoding profile response: %v", err)
	}

	return profile, nil
}

// Generate a random API key
func generateAPIKey() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		fmt.Println("Error generating API key:", err)
		return ""
	}
	return hex.EncodeToString(bytes)
}
