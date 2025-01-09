package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"

	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/gomail.v2"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/internal/services"
	"github.com/ranjankuldeep/fakeNumber/internal/utils"
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// Log: Starting CAPTCHA verification
	log.Println("INFO: Starting CAPTCHA verification")

	// Retrieve the reCAPTCHA secret key from the environment variable
	recaptchaSecretKey := os.Getenv("RECAPTCHA_SECRET_KEY")
	if recaptchaSecretKey == "" {
		log.Println("ERROR: RECAPTCHA_SECRET_KEY is not set in environment variables")
		return errors.New("internal server error")
	}
	log.Println("INFO: Retrieved reCAPTCHA secret key from environment variables")

	// Build the verification URL
	url := fmt.Sprintf("https://www.google.com/recaptcha/api/siteverify?secret=%s&response=%s", recaptchaSecretKey, captcha)
	log.Printf("INFO: Built reCAPTCHA verification URL: %s\n", url)

	// Make the HTTP POST request to verify the CAPTCHA
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Printf("ERROR: Failed to make HTTP request to reCAPTCHA verification API: %v\n", err)
		return errors.New("invalid CAPTCHA")
	}
	defer resp.Body.Close()

	log.Printf("INFO: Received response from reCAPTCHA API with status code: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		log.Println("ERROR: reCAPTCHA API returned a non-OK status")
		return errors.New("invalid CAPTCHA")
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read reCAPTCHA API response body: %v\n", err)
		return errors.New("invalid CAPTCHA")
	}
	log.Printf("INFO: Successfully read reCAPTCHA API response body: %s\n", string(body))

	// Parse the response body as JSON
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("ERROR: Failed to parse reCAPTCHA API response body as JSON: %v\n", err)
		return errors.New("invalid CAPTCHA")
	}
	log.Printf("INFO: Parsed reCAPTCHA API response: %+v\n", result)

	// Check the "success" field in the response
	if success, ok := result["success"].(bool); !ok || !success {
		log.Println("ERROR: reCAPTCHA verification failed")
		return errors.New("CAPTCHA verification failed")
	}

	// Log: CAPTCHA verification successful
	log.Println("INFO: CAPTCHA verification successful")
	return nil
}

// Function to generate OTP (stubbed here for simplicity)
func generateOTP() (string, error) {
	var otp string
	for i := 0; i < 6; i++ { // Generate 6 random digits
		num, err := rand.Int(rand.Reader, big.NewInt(10)) // Random number in range [0, 9]
		if err != nil {
			return "", errors.New("failed to generate a secure random number")
		}
		otp += num.String()
	}
	return otp, nil
}

// Handler function for signup
func SignUp(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	log.Println("INFO: Starting SignUp handler")

	var req SignupRequest
	log.Println("INFO: Binding request payload")
	if err := c.Bind(&req); err != nil {
		log.Println("ERROR: Invalid request payload:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	log.Printf("INFO: Received request: %+v\n", req)

	// Check if email domain is allowed
	log.Println("INFO: Checking email domain")
	emailDomain := getEmailDomain(req.Email)
	isAllowed := false
	for _, domain := range allowedDomains {
		if domain == emailDomain {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		log.Printf("ERROR: Email domain '%s' is not allowed\n", emailDomain)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Use a valid email"})
	}
	log.Println("INFO: Email domain is allowed")

	// Check for CAPTCHA
	log.Println("INFO: Verifying CAPTCHA")
	if req.Captcha == "" {
		log.Println("ERROR: CAPTCHA is missing")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Please complete the CAPTCHA"})
	}

	if err := verifyCaptcha(req.Captcha); err != nil {
		log.Println("ERROR: CAPTCHA verification failed:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	log.Println("INFO: CAPTCHA verification successful")

	// Generate and send OTP
	log.Println("INFO: Generating OTP")
	otp, err := generateOTP()
	if err != nil {
		log.Println("ERROR: Failed to generate OTP:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate OTP"})
	}

	log.Printf("INFO: Generated OTP: %s\n", otp)

	log.Println("INFO: Sending OTP to email")
	otpText := "Your OTP for registration is"
	subText := "OTP verification"
	if err := sendOTPByEmail(req.Email, otp, otpText, subText); err != nil {
		log.Println("ERROR: Failed to send OTP via email:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to send OTP"})
	}
	log.Println("INFO: OTP sent successfully")

	// Store OTP for the user in the database
	log.Println("INFO: Storing OTP in the database")
	if err := storeOTP(db, req.Email, otp); err != nil {
		log.Println("ERROR: Failed to store OTP:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to store OTP"})
	}
	log.Println("INFO: OTP stored successfully")

	// Respond with success
	log.Println("INFO: Returning success response")
	return c.JSON(http.StatusOK, echo.Map{
		"status":  "PENDING",
		"message": "Verification OTP sent.",
		"email":   req.Email,
	})
}

// Helper functions (stubbed for demonstration)

// Replace with actual DB lookup logic
func findUserByEmail(email string) (interface{}, error) {
	return nil, nil
}

// Replace with actual email sending logic
func sendOTPByEmail(email, otp, text, subject string) error {
	// SMTP credentials
	smtpHost := "smtp.gmail.com"
	smtpPort := 587

	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	// Create a new email message
	message := gomail.NewMessage()
	message.SetHeader("From", smtpUser)
	message.SetHeader("To", email)
	message.SetHeader("Subject", subject)
	message.SetBody("text/plain", text+": "+otp)
	dialer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

	log.Printf("INFO: Sending email to %s with subject: %s\n", email, subject)
	if err := dialer.DialAndSend(message); err != nil {
		log.Printf("ERROR: Failed to send email: %v\n", err)
		return err
	}
	log.Println("INFO: Email sent successfully")
	return nil
}

type OTP struct {
	Email     string    `bson:"email"`
	HashedOTP string    `bson:"otp"`
	ExpiresAt time.Time `bson:"expiresAt"`
}

const OTPExpirationTime = 5 * time.Minute

func hashOTP(otp string) string {
	hash := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(hash[:])
}

func storeOTP(db *mongo.Database, email string, otp string) error {
	otpCollection := models.InitializeOTPCollection(db)
	hashedOTP := hashOTP(otp)
	filter := bson.M{"email": email}

	// New OTP document to insert or update
	update := bson.M{
		"$set": bson.M{
			"email":     email,
			"otp":       otp,
			"hashedOTP": hashedOTP,
			"expiresAt": time.Now().Add(5 * time.Minute), // Adjust expiration time as needed
		},
	}

	// Upsert the document: update if it exists, insert if it doesn't
	opts := options.Update().SetUpsert(true)
	_, err := otpCollection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		log.Println("ERROR: Failed to upsert OTP:", err)
		return err
	}

	// Set a timeout to delete the OTP after expiration
	go func() {
		time.Sleep(5 * time.Minute) // Adjust timeout duration as needed
		_, err := otpCollection.DeleteOne(context.Background(), bson.M{"email": email})
		if err != nil {
			log.Printf("ERROR: Failed to delete expired OTP for email %s: %v", email, err)
		} else {
			log.Printf("INFO: OTP deleted after expiration for email: %s", email)
		}
	}()

	log.Printf("INFO: OTP successfully stored or updated for email: %s", email)
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

		trxPrivateKey, trxAddress, err := services.GenerateTronAddress()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate TRON wallet"})
		}

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

		token := generateJWT(newUser.Email, newUser.ID.Hex(), "google", trxAddress)
		return c.JSON(http.StatusOK, map[string]string{"token": token})
	}

	return c.JSON(http.StatusBadRequest, map[string]string{"error": "User already exists, Please Login."})
}
func Login(c echo.Context) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Captcha  string `json:"captcha"`
	}

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		log.Println("ERROR: Invalid request body:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Check CAPTCHA token
	if req.Captcha == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Please complete the CAPTCHA"})
	}

	// Verify CAPTCHA
	recaptchaSecretKey := os.Getenv("RECAPTCHA_SECRET_KEY")
	url := "https://www.google.com/recaptcha/api/siteverify?secret=" + recaptchaSecretKey + "&response=" + req.Captcha
	resp, err := http.Post(url, "application/json", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Println("ERROR: CAPTCHA verification failed:", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid CAPTCHA"})
	}
	defer resp.Body.Close()

	var captchaResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&captchaResult); err != nil {
		log.Println("ERROR: Failed to decode CAPTCHA response:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}
	if success, ok := captchaResult["success"].(bool); !ok || !success {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "CAPTCHA verification failed"})
	}

	// Database setup
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")
	walletCol := db.Collection("apikey_and_balances")

	// Define a struct to represent user data
	type LoginUser struct {
		ID       string `bson:"_id"`
		Email    string `bson:"email"`
		Password string `bson:"password"`
	}

	// Find the user by email
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var loginUser LoginUser
	err = userCol.FindOne(ctx, bson.M{"email": req.Email}).Decode(&loginUser)
	if err == mongo.ErrNoDocuments {
		log.Println("ERROR: User not found")
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	} else if err != nil {
		log.Println("ERROR: Database error while fetching user:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	if loginUser.Password != req.Password {
		log.Println("ERROR: Invalid credentials")
		c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
	}

	// Fetch the wallet information
	type WalletUser struct {
		UserID     string `bson:"userId"`
		TrxAddress string `bson:"trxAddress"`
	}

	var wallet WalletUser
	userID, err := primitive.ObjectIDFromHex(loginUser.ID)
	err = walletCol.FindOne(ctx, bson.M{"userId": userID}).Decode(&wallet)
	if err == mongo.ErrNoDocuments {
		log.Println("ERROR: Wallet not found for user:", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Wallet not found"})
	} else if err != nil {
		log.Println("ERROR: Database error while fetching wallet:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	// Generate JWT token
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	claims := jwt.MapClaims{
		"email":      loginUser.Email,
		"userId":     loginUser.ID,
		"trxAddress": wallet.TrxAddress,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		log.Println("ERROR: Failed to generate JWT token:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"})
	}

	log.Println("INFO: User logged in successfully")
	return c.JSON(http.StatusOK, map[string]string{"token": tokenString})
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

// ForgotPasswordRequest represents the request body for forgot password
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles the forgot password functionality
func ForgotPassword(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)
	otpCol := models.InitializeForgotOTPCollection(db)

	var request ForgotPasswordRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := request.Email
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}

	// Check if the user exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user bson.M
	err := userCol.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "User does not exist. Please sign up for an account.",
		})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Generate a new OTP
	newOtp, err := generateOTP()
	if err != nil {
		log.Println("ERROR: Failed to generate OTP:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate OTP"})
	}
	otpText := "Your OTP for Changing Password is"
	subText := "Forgot Password OTP verification"

	// Send OTP via email
	sendOtpResult := sendOTPByEmail(email, newOtp, otpText, subText)
	if sendOtpResult != nil { // Check if there was an error
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to send OTP"})
	}

	// Store OTP in the database
	otpData := models.OTP{
		ID:        primitive.NewObjectID(),
		Email:     email,
		OTP:       newOtp,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = otpCol.InsertOne(ctx, otpData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to store OTP"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "PENDING",
		"message": "Forgot Password OTP sent successfully.",
		"email":   email,
	})
}

func GenerateSecureOTP() string {
	bytes := make([]byte, 3) // 3 bytes for a 6-digit number
	_, err := rand.Read(bytes)
	if err != nil {
		fmt.Println("Error generating secure OTP:", err)
		return ""
	}
	otp := int(bytes[0])<<16 + int(bytes[1])<<8 + int(bytes[2])
	return fmt.Sprintf("%06d", otp%1000000)
}

// ResendForgotOTPRequest represents the request body for resending OTP
type ResendForgotOTPRequest struct {
	Email string `json:"email"`
}

// ResendForgotOTP handles the logic for resending the OTP
func ResendForgotOTP(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)
	otpCol := models.InitializeOTPCollection(db)

	var request ResendForgotOTPRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := request.Email
	if email == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email is required"})
	}

	// Check if the user exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user bson.M
	err := userCol.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "User does not exist"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Generate a new OTP
	newOtp := GenerateSecureOTP() // Ensure you have a GenerateOTP function
	otpText := "Your new OTP for Changing Password is"
	subText := "Forgot Password OTP verification"

	// Send the new OTP via email
	if err := sendOTPByEmail(email, newOtp, otpText, subText); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to send OTP"})
	}

	// Update the OTP in the database
	filter := bson.M{"email": email}
	update := bson.M{
		"$set": bson.M{
			"otp": newOtp,
			"ttl": time.Now().Add(15 * time.Minute), // Update TTL for new OTP
		},
	}

	_, err = otpCol.UpdateOne(ctx, filter, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update OTP in database"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "PENDING",
		"message": "New OTP sent successfully.",
		"email":   email,
	})
}

// ForgotVerifyOTPRequest represents the request body for verifying an OTP
type ForgotVerifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// ForgotVerifyOTP handles the OTP verification process
func ForgotVerifyOTP(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	otpCol := models.InitializeForgotOTPCollection(db) // Ensure this collection corresponds to where OTPs are stored

	var request ForgotVerifyOTPRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := request.Email
	otp := request.OTP

	if email == "" || otp == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email and OTP are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the OTP document for the provided email
	var otpDoc bson.M
	err := otpCol.FindOne(ctx, bson.M{"email": email}).Decode(&otpDoc)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "No OTP sent"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Retrieve the stored OTP from the document
	storedOTP, ok := otpDoc["otp"].(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Invalid OTP data format"})
	}

	// Compare the provided OTP with the stored OTP
	if storedOTP != otp {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid OTP"})
	}

	// OTP verified successfully
	return c.JSON(http.StatusOK, echo.Map{
		"status":  "VERIFIED",
		"message": "OTP verified successfully!",
	})
}

// ChangePasswordUnauthenticatedRequest represents the request body for changing a password
type ChangePasswordUnauthenticatedRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ChangePasswordUnauthenticated handles changing the password without authentication
func ChangePasswordUnauthenticated(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)
	otpCol := models.InitializeForgotOTPCollection(db)

	var request ChangePasswordUnauthenticatedRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	email := request.Email
	password := request.Password

	if email == "" || password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email and password are required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var otpDoc bson.M
	err := otpCol.FindOne(ctx, bson.M{"email": email}).Decode(&otpDoc)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "OTP not verified"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	_, err = otpCol.DeleteOne(ctx, bson.M{"email": email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete OTP document"})
	}

	update := bson.M{"$set": bson.M{"password": string(password)}}
	_, err = userCol.UpdateOne(ctx, bson.M{"email": email}, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update password"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "SUCCESS",
		"message": "Password changed successfully!",
	})
}

// ChangePasswordAuthenticatedRequest represents the request body for changing the password
type ChangePasswordAuthenticatedRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	UserID          string `json:"userId"`
	Captcha         string `json:"captcha"`
}

// ChangePasswordAuthenticated handles password change for authenticated users
func ChangePasswordAuthenticated(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")

	var request ChangePasswordAuthenticatedRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if request.Captcha == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Please complete the CAPTCHA"})
	}

	if request.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "please enter new pasword"})
	}

	// Verify CAPTCHA
	if err := verifyCaptcha(request.Captcha); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert UserID to ObjectId
	userID, err := primitive.ObjectIDFromHex(request.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid UserID format"})
	}

	// Find the user by UserID
	var user bson.M
	err = userCol.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	currentPassword := user["password"].(string)
	if currentPassword != request.CurrentPassword {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "current password doesn't match"})
	}

	_, err = userCol.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"password": request.NewPassword}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update password"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "SUCCESS",
		"message": "Password changed successfully!",
	})
}

// User represents the user structure
// User represents the structure of user data
type User struct {
	ID        string  `bson:"_id" json:"id"`
	Email     string  `bson:"email" json:"email"`
	Username  string  `bson:"username" json:"username"`
	CreatedAt string  `bson:"created_at" json:"created_at"`
	UpdatedAt string  `bson:"updated_at" json:"updated_at"`
	Balance   float64 `json:"balance"` // Ensure balance is of type float64
}

// GetUserBalance retrieves the balance of a user by their userId
func GetUserBalance(userID string, walletCol *mongo.Collection) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Structure to hold the wallet data
	var wallet struct {
		Balance float64 `bson:"balance"`
	}

	// Find the wallet document by userId
	err := walletCol.FindOne(ctx, bson.M{"userId": userID}).Decode(&wallet)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil // Return 0 if user is not found
		}
		return 0, err // Return the error for other cases
	}

	return wallet.Balance, nil
}

// GetAllUsers retrieves all users and includes their balances
func GetAllUsers(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")
	walletCol := db.Collection("apikey_and_balances")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Define the filter to fetch all users
	filter := bson.M{}
	projection := bson.M{"password": 0} // Exclude the password field

	// Set the find options to include the projection
	findOptions := options.Find().SetProjection(projection)

	cursor, err := userCol.Find(ctx, filter, findOptions)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch user data"})
	}
	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to parse user data"})
	}

	// Fetch balance for each user
	for i, user := range users {
		userID, ok := user["_id"].(primitive.ObjectID)
		if !ok {
			continue
		}

		var wallet bson.M
		if err := walletCol.FindOne(ctx, bson.M{"userId": userID}).Decode(&wallet); err == nil {
			if balance, ok := wallet["balance"].(float64); ok {
				users[i]["balance"] = balance
			} else {
				users[i]["balance"] = 0.0
			}
		} else {
			users[i]["balance"] = 0.0
		}
	}

	if len(users) == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "No user data"})
	}
	return c.JSON(http.StatusOK, users)
}

// GetUser fetches user details along with API wallet data
func GetUser(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")
	walletCol := db.Collection("apikey_and_balances")

	// Retrieve userId from query parameters
	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "userId is required"})
	}

	objID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user bson.M
	err = userCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch user data"})
	}

	var wallet bson.M
	err = walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&wallet)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "API wallet not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch API wallet data"})
	}

	userDataWithWallet := user
	if balance, ok := wallet["balance"]; ok {
		userDataWithWallet["balance"] = balance
	} else {
		userDataWithWallet["balance"] = 0.0
	}

	if apiKey, ok := wallet["api_key"]; ok {
		userDataWithWallet["api_key"] = apiKey
	}

	if trxAddress, ok := wallet["trxAddress"]; ok {
		userDataWithWallet["trxAddress"] = trxAddress
	}

	if trxPrivateKey, ok := wallet["trxPrivateKey"]; ok {
		userDataWithWallet["trxPrivateKey"] = trxPrivateKey
	}
	return c.JSON(http.StatusOK, userDataWithWallet)
}

// BlockUnblockUser handles blocking or unblocking a user
func BlockUnblockUser(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")

	// Define the request body structure
	type RequestBody struct {
		Blocked bool   `json:"blocked"`
		UserID  string `json:"userId"`
	}

	var body RequestBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	// Validate userId
	objID, err := primitive.ObjectIDFromHex(body.UserID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the user exists
	var user bson.M
	err = userCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Update the blocked status
	_, err = userCol.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": bson.M{"blocked": body.Blocked}})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update user status"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "SUCCESS",
		"message": "User saved successfully",
	})
}

// BlockedUser checks if a user is blocked
func BlockedUser(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")

	// Get userId from query parameters
	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "userId is required"})
	}

	// Validate the userId format
	objID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the user by ID
	var user bson.M
	err = userCol.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Check if the user is blocked
	blocked, ok := user["blocked"].(bool)
	if !ok {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Unable to determine blocked status"})
	}

	// Return appropriate response based on blocked status
	if blocked {
		return c.JSON(http.StatusOK, echo.Map{"message": "User is blocked"})
	} else {
		return c.JSON(http.StatusOK, echo.Map{"message": "User is not blocked"})
	}
}

// GetAllBlockedUsers retrieves all blocked users
// GetAllBlockedUsers retrieves all blocked users
func GetAllBlockedUsers(c echo.Context) error {
	// Retrieve the database instance
	db, ok := c.Get("db").(*mongo.Database)
	if !ok {
		log.Println("ERROR: Failed to retrieve database instance from context")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Access the "users" collection
	userCol := db.Collection("users")

	// Define context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query for all users with "blocked" set to true
	cursor, err := userCol.Find(ctx, bson.M{"blocked": true})
	if err != nil {
		log.Println("ERROR: Error querying blocked users:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Println("ERROR: Error closing cursor:", err)
		}
	}()

	// Parse the results into a slice of users
	var blockedUsers []bson.M
	if err := cursor.All(ctx, &blockedUsers); err != nil {
		log.Println("ERROR: Failed to parse blocked users:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to parse blocked users"})
	}

	// Handle case when no blocked users are found
	if len(blockedUsers) == 0 {
		log.Println("INFO: No blocked users found")
		return c.JSON(http.StatusOK, echo.Map{"data": []bson.M{}})
	}

	// Log and return the results
	log.Printf("INFO: Retrieved %d blocked users\n", len(blockedUsers))
	return c.JSON(http.StatusOK, echo.Map{"data": blockedUsers})
}

// GetOrdersByUserId retrieves orders by a specific user ID
// GetOrdersByUserId retrieves orders by a specific user ID
func GetOrdersByUserId(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	orderCol := models.InitializeOrderCollection(db)

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "userId is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert userId to ObjectID
	userPrimitiveId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid userId format"})
	}

	// Query to find orders for the userId and sort them by orderTime in descending order
	filter := bson.M{"userId": userPrimitiveId}
	opts := options.Find().SetSort(bson.D{{Key: "orderTime", Value: -1}})

	cursor, err := orderCol.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error fetching orders", "error": err.Error()})
	}
	defer cursor.Close(ctx)

	// Decode the cursor into a slice of orders
	var orders []bson.M
	if err := cursor.All(ctx, &orders); err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error decoding orders", "error": err.Error()})
	}

	// Explicitly handle case where no orders are found
	if len(orders) == 0 {
		return c.JSON(http.StatusOK, []bson.M{}) // Return empty array
	}

	return c.JSON(http.StatusOK, orders)
}

// VerifyOTP verifies the OTP and registers the user
func VerifyOTP(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)
	otpCol := models.InitializeOTPCollection(db)
	apiWalletCol := models.InitializeApiWalletuserCollection(db)

	type RequestBody struct {
		Email    string `json:"email"`
		OTP      string `json:"otp"`
		Password string `json:"password"`
	}

	var body RequestBody
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if the email already exists
	var existingUser models.User
	err := userCol.FindOne(ctx, bson.M{"email": body.Email}).Decode(&existingUser)
	if err == nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Email already exists"})
	} else if err != mongo.ErrNoDocuments {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Find the OTP document
	var otpDoc bson.M
	err = otpCol.FindOne(ctx, bson.M{"email": body.Email}).Decode(&otpDoc)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "OTP not found or expired"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Validate the OTP
	storedHashedOTP := otpDoc["hashedOTP"].(string) // OTP hash stored in the database
	inputHashedOTP := hashOTP(body.OTP)             // Hash the provided OTP using the same function
	if storedHashedOTP != inputHashedOTP {
		log.Printf("ERROR: Invalid OTP provided. Provided Hash: %s, Stored Hash: %s\n", inputHashedOTP, storedHashedOTP)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid OTP"})
	}

	// Delete the OTP document after validation
	_, err = otpCol.DeleteOne(ctx, bson.M{"email": body.Email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete OTP"})
	}

	// Create a new user
	newUser := models.User{
		ID:        primitive.NewObjectID(),
		Email:     body.Email,
		Password:  body.Password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = userCol.InsertOne(ctx, newUser)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to register user"})
	}

	// Generate API key and TRON wallet
	apiKey := generateAPIKey()
	trxPrivateKey, trxAddress, err := services.GenerateTronAddress()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate TRON wallet"})
	}

	// Create a new API wallet entry
	apiWallet := models.ApiWalletUser{
		UserID:        newUser.ID,
		APIKey:        apiKey,
		Balance:       0,
		TRXAddress:    trxAddress,
		TRXPrivateKey: trxPrivateKey,
	}
	_, err = apiWalletCol.InsertOne(ctx, apiWallet)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create wallet"})
	}

	// Respond with the original success response
	return c.JSON(http.StatusOK, echo.Map{
		"status":  "VERIFIED",
		"message": "User registered successfully",
	})
}

type EmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func ResendOTP(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	var req EmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email is required"})
	}

	email := req.Email
	// var existUser models.User
	// userCollection := models.InitializeUserCollection(db)
	// err := userCollection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&existUser)
	// if err == mongo.ErrEmptySlice || err == mongo.ErrNoDocuments {
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user not found"})
	// }
	// if err != nil {
	// 	logs.Logger.Error(err)
	// 	return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unable to Fetch user details"})
	// }

	newOtp := utils.GenerateOTP()
	otpText := "Your new OTP for registration is"
	subText := "OTP verification"
	if err := sendOTPByEmail(email, newOtp, otpText, subText); err != nil {
		log.Printf("Error sending OTP: %v\n", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to send OTP"})
	}

	err := storeOTP(db, email, newOtp)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to send OTP"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "PENDING",
		"message": "New OTP sent successfully.",
		"email":   email,
	})
}
