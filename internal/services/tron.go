package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ranjankuldeep/fakeNumber/internal/utils"
)

type ResponseData struct {
	EncryptedPrivateKey string `json:"encryptedPrivateKey"`
	IV                  string `json:"iv"`
	Address             string `json:"address"`
}

func GenerateTronAddress() (string, string, error) {
	baseTronServerUrl := os.Getenv("TRON_SERVER_BASE_URL")
	expressServerURL := fmt.Sprintf("%s/api/internal/generate-tron-address", baseTronServerUrl) // Replace with your actual Express server URL

	// Example payload for the request (modify as needed)
	payload := map[string]string{
		"clientId": "example-client-id", // Example of extra data (modify or remove if not required)
	}

	// Convert payload to JSON
	requestBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return "", "", errors.New("cannot marshal the payload")
	}

	// Make the HTTP POST request
	resp, err := http.Post(expressServerURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return "", "", errors.New("Error making post request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Received non-OK response: %d\n", resp.StatusCode)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("Response body:", string(body))
		return "", "", errors.New("RESPONSE STATUS INVALID")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", "", errors.New("EMPTY RESPONSE BODY")
	}

	var responseData ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		fmt.Println("Error unmarshaling response body:", err)
		return "", "", errors.New("CANNOT UNMARSHAL RESPONSE")
	}

	secretKey := os.Getenv("TRON_SECRET_KEY")
	privateKey := responseData.EncryptedPrivateKey
	walletAddress := responseData.Address
	vectorHash := responseData.IV

	decryptedPrivateKey, err := utils.Decrypt(privateKey, vectorHash, secretKey)
	if err != nil {
		return "", "", err
	}
	return decryptedPrivateKey, walletAddress, nil
}
