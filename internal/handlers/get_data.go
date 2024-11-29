package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"github.com/ranjankuldeep/fakeNumber/logs"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BalanceRequest struct {
	Url     string
	Headers map[string]string
}

type ServerCred struct {
	Token  string
	ApiKey string
}

type Balance struct {
	Value  float64
	Symbol string
}

func GetServersData(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	serverListCol := models.InitializeServerListCollection(db)

	sname := c.QueryParam("sname")
	if sname == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Service name is required."})
	}

	normalizedSname := strings.ToLower(sname)
	normalizedSname = strings.ReplaceAll(normalizedSname, " ", "")
	normalizedSname = strings.ReplaceAll(normalizedSname, "[^a-z0-9]", "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var data struct {
		Servers     []models.Server `bson:"servers"`
		LowestPrice string          `bson:"lowestPrice"`
	}
	err := serverListCol.FindOne(ctx, bson.M{"name": bson.M{"$regex": "^" + normalizedSname + "$", "$options": "i"}}).Decode(&data)
	if err == mongo.ErrNoDocuments {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Service not found."})
	} else if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Sort servers by serverNumber
	servers := data.Servers
	for i := 0; i < len(servers)-1; i++ {
		for j := i + 1; j < len(servers); j++ {
			if servers[i].ServerNumber > servers[j].ServerNumber {
				servers[i], servers[j] = servers[j], servers[i]
			}
		}
	}

	return c.JSON(http.StatusOK, servers)
}

// // Handler to retrieve and process service data for admin
// func GetServiceDataAdmin(c echo.Context) error {
// 	db := c.Get("db").(*mongo.Database)
// 	serverListCol := models.InitializeServerListCollection(db)
// 	serviceDiscountCol := models.InitializeServiceDiscountCollection(db)
// 	serverDiscountCol := models.InitializeServerDiscountCollection(db)

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	// Fetch server list data
// 	var serverListData []models.ServerList
// 	cursor, err := serverListCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch server list data"})
// 	}
// 	defer cursor.Close(ctx)

// 	if err := cursor.All(ctx, &serverListData); err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error decoding server list data"})
// 	}

// 	// Fetch service discount data
// 	var serviceDiscountData []models.ServiceDiscount
// 	serviceDiscountCursor, err := serviceDiscountCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch service discount data"})
// 	}
// 	defer serviceDiscountCursor.Close(ctx)

// 	if err := serviceDiscountCursor.All(ctx, &serviceDiscountData); err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error decoding service discount data"})
// 	}

// 	// Fetch server discount data
// 	var serverDiscountData []models.ServerDiscount
// 	serverDiscountCursor, err := serverDiscountCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch server discount data"})
// 	}
// 	defer serverDiscountCursor.Close(ctx)

// 	if err := serverDiscountCursor.All(ctx, &serverDiscountData); err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Error decoding server discount data"})
// 	}

// 	// Create a map for quick lookup of service discounts by service name and server number
// 	serviceDiscountMap := make(map[string]float64)
// 	for _, discount := range serviceDiscountData {
// 		key := discount.Service + "_" + strconv.Itoa(discount.Server)
// 		serviceDiscountMap[key] = discount.Discount
// 	}

// 	// Create a map for quick lookup of server discounts by server number
// 	serverDiscountMap := make(map[int]float64)
// 	for _, discount := range serverDiscountData {
// 		serverDiscountMap[discount.Server] = discount.Discount
// 	}

// 	// Sort the data by service name
// 	sort.SliceStable(serverListData, func(i, j int) bool {
// 		return serverListData[i].Name < serverListData[j].Name
// 	})

// 	// Sort servers within each service and apply discounts
// 	for i := range serverListData {
// 		service := &serverListData[i]

// 		// Sort servers by server number
// 		sort.SliceStable(service.Servers, func(a, b int) bool {
// 			return service.Servers[a].ServerNumber < service.Servers[b].ServerNumber
// 		})

// 		// Apply discounts to each server
// 		for j := range service.Servers {
// 			serverData := &service.Servers[j]
// 			serviceKey := service.Name + "_" + strconv.Itoa(serverData.ServerNumber)

// 			// Get service discount
// 			discount := serviceDiscountMap[serviceKey]

// 			// Add server discount if exists
// 			discount += serverDiscountMap[serverData.Server]

// 			// Update server price with discount
// 			originalPrice, _ := strconv.ParseFloat(serverData.Price, 64)
// 			serverData.Price = strconv.FormatFloat(originalPrice+discount, 'f', 2, 64)
// 		}
// 	}

// 	return c.JSON(http.StatusOK, serverListData)
// }

// GetServiceData handles the service data fetching
type ServiceResponse struct {
	Name    string         `json:"name"`
	Servers []ServerDetail `json:"servers"`
}

type ServerDetail struct {
	Server string `json:"server"`
	Price  string `json:"price"`
	Code   string `json:"code"`
	Otp    string `json:"otp"`
}

func GetServiceData(c echo.Context) error {
	userId := c.QueryParam("userId")
	db := c.Get("db").(*mongo.Database)

	serverCollection := models.InitializeServerCollection(db)
	serviceCollection := models.InitializeServerListCollection(db)
	serviceDiscountCollection := models.InitializeServiceDiscountCollection(db)
	serverDiscountCollection := models.InitializeServerDiscountCollection(db)
	userDiscountCollection := models.InitializeUserDiscountCollection(db)

	// Check site maintenance status
	var maintenanceStatus struct {
		Maintenance bool `bson:"maintainance"`
	}

	err := serverCollection.FindOne(context.Background(), bson.M{"server": 0}).Decode(&maintenanceStatus)
	if err == nil && maintenanceStatus.Maintenance {
		log.Println(err)
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	// Fetch servers in maintenance
	serversInMaintenance, err := serverCollection.Find(context.Background(), bson.M{"maintainance": true})
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer serversInMaintenance.Close(context.Background())

	// Extract server numbers in maintenance
	var maintenanceServerNumbers []int
	for serversInMaintenance.Next(context.Background()) {
		var server struct {
			ServerNumber int `bson:"server"`
		}
		if err := serversInMaintenance.Decode(&server); err == nil {
			log.Println(err)
			maintenanceServerNumbers = append(maintenanceServerNumbers, server.ServerNumber)
		}
	}

	// Fetch service data
	cursor, err := serviceCollection.Find(context.Background(), bson.D{})
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer cursor.Close(context.Background())

	var services []models.ServerList
	for cursor.Next(context.Background()) {
		var service models.ServerList
		if err := cursor.Decode(&service); err != nil {
			logs.Logger.Error(err)
		}
		services = append(services, service)
	}

	// Load discounts
	serviceDiscounts, serverDiscounts, userDiscounts, err := loadDiscounts(serviceDiscountCollection, serverDiscountCollection, userDiscountCollection, userId)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Process and filter data
	filteredData := []ServiceResponse{}
	for _, service := range services {
		serverDetails := []ServerDetail{}
		for _, server := range service.Servers {
			if contains(maintenanceServerNumbers, server.Server) {
				continue
			}
			// Calculate discounts
			discount := CalculateDiscount(serviceDiscounts, serverDiscounts, userDiscounts, service.Name, server.Server, userId)
			price, _ := strconv.ParseFloat(server.Price, 64)
			adjustedPrice := strconv.FormatFloat(price+discount, 'f', 2, 64) // final price

			serverDetails = append(serverDetails, ServerDetail{
				Server: strconv.Itoa(server.Server),
				Price:  adjustedPrice,
				Code:   server.Code,
				Otp:    server.Otp,
			})
		}

		// Sort by server number and add to filtered data
		sort.Slice(serverDetails, func(i, j int) bool {
			return serverDetails[i].Server < serverDetails[j].Server
		})
		filteredData = append(filteredData, ServiceResponse{
			Name:    service.Name,
			Servers: serverDetails,
		})
	}

	// Sort the data by service name
	sort.Slice(filteredData, func(i, j int) bool {
		return filteredData[i].Name < filteredData[j].Name
	})

	return c.JSON(http.StatusOK, filteredData)
}

// GetUserServiceData handles user-specific service data fetching
func GetUserServiceData(c echo.Context) error {
	userId := c.QueryParam("userId")
	apiKey := c.QueryParam("api_key")

	db := c.Get("db").(*mongo.Database)
	serviceDiscountCollection := db.Collection("serviceDiscount")
	serverDiscountCollection := db.Collection("serverDiscount")
	userDiscountCollection := db.Collection("userDiscount")

	// Validate API key
	var apiUser models.ApiWalletUser
	apiCollection := db.Collection("apiWalletUser")
	err := apiCollection.FindOne(context.Background(), bson.M{"api_key": apiKey}).Decode(&apiUser)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "BAD REQUEST"})
	}

	// Check site maintenance status
	var maintenanceStatus struct {
		Maintenance bool `bson:"maintainance"`
	}
	serverCollection := db.Collection("servers")
	err = serverCollection.FindOne(context.Background(), bson.M{"server": 0}).Decode(&maintenanceStatus)
	if err == nil && maintenanceStatus.Maintenance {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "Site is under maintenance."})
	}

	// Fetch servers in maintenance
	serversInMaintenance, err := serverCollection.Find(context.Background(), bson.M{"maintainance": true})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer serversInMaintenance.Close(context.Background())

	// Extract server numbers in maintenance
	var maintenanceServerNumbers []int
	for serversInMaintenance.Next(context.Background()) {
		var server struct {
			Server int `bson:"server"`
		}
		if err := serversInMaintenance.Decode(&server); err == nil {
			maintenanceServerNumbers = append(maintenanceServerNumbers, server.Server)
		}
	}

	// Fetch service data
	serviceCollection := db.Collection("serverList")
	cursor, err := serviceCollection.Find(context.Background(), bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}
	defer cursor.Close(context.Background())

	var services []models.ServerList
	for cursor.Next(context.Background()) {
		var service models.ServerList
		if err := cursor.Decode(&service); err == nil {
			services = append(services, service)
		}
	}

	// Load discounts
	serviceDiscounts, serverDiscounts, userDiscounts, err := loadDiscounts(serviceDiscountCollection, serverDiscountCollection, userDiscountCollection, userId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Internal server error"})
	}

	// Process and filter data
	filteredData := []ServiceResponse{}
	for _, service := range services {
		serverDetails := []ServerDetail{}
		for _, server := range service.Servers {
			if contains(maintenanceServerNumbers, server.Server) {
				continue
			}
			// Calculate discounts
			discount := CalculateDiscount(serviceDiscounts, serverDiscounts, userDiscounts, service.Name, server.Server, userId)
			price, _ := strconv.ParseFloat(server.Price, 64)
			adjustedPrice := strconv.FormatFloat(price+discount, 'f', 2, 64)

			serverDetails = append(serverDetails, ServerDetail{
				Server: strconv.Itoa(server.Server),
				Price:  adjustedPrice,
			})
		}

		// Sort by server number and add to filtered data
		sort.Slice(serverDetails, func(i, j int) bool {
			return serverDetails[i].Server < serverDetails[j].Server
		})
		filteredData = append(filteredData, ServiceResponse{
			Servers: serverDetails,
		})
	}

	return c.JSON(http.StatusOK, filteredData)
}

type ServiceServer struct {
	ServerNumber int    `bson:"serverNumber" json:"serverNumber"`
	Price        string `bson:"price" json:"price"`
}

type Service struct {
	Name        string          `bson:"name" json:"name"`
	LowestPrice string          `bson:"lowestPrice" json:"lowestPrice"`
	Servers     []ServiceServer `bson:"servers" json:"servers"`
}

type Discount struct {
	Service  string  `bson:"service" json:"service"`
	Server   int     `bson:"server" json:"server"`
	Discount float64 `bson:"discount" json:"discount"`
}

func GetServiceDataAdmin(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)

	// Logger: Start of the function
	log.Println("INFO: Fetching service data for admin")

	// Fetch the server list data
	var serverListData []Service
	serverListCol := db.Collection("serverlists")

	log.Println("INFO: Fetching server list data...")
	findOptions := options.Find().SetProjection(bson.M{
		"name":        1,
		"lowestPrice": 1,
		"servers":     1,
	})

	cursor, err := serverListCol.Find(context.Background(), bson.M{}, findOptions)
	if err != nil {
		log.Println("ERROR: Failed to fetch server list data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch server list data"})
	}
	if err := cursor.All(context.Background(), &serverListData); err != nil {
		log.Println("ERROR: Failed to decode server list data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to decode server list data"})
	}
	if len(serverListData) == 0 {
		log.Println("INFO: No server list data found")
		return c.JSON(http.StatusOK, []Service{}) // Return empty array
	}
	log.Println("INFO: Successfully fetched server list data")

	// Fetch the service discount data
	var serviceDiscountData []Discount
	serviceDiscountCol := db.Collection("serviceDiscounts")

	log.Println("INFO: Fetching service discount data...")
	cursor, err = serviceDiscountCol.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("ERROR: Failed to fetch service discount data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch service discount data"})
	}
	if err := cursor.All(context.Background(), &serviceDiscountData); err != nil {
		log.Println("ERROR: Failed to decode service discount data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to decode service discount data"})
	}
	log.Println("INFO: Successfully fetched service discount data")

	// Fetch the server discount data
	var serverDiscountData []Discount
	serverDiscountCol := db.Collection("serverDiscounts")

	log.Println("INFO: Fetching server discount data...")
	cursor, err = serverDiscountCol.Find(context.Background(), bson.M{})
	if err != nil {
		log.Println("ERROR: Failed to fetch server discount data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch server discount data"})
	}
	if err := cursor.All(context.Background(), &serverDiscountData); err != nil {
		log.Println("ERROR: Failed to decode server discount data:", err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to decode server discount data"})
	}
	log.Println("INFO: Successfully fetched server discount data")

	// Create maps for discounts
	log.Println("INFO: Creating discount maps...")
	serviceDiscountMap := make(map[string]float64)
	for _, discount := range serviceDiscountData {
		key := discount.Service + "_" + strconv.Itoa(discount.Server)
		serviceDiscountMap[key] = discount.Discount
	}

	serverDiscountMap := make(map[int]float64)
	for _, discount := range serverDiscountData {
		serverDiscountMap[discount.Server] = discount.Discount
	}
	log.Println("INFO: Discount maps created")

	log.Println("INFO: Sorting server list data...")
	sort.Slice(serverListData, func(i, j int) bool {
		return serverListData[i].Name < serverListData[j].Name
	})

	for i, service := range serverListData {
		sort.Slice(service.Servers, func(a, b int) bool {
			return service.Servers[a].ServerNumber < service.Servers[b].ServerNumber
		})

		for j, server := range service.Servers {
			serviceKey := service.Name + "_" + strconv.Itoa(server.ServerNumber)
			discount := serviceDiscountMap[serviceKey] + serverDiscountMap[server.ServerNumber]
			originalPrice, err := strconv.ParseFloat(server.Price, 64)
			if err != nil {
				log.Printf("ERROR: Invalid price format for service %s, server %d: %v\n", service.Name, server.ServerNumber, err)
				continue
			}
			finalPrice := originalPrice + discount
			serverListData[i].Servers[j].Price = strconv.FormatFloat(finalPrice, 'f', 2, 64)
		}
	}
	return c.JSON(http.StatusOK, serverListData)
}

// Helper functions
func contains(arr []int, num int) bool {
	for _, n := range arr {
		if n == num {
			return true
		}
	}
	return false
}

// loadDiscounts loads discount data for services, servers, and users
func loadDiscounts(serviceDiscountCollection, serverDiscountCollection, userDiscountCollection *mongo.Collection, userId string) (map[string]float64, map[int]float64, map[string]float64, error) {
	// Load service discounts
	serviceDiscounts := make(map[string]float64)
	serviceCursor, _ := serviceDiscountCollection.Find(context.Background(), bson.M{})
	defer serviceCursor.Close(context.Background())
	for serviceCursor.Next(context.Background()) {
		var discount models.ServiceDiscount
		if err := serviceCursor.Decode(&discount); err == nil {
			key := discount.Service + "_" + strconv.Itoa(discount.Server)
			serviceDiscounts[key] = discount.Discount
		}
	}

	// Load server discounts
	serverDiscounts := make(map[int]float64)
	serverCursor, _ := serverDiscountCollection.Find(context.Background(), bson.M{})
	defer serverCursor.Close(context.Background())
	for serverCursor.Next(context.Background()) {
		var discount models.ServerDiscount
		if err := serverCursor.Decode(&discount); err == nil {
			serverDiscounts[discount.Server] = discount.Discount
		}
	}

	// Load user discounts if userId is provided
	userIdObject, _ := primitive.ObjectIDFromHex(userId)
	userDiscounts := make(map[string]float64)
	if userId != "" {
		userCursor, _ := userDiscountCollection.Find(context.Background(), bson.M{"userId": userIdObject})
		defer userCursor.Close(context.Background())
		for userCursor.Next(context.Background()) {
			var discount models.UserDiscount
			if err := userCursor.Decode(&discount); err == nil {
				key := discount.Service + "_" + strconv.Itoa(discount.Server)
				userDiscounts[key] = float64(discount.Discount)
			}
		}
	}
	return serviceDiscounts, serverDiscounts, userDiscounts, nil
}

func CalculateDiscount(serviceDiscounts map[string]float64, serverDiscounts map[int]float64, userDiscounts map[string]float64, serviceName string, serverNumber int, userId string) float64 {
	key := serviceName + "_" + strconv.Itoa(serverNumber)
	return serviceDiscounts[key] + serverDiscounts[serverNumber] + userDiscounts[key]
}

// Handler to calculate total recharge amount
func TotalRecharge(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	rechargeHistoryCol := models.InitializeRechargeHistoryCollection(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := rechargeHistoryCol.Find(ctx, bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch recharge history"})
	}
	defer cursor.Close(ctx)

	var totalAmount float64
	var histories []models.RechargeHistory
	cursor.All(ctx, &histories)
	for _, history := range histories {
		amount, _ := strconv.ParseFloat(history.Amount, 64)
		totalAmount += amount
	}

	return c.JSON(http.StatusOK, echo.Map{"totalAmount": strconv.FormatFloat(totalAmount, 'f', 2, 64)})
}

// Handler to retrieve total user count
func GetTotalUserCount(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	userCol := models.InitializeUserCollection(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userCount, err := userCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error fetching user count"})
	}

	return c.JSON(http.StatusOK, echo.Map{"totalUserCount": userCount})
}

func GetServerBalanceHandler(c echo.Context) error {
	db := c.Get("db").(*mongo.Database)
	server := c.QueryParam("server")

	balance, err := GetServerBalance(db, server)
	if err != nil {
		logs.Logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "Error fetching server balance"})
	}
	return c.JSON(http.StatusOK, echo.Map{"balance": fmt.Sprintf("%0.2f%s", balance.Value, balance.Symbol)})
}

func GetServerBalance(db *mongo.Database, server string) (Balance, error) {
	serverNumber, _ := strconv.Atoi(server)
	var serverInfo models.Server
	serverCollection := models.InitializeServerCollection(db)
	err := serverCollection.FindOne(context.TODO(), bson.M{"server": serverNumber}).Decode(&serverInfo)
	if err != nil {
		logs.Logger.Error(err)
		return Balance{}, err
	}

	cred := ServerCred{
		Token:  serverInfo.Token,
		ApiKey: serverInfo.APIKey,
	}

	balanceFetchUrl, err := ConstructBalanceUrl(server, cred)
	if err != nil {
		logs.Logger.Error(err)
		return Balance{}, err
	}
	balance, err := FetchBalance(server, balanceFetchUrl.Url, balanceFetchUrl.Headers)
	if err != nil {
		return Balance{}, err
	}
	return balance, nil
}

func ConstructBalanceUrl(server string, serverCred ServerCred) (BalanceRequest, error) {
	switch server {
	case "1":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "2":
		return BalanceRequest{
			Url: fmt.Sprintf("https://5sim.net/v1/user/profile"),
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", serverCred.Token),
				"Accept":        "application/json",
			},
		}, nil
	case "3":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "4":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "5":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "6":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://tempnum.org/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "7":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "8":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://api.sms-activate.guru/stubs/handler_api.php?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "9":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://sms-activation-service.pro/stubs/handler_api?api_key=%s&action=getBalance", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	case "10":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://own5k.in/p/ccpay.php?type=balance"),
			Headers: map[string]string{},
		}, nil
	case "11":
		return BalanceRequest{
			Url:     fmt.Sprintf("https://api.sms-man.com/control/get-balance?token=%s", serverCred.ApiKey),
			Headers: map[string]string{},
		}, nil
	}
	return BalanceRequest{}, nil
}

func FetchBalance(server string, apiURL string, headers map[string]string) (Balance, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to create balance request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to read response body: %w", err)
	}
	responseData := string(body)
	logs.Logger.Info(responseData)
	if strings.TrimSpace(responseData) == "" {
		return Balance{}, errors.New("RECEIVED_EMPTY_RESPONSE_FROM_THIRD_PARTY_SERVER")
	}

	switch server {
	case "1":
		balance := strings.TrimPrefix(responseData, "ACCESS_BALANCE:")
		value, err := strconv.ParseFloat(balance, 64)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse balance: %w", err)
		}
		return Balance{Value: value, Symbol: "p"}, nil

	case "2":
		var responseDataJSON struct {
			Balance float64 `json:"balance"`
		}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse JSON response for balance: %w", err)
		}
		return Balance{Value: responseDataJSON.Balance, Symbol: "p"}, nil

	case "3":
		balance := strings.TrimPrefix(responseData, "ACCESS_BALANCE:")
		value, err := strconv.ParseFloat(balance, 64)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse balance: %w", err)
		}
		return Balance{Value: value, Symbol: "$"}, nil

	case "4", "5", "6", "7", "8":
		balance := strings.TrimPrefix(responseData, "ACCESS_BALANCE:")
		value, err := strconv.ParseFloat(balance, 64)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse balance: %w", err)
		}
		return Balance{Value: value, Symbol: "p"}, nil

	case "9":
		value, err := strconv.ParseFloat(responseData, 64)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse balance: %w", err)
		}
		return Balance{Value: value, Symbol: "p"}, nil

	case "10":
		value, err := strconv.ParseFloat(responseData, 64)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse balance: %w", err)
		}
		return Balance{Value: value, Symbol: "$"}, nil

	case "11":
		var responseDataJSON struct {
			Balance string `json:"balance"`
		}
		err = json.Unmarshal(body, &responseDataJSON)
		if err != nil {
			return Balance{}, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		floatValue, _ := strconv.ParseFloat(responseDataJSON.Balance, 64)
		return Balance{Value: floatValue, Symbol: "p"}, nil

	default:
		return Balance{}, errors.New("INVALID_SERVER_VALUE")
	}
}
