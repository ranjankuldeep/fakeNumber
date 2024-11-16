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
	"github.com/ranjankuldeep/fakeNumber/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
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
		// apiKey := generateAPIKey()

		// wallet, err := lib.GenerateTronAddress()
		// if err != nil {
		// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate TRON wallet"})
		// }

		// // todo
		// trxAddress := wallet["address"]
		// trxPrivateKey := wallet["privateKey"]

		// apiWallet := models.ApiWalletUser{
		// 	UserID:        newUser.ID,
		// 	APIKey:        apiKey,
		// 	Balance:       0,
		// 	TRXAddress:    trxAddress,
		// 	TRXPrivateKey: trxPrivateKey,
		// }

		// apiWalletColl := models.InitializeApiWalletuserCollection(db)
		// _, err = apiWalletColl.InsertOne(context.TODO(), apiWallet)
		// if err != nil {
		// 	return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create wallet"})
		// }

		// Generate JWT token
		// token := generateJWT(newUser.Email, newUser.ID.Hex(), "google", trxAddress)
		// return c.JSON(http.StatusOK, map[string]string{"token": token})

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

// ForgotPasswordRequest represents the request body for forgot password
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// OTP represents the structure of the OTP document
type OTP struct {
	Email string    `bson:"email"` // Email associated with the OTP
	OTP   string    `bson:"otp"`   // The generated OTP
	TTL   time.Time `bson:"ttl"`   // Time-to-live for the OTP
}

// ForgotPassword handles the forgot password functionality
func ForgotPassword(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")
	otpCol := db.Collection("otp")

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
	newOtp := generateOTP()
	otpText := "Your OTP for Changing Password is"
	subText := "Forgot Password OTP verification"

	// Send OTP via email
	sendOtpResult := sendOTPByEmail(email, newOtp, otpText, subText)
	if sendOtpResult != nil { // Check if there was an error
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to send OTP"})
	}

	// Store OTP in the database
	otpData := OTP{
		Email: email,
		OTP:   newOtp,
		TTL:   time.Now().Add(15 * time.Minute), // OTP valid for 15 minutes
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
	userCol := db.Collection("users")
	otpCol := db.Collection("otp")

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
	otpCol := db.Collection("otp") // Ensure this collection corresponds to where OTPs are stored

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

	// Retrieve the stored hashed OTP from the document
	storedHashedOTP, ok := otpDoc["otp"].(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Invalid OTP data format"})
	}

	// Compare the provided OTP with the stored hashed OTP
	err = bcrypt.CompareHashAndPassword([]byte(storedHashedOTP), []byte(otp))
	if err != nil {
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
	userCol := db.Collection("users")
	otpCol := db.Collection("otp")

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

	// Find the OTP document for the provided email
	var otpDoc bson.M
	err := otpCol.FindOne(ctx, bson.M{"email": email}).Decode(&otpDoc)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "OTP not verified"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Delete the OTP document
	_, err = otpCol.DeleteOne(ctx, bson.M{"email": email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete OTP document"})
	}

	// Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash password"})
	}

	// Update the user document with the new password
	update := bson.M{"$set": bson.M{"password": string(newHashedPassword)}}
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

	// Verify CAPTCHA
	if err := verifyCaptcha(request.Captcha); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the user by UserID
	var user bson.M
	err := userCol.FindOne(ctx, bson.M{"_id": request.UserID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Database error"})
	}

	// Compare the provided current password with the stored hashed password
	currentPassword := user["password"].(string)
	err = bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(request.CurrentPassword))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Current password is incorrect"})
	}

	// Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash password"})
	}

	// Update the user's password
	filter := bson.M{"_id": request.UserID}
	update := bson.M{"$set": bson.M{"password": string(newHashedPassword)}}
	_, err = userCol.UpdateOne(ctx, filter, update, options.Update().SetUpsert(false))
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
	walletCol := db.Collection("wallets")

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
	walletCol := db.Collection("wallets")

	// Retrieve userId from query parameters
	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "userId is required"})
	}

	// Convert userId to ObjectID
	objID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid userId format"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch user data excluding the password field
	var user bson.M
	projection := bson.M{"password": 0}
	err = userCol.FindOne(ctx, bson.M{"_id": objID}, options.FindOne().SetProjection(projection)).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch user data"})
	}

	// Fetch user's API wallet data
	var wallet bson.M
	err = walletCol.FindOne(ctx, bson.M{"userId": objID}).Decode(&wallet)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "API wallet not found"})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch API wallet data"})
	}

	// Combine user data with wallet data
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
func GetAllBlockedUsers(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := db.Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query for all users with "blocked" set to true
	cursor, err := userCol.Find(ctx, bson.M{"blocked": true})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "No blocked users found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer cursor.Close(ctx)

	// Parse the results into a slice of documents
	var blockedUsers []bson.M
	if err := cursor.All(ctx, &blockedUsers); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to parse blocked users"})
	}

	return c.JSON(http.StatusOK, echo.Map{"data": blockedUsers})
}

// GetOrdersByUserId retrieves orders by a specific user ID
func GetOrdersByUserId(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	orderCol := db.Collection("orders")

	userId := c.QueryParam("userId")
	if userId == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "userId is required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Query to find orders for the userId and sort them by orderTime in descending order
	filter := bson.M{"userId": userId}
	opts := options.Find().SetSort(bson.D{{Key: "orderTime", Value: -1}})

	cursor, err := orderCol.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error fetching orders", "error": err.Error()})
	}
	defer cursor.Close(ctx)

	// Decode the cursor into a slice of orders
	var orders []bson.M
	if err := cursor.All(ctx, &orders); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error decoding orders", "error": err.Error()})
	}

	return c.JSON(http.StatusOK, orders)
}

// VerifyOTP verifies the OTP and registers the user
func VerifyOTP(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)
	otpCol := db.Collection("otp")
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
	hashedOTP := otpDoc["otp"].(string)
	if err := bcrypt.CompareHashAndPassword([]byte(hashedOTP), []byte(body.OTP)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid OTP"})
	}

	// Delete the OTP document
	_, err = otpCol.DeleteOne(ctx, bson.M{"email": body.Email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete OTP"})
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to hash password"})
	}

	// Create a new user
	newUser := models.User{
		ID:        primitive.NewObjectID(),
		Email:     body.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = userCol.InsertOne(ctx, newUser)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to register user"})
	}

	// Generate API key
	apiKeyBytes := make([]byte, 16)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate API key"})
	}
	apiKey := hex.EncodeToString(apiKeyBytes)

	// Generate TRON wallet
	wallet, err := lib.GenerateTronAddress()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to generate TRON wallet"})
	}
	trxAddress := wallet["address"]
	trxPrivateKey := wallet["privateKey"]

	// Create API wallet user
	apiWallet := models.ApiWalletUser{
		UserID:        newUser.ID,
		APIKey:        apiKey,
		Balance:       0,
		TRXAddress:    trxAddress,
		TRXPrivateKey: trxPrivateKey,
	}

	_, err = apiWalletCol.InsertOne(ctx, apiWallet)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to create API wallet"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  "VERIFIED",
		"message": "User registered successfully",
	})
}
