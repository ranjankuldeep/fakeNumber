package handlers

// // Define URLs
// var urls = []string{
// 	"https://php.paidsms.in/p/fastsms.txt",
// 	"https://php.paidsms.in/p/5sim.txt",
// 	"https://php.paidsms.in/p/smshub.txt",
// 	"https://php.paidsms.in/p/tigersms.txt",
// 	"https://php.paidsms.in/p/grizzlysms.txt",
// 	"https://php.paidsms.in/p/tempnumber.txt",
// 	"https://php.paidsms.in/p/smsmansingle.txt",
// 	"https://php.paidsms.in/p/smsmanmulti.txt",
// 	"https://php.paidsms.in/p/cpay.txt",
// }

// // Map for calculating prices based on the URL
// var priceCalculations = map[string]func(float64) float64{
// 	"https://php.paidsms.in/p/fastsms.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/5sim.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/smshub.txt": func(price float64) float64 {
// 		return (price*95 + 1)
// 	},
// 	"https://php.paidsms.in/p/tigersms.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/grizzlysms.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/tempnumber.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/smsmansingle.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/smsmanmulti.txt": func(price float64) float64 {
// 		return (price*1.3 + 1)
// 	},
// 	"https://php.paidsms.in/p/cpay.txt": func(price float64) float64 {
// 		return (price*95 + 1)
// 	},
// }

// // Function to get the server number from the URL
// func getServerNumberFromUrl(url string) int {
// 	urlToServerMap := map[string]int{
// 		"https://php.paidsms.in/p/fastsms.txt":      1,
// 		"https://php.paidsms.in/p/5sim.txt":         2,
// 		"https://php.paidsms.in/p/smshub.txt":       3,
// 		"https://php.paidsms.in/p/tigersms.txt":     4,
// 		"https://php.paidsms.in/p/grizzlysms.txt":   5,
// 		"https://php.paidsms.in/p/tempnumber.txt":   6,
// 		"https://php.paidsms.in/p/smsmansingle.txt": 7,
// 		"https://php.paidsms.in/p/smsmanmulti.txt":  8,
// 		"https://php.paidsms.in/p/cpay.txt":         9,
// 	}
// 	return urlToServerMap[url]
// }

// type BlockUnblockRequest struct {
// 	Name         string `json:"name" validate:"required"`
// 	ServerNumber int    `json:"serverNumber" validate:"required"`
// 	Block        bool   `json:"block" validate:"required"`
// }

// // fetchDataWithRetry attempts to fetch data from a URL, retrying indefinitely if an error occurs.
// func FetchDataWithRetry(url string, delay time.Duration) ([]interface{}, error) {
// 	attempts := 0

// 	for {
// 		attempts++
// 		fmt.Printf("Attempt %d to fetch data from %s\n", attempts, url)

// 		resp, err := http.Get(url)
// 		if err != nil {
// 			fmt.Printf("Error fetching data from %s (attempt %d): %v\n", url, attempts, err)
// 		} else {
// 			defer resp.Body.Close()
// 			if resp.StatusCode != http.StatusOK {
// 				fmt.Printf("HTTP error! Status: %d\n", resp.StatusCode)
// 			} else {
// 				// Read and parse JSON data
// 				body, readErr := ioutil.ReadAll(resp.Body)
// 				if readErr != nil {
// 					fmt.Printf("Error reading response body: %v\n", readErr)
// 				} else {
// 					var data []interface{}
// 					jsonErr := json.Unmarshal(body, &data)
// 					if jsonErr != nil {
// 						fmt.Printf("Error parsing JSON data: %v\n", jsonErr)
// 					} else if len(data) > 0 {
// 						fmt.Printf("Data fetched successfully from %s\n", url)
// 						return data, nil
// 					} else {
// 						fmt.Printf("No data found at %s\n", url)
// 					}
// 				}
// 			}
// 		}

// 		// Wait before retrying
// 		time.Sleep(delay)
// 	}
// }

// // normalizeName normalizes the input string by converting it to lowercase,
// // removing whitespace, and keeping only alphanumeric characters.
// func normalizeName(name string) string {
// 	// Convert to lowercase, remove whitespace, and keep only alphanumeric characters
// 	re := regexp.MustCompile(`[^a-z0-9]`)
// 	return re.ReplaceAllString(strings.ToLower(strings.ReplaceAll(name, " ", "")), "")
// }

// // stringSimilarity calculates a simple similarity metric based on character match (adjustable as needed).
// func stringSimilarity(s1, s2 string) float64 {
// 	matches := 0
// 	for i := 0; i < len(s1) && i < len(s2); i++ {
// 		if s1[i] == s2[i] {
// 			matches++
// 		}
// 	}
// 	return float64(matches) / float64(math.Max(float64(len(s1)), float64(len(s2))))
// }

// // findExistingItem searches for the best matching item by name in the ServerList collection.
// func findExistingItem(ctx context.Context, db *mongo.Database, name string) (bestMatch interface{}, highestSimilarity float64) {
// 	serverListCol := db.Collection("server_list")
// 	cursor, err := serverListCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		log.Fatalf("Failed to find items: %v", err)
// 	}
// 	defer cursor.Close(ctx)

// 	normalizedName := normalizeName(name)
// 	highestSimilarity = 0.0

// 	for cursor.Next(ctx) {
// 		var item bson.M
// 		cursor.Decode(&item)

// 		itemName := item["name"].(string)
// 		similarity := stringSimilarity(normalizedName, normalizeName(itemName))
// 		if similarity > highestSimilarity {
// 			highestSimilarity = similarity
// 			bestMatch = item
// 		}
// 	}

// 	return bestMatch, highestSimilarity
// }

// // pricesAreEqual checks if two prices are equal within a specified tolerance.
// func pricesAreEqual(price1, price2, tolerance float64) bool {
// 	return math.Abs(price1-price2) < tolerance
// }

// // calculateLowestPrice finds the lowest price from a list of server prices.
// func calculateLowestPrice(servers []interface{}) string {
// 	lowest := math.MaxFloat64

// 	for _, s := range servers {
// 		server := s.(bson.M)
// 		priceStr := server["price"].(string)
// 		price, err := strconv.ParseFloat(priceStr, 64)
// 		if err == nil && price < lowest {
// 			lowest = price
// 		}
// 	}

// 	return fmt.Sprintf("%.2f", lowest)
// }

// // calculateLowestPrices updates each server document with its lowest price.
// func calculateLowestPrices(ctx context.Context, db *mongo.Database) {
// 	serverListCol := db.Collection("server_list")
// 	cursor, err := serverListCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		log.Fatalf("Failed to find server list: %v", err)
// 	}
// 	defer cursor.Close(ctx)

// 	for cursor.Next(ctx) {
// 		var server bson.M
// 		cursor.Decode(&server)

// 		servers := server["servers"].(primitive.A)
// 		lowestPrice := calculateLowestPrice(servers)
// 		filter := bson.M{"_id": server["_id"]}
// 		update := bson.M{"$set": bson.M{"lowestPrice": lowestPrice}}

// 		_, err = serverListCol.UpdateOne(ctx, filter, update)
// 		if err != nil {
// 			log.Printf("Failed to update lowest price for server %v: %v", server["_id"], err)
// 		}
// 	}

// 	log.Println("Lowest prices updated successfully")
// }

// // calculatePrice applies a multiplier and offset to the provided price.
// func calculatePrice(price string) string {
// 	multiplier := 95.0
// 	offset := 1.0

// 	numericPrice, err := strconv.ParseFloat(price, 64)
// 	if err != nil {
// 		log.Printf("Invalid price value: %s", price)
// 		return price
// 	}

// 	calculatedPrice := numericPrice*multiplier + offset
// 	return fmt.Sprintf("%.2f", calculatedPrice)
// }

// func saveServerDataOnce(c echo.Context) error {
// 	go processServerData(context.Background(), db) // Run data processing in the background

// 	return c.JSON(http.StatusOK, map[string]string{
// 		"message": "Data fetching and saving has started. It will continue in the background.",
// 	})
// }

// // Main function to fetch, process, and save server data
// func processServerData(ctx context.Context, db *mongo.Database) {
// 	urls := []string{
// 		"https://php.paidsms.in/p/fastsms.txt",
// 		"https://php.paidsms.in/p/5sim.txt",
// 		// Add other URLs as needed
// 	}

// 	for _, url := range urls {
// 		data, err := fetchDataWithRetry(url)
// 		if err != nil || data == nil {
// 			log.Printf("Skipping URL due to no data: %s\n", url)
// 			continue
// 		}

// 		serverNumber := getServerNumberFromUrl(url)
// 		server := fetchServerConfig(ctx, db, serverNumber)
// 		if server == nil {
// 			log.Printf("No server configuration found for %s\n", url)
// 			continue
// 		}

// 		for _, item := range data {
// 			itemPrice, _ := strconv.ParseFloat(item["price"].(string), 64)
// 			adjustedPrice := calculatePrice(itemPrice, server.ExchangeRate, server.Margin)

// 			existingItem, similarity := findExistingItem(ctx, db, item["name"].(string))
// 			if existingItem != nil && similarity > 0.8 {
// 				updateServerItem(ctx, db, existingItem, item, adjustedPrice)
// 			} else {
// 				saveNewItem(ctx, db, item, adjustedPrice)
// 			}
// 		}

// 		log.Printf("Data from %s saved to the database successfully.\n", url)
// 		time.Sleep(1 * time.Second)
// 	}

// 	mergeDuplicates(ctx, db)
// 	updateServiceCodes(ctx, db)
// 	log.Println("Background data fetched and saved successfully")
// }

// // Route handler to check for duplicates in the server list
// func checkDuplicates(c echo.Context) error {
// 	duplicates, err := findDuplicates(context.Background(), db.Collection("server_list"))
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
// 	}

// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"serverListDuplicates": duplicates,
// 	})
// }

// // Function to merge duplicate documents in MongoDB
// func mergeDuplicates(ctx context.Context, db *mongo.Database) {
// 	duplicates, err := findDuplicates(ctx, db.Collection("server_list"))
// 	if err != nil {
// 		log.Printf("Error finding duplicates: %v\n", err)
// 		return
// 	}

// 	for _, name := range duplicates {
// 		var duplicateDocs []ServerList
// 		cursor, _ := db.Collection("server_list").Find(ctx, bson.M{"name": name})
// 		cursor.All(ctx, &duplicateDocs)
// 		if len(duplicateDocs) > 1 {
// 			masterDoc := duplicateDocs[0]
// 			docsToMerge := duplicateDocs[1:]

// 			for _, doc := range docsToMerge {
// 				masterDoc.Servers = append(masterDoc.Servers, doc.Servers...)
// 				db.Collection("server_list").DeleteOne(ctx, bson.M{"_id": doc.ID})
// 			}

// 			uniqueServers := uniqueServers(masterDoc.Servers)
// 			masterDoc.Servers = uniqueServers
// 			db.Collection("server_list").ReplaceOne(ctx, bson.M{"_id": masterDoc.ID}, masterDoc)
// 		}
// 	}
// 	log.Println("Duplicates merged successfully")
// }

// // Function to update service codes in each server document
// func updateServiceCodes(ctx context.Context, db *mongo.Database) {
// 	serverListCol := db.Collection("server_list")
// 	cursor, err := serverListCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		log.Printf("Error fetching server list for updating service codes: %v\n", err)
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	for cursor.Next(ctx) {
// 		var server ServerList
// 		cursor.Decode(&server)

// 		normalizedCode := normalizeName(server.Name)
// 		server.ServiceCode = normalizedCode

// 		_, err := serverListCol.ReplaceOne(ctx, bson.M{"_id": server.ID}, server)
// 		if err != nil {
// 			log.Printf("Error updating service code for server %s: %v\n", server.Name, err)
// 		}
// 	}

// 	log.Println("Service codes updated successfully")
// }

// // Handler function to update server prices
// func updateServerPrices(c echo.Context) error {
// 	ctx := context.Background()
// 	serverListCol := db.Collection("server_list")

// 	cursor, err := serverListCol.Find(ctx, bson.M{})
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch server list"})
// 	}
// 	defer cursor.Close(ctx)

// 	var serverList []ServerList
// 	if err := cursor.All(ctx, &serverList); err != nil {
// 		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse server list"})
// 	}

// 	for _, url := range urls {
// 		data, err := fetchDataWithRetry(url)
// 		if err != nil || data == nil {
// 			log.Printf("Skipping URL due to no data: %s\n", url)
// 			continue
// 		}

// 		calculatePrice := priceCalculations[url]

// 		for _, server := range serverList {
// 			priceUpdated := false
// 			for i := range server.Servers {
// 				serverData := &server.Servers[i]
// 				matchedItem := findMatchingItem(data, server.Name, serverData.ServerNumber)

// 				if matchedItem != nil {
// 					adjustedPrice := calculatePrice(matchedItem.Price)
// 					if !pricesAreEqual(adjustedPrice, serverData.Price) {
// 						serverData.Price = fmt.Sprintf("%.2f", adjustedPrice)
// 						priceUpdated = true
// 					}
// 				}
// 			}

// 			if priceUpdated {
// 				_, err := serverListCol.ReplaceOne(ctx, bson.M{"_id": server.ID}, server)
// 				if err != nil {
// 					log.Printf("Error updating price for server %s: %v\n", server.Name, err)
// 				}
// 			}
// 		}
// 		log.Printf("Data from %s processed successfully\n", url)
// 		time.Sleep(1 * time.Second) // Delay between URL fetches
// 	}

// 	// Update prices for server 9 using the ccpay URL
// 	for _, server := range serverList {
// 		server9 := findServerNine(server.Servers)
// 		if server9 != nil && server9.ServiceName != "" {
// 			ccpayUrl := fmt.Sprintf("https://php.paidsms.in/p/?server=ccpay&name=%s", server9.ServiceName)
// 			log.Printf("Fetching data from %s\n", ccpayUrl)
// 			resp, err := http.Get(ccpayUrl)
// 			if err != nil {
// 				log.Printf("Error fetching data from %s: %v\n", ccpayUrl, err)
// 				continue
// 			}
// 			defer resp.Body.Close()

// 			var ccpayData struct {
// 				Price float64 `json:"price"`
// 			}
// 			if err := json.NewDecoder(resp.Body).Decode(&ccpayData); err != nil {
// 				log.Printf("Error decoding JSON from %s: %v\n", ccpayUrl, err)
// 				continue
// 			}

// 			if ccpayData.Price > 0 {
// 				calculatedPrice := calculatePrice(ccpayData.Price)
// 				if !pricesAreEqual(calculatedPrice, server9.Price) {
// 					server9.Price = fmt.Sprintf("%.2f", calculatedPrice)
// 					_, err := serverListCol.ReplaceOne(ctx, bson.M{"_id": server.ID}, server)
// 					if err != nil {
// 						log.Printf("Error updating price for %s: %v\n", server9.ServiceName, err)
// 					}
// 					log.Printf("Updated price for %s to %.2f\n", server9.ServiceName, calculatedPrice)
// 				} else {
// 					log.Printf("Price for %s is already up-to-date\n", server9.ServiceName)
// 				}
// 			} else {
// 				log.Printf("No valid data found for %s at %s\n", server9.ServiceName, ccpayUrl)
// 			}
// 		}
// 	}

// 	return c.JSON(http.StatusOK, map[string]string{
// 		"message": "Server prices updated successfully",
// 	})
// }

// // Handler to add new service data based on URLs
// func addNewServiceData(c echo.Context) error {
// 	name := c.FormValue("name")

// 	for _, url := range urls {
// 		data, err := fetchDataWithRetry(url)
// 		if err != nil || data == nil {
// 			log.Printf("Skipping URL due to no data: %s\n", url)
// 			continue
// 		}

// 		calculatePrice := priceCalculations[url]
// 		matchedItem := findMatchingItem(data, name)

// 		if matchedItem != nil {
// 			adjustedPrice := calculatePrice(matchedItem.Price)

// 			existingItem, similarity := findExistingItem(name)
// 			if existingItem != nil && similarity > 0.8 {
// 				updateOrInsertServerData(existingItem, matchedItem, adjustedPrice)
// 			} else {
// 				saveNewServiceData(name, matchedItem, adjustedPrice)
// 			}

// 			log.Printf("Data for %s from %s saved to the database successfully.\n", name, url)
// 			time.Sleep(1 * time.Second)
// 		} else {
// 			log.Printf("No data found for %s at %s\n", name, url)
// 		}
// 	}

// 	updateServiceCodes()
// 	calculateLowestPrices()

// 	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("Data for %s added and saved successfully", name)})
// }

// // Handler to add or update service data using CCPAY URL
// func addccpayServiceNameData(c echo.Context) error {
// 	name := c.FormValue("name")
// 	serviceName := c.FormValue("serviceName")

// 	ccpayUrl := fmt.Sprintf("https://php.paidsms.in/p/ccpay.php?type=ccpay&name=%s", serviceName)
// 	data, err := fetchDataFromURL(ccpayUrl)
// 	if err != nil || data == nil || data.Price == 0 {
// 		return c.JSON(http.StatusNotFound, map[string]string{"message": fmt.Sprintf("No valid data found for %s", serviceName)})
// 	}

// 	calculatedPrice := calculatePrice(data.Price)
// 	existingItem, similarity := findExistingItem(name)
// 	if existingItem != nil && similarity > 0.8 {
// 		updateOrInsertServerData(existingItem, &ServerData{Name: name, ServerNumber: 9}, calculatedPrice)
// 	} else {
// 		saveNewServiceData(name, &ServerData{Name: name, ServerNumber: 9}, calculatedPrice)
// 	}

// 	updateServiceCodes()
// 	calculateLowestPrices()

// 	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("Data for %s with serviceName %s saved successfully", name, serviceName)})
// }

// // Handler to block or unblock a service
// func BlockUnblockService(c echo.Context) error {
// 	var req BlockUnblockRequest
// 	if err := c.Bind(&req); err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
// 	}
// 	if err := c.Validate(&req); err != nil {
// 		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid data"})
// 	}

// 	db := models.GetDB()
// 	collection := db.Collection("serverList")

// 	// Find the existing service by name
// 	var existingItem models.ServerList
// 	err := collection.FindOne(context.Background(), bson.M{"name": req.Name}).Decode(&existingItem)
// 	if err != nil {
// 		return c.JSON(http.StatusNotFound, echo.Map{"message": fmt.Sprintf("Service with name %s not found.", req.Name)})
// 	}

// 	// Find the specific server within the service
// 	found := false
// 	for i, server := range existingItem.Servers {
// 		if server.ServerNumber == req.ServerNumber {
// 			existingItem.Servers[i].Block = req.Block
// 			found = true
// 			break
// 		}
// 	}

// 	if !found {
// 		return c.JSON(http.StatusNotFound, echo.Map{
// 			"message": fmt.Sprintf("Server number %d not found for service with name %s.", req.ServerNumber, req.Name),
// 		})
// 	}

// 	// Update the block status in the database
// 	update := bson.M{"$set": bson.M{"servers": existingItem.Servers}}
// 	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": existingItem.ID}, update)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update block status"})
// 	}

// 	status := "unblocked"
// 	if req.Block {
// 		status = "blocked"
// 	}
// 	return c.JSON(http.StatusOK, echo.Map{
// 		"message": fmt.Sprintf("Service with name %s and server number %d has been %s.", req.Name, req.ServerNumber, status),
// 	})
// }

// // Handler to delete a service by name
// func deleteService(c echo.Context) error {
// 	name := c.FormValue("name")
// 	result, err := deleteServiceByName(name)
// 	if err != nil || result.DeletedCount == 0 {
// 		return c.JSON(http.StatusNotFound, map[string]string{"message": fmt.Sprintf("Service %s not found", name)})
// 	}
// 	return c.JSON(http.StatusOK, map[string]string{"message": fmt.Sprintf("Service %s has been deleted successfully", name)})
// }

// // Helper function to fetch data with retry logic
// func fetchDataWithRetry(url string) ([]ServerData, error) {
// 	// Similar logic as before, retrying on failure
// }

// // Helper function to find matching item by name
// func findMatchingItem(data []ServerData, name string) *ServerData {
// 	for _, item := range data {
// 		if normalizeName(item.Name) == normalizeName(name) {
// 			return &item
// 		}
// 	}
// 	return nil
// }

// // Helper function to calculate price
// func calculatePrice(price float64) float64 {
// 	// Custom price calculation logic here
// }

// // Helper function to update or insert server data in existing item
// func updateOrInsertServerData(existingItem *ServerList, matchedItem *ServerData, price float64) {
// 	// Update or insert server data in MongoDB
// }

// // Helper function to save new service data
// func saveNewServiceData(name string, matchedItem *ServerData, price float64) {
// 	// Insert a new document in MongoDB
// }

// // Helper function to delete a service by name
// func deleteServiceByName(name string) (*mongo.DeleteResult, error) {
// 	return db.Collection("server_list").DeleteOne(context.Background(), bson.M{"name": name})
// }

// // Helper function to update block status
// func updateBlockStatus(existingItem *ServerList, serverNumber int, block bool) {
// 	// Update the block status of the specified server number
// }
