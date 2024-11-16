package lib

// func GenerateTronAddress(tronName, tronPasswd string)(map[string]string, error){
// 	address := map[string]string{}

// 	// Create a new TRON key using the KeyManager
// 	walletManager := tron.NewWalletManager()
// 	wallet, _, err := walletManager.CreateNewWallet(tronName, tronPasswd)
// 	if err != nil {
// 		logs.Logger.Error("tron wallet is not created")
// 		return map[string]string{}, err
// 	}

// 	addr := walletManager.GenerateAddress()

// 	key, err := keyManager.NewKey()
// 	if err != nil {
// 		return "", "", fmt.Errorf("failed to generate key: %v", err)
// 	}

// 	// Extract private key and address
// 	privateKey := key.Private
// 	address := key.Address

// 	return address, privateKey, nil
// }
