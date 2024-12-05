package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ranjankuldeep/fakeNumber/internal/database/models"
	"go.mongodb.org/mongo-driver/mongo"
)

func constructApiUrl(db *mongo.Database, server, apiKeyServer string, apiToken string, data models.ServerData, isMultiple string) (ApiRequest, error) {
	marginMap, exchangeMap, err := FetchMarginAndExchangeRate(context.TODO(), db)
	if err != nil {
		return ApiRequest{}, err
	}
	serverNumber, _ := strconv.Atoi(server)
	switch server {
	case "1":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "2":
		return ApiRequest{
			URL: fmt.Sprintf("https://5sim.net/v1/user/buy/activation/india/any/%s", data.Code),
			Headers: map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", apiToken),
				"Accept":        "application/json",
			},
		}, nil

	case "3":
		priceFloat, err := strconv.ParseFloat(data.Price, 64)
		if err != nil {
			return ApiRequest{}, fmt.Errorf("invalid price for server %d: %v", serverNumber, err)
		}
		priceFloat = (priceFloat - marginMap[serverNumber]) / exchangeMap[serverNumber]
		priceStr := fmt.Sprintf("%.2f", priceFloat)
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://smshub.org/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&operator=any&country=22&maxPrice=%s",
				apiKeyServer, data.Code, priceStr,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "4":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "5":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "6":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://tempnum.org/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "7":
		priceFloat, err := strconv.ParseFloat(data.Price, 64)
		if err != nil {
			return ApiRequest{}, fmt.Errorf("invalid price for server %d: %v", serverNumber, err)
		}
		priceFloat = (priceFloat - marginMap[serverNumber]) / exchangeMap[serverNumber]
		priceStr := fmt.Sprintf("%.2f", priceFloat)
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://smsbower.online/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&country=22&maxPrice=%s",
				apiKeyServer, data.Code, priceStr,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "8":
		priceFloat, err := strconv.ParseFloat(data.Price, 64)
		if err != nil {
			return ApiRequest{}, fmt.Errorf("invalid price for server %d: %v", serverNumber, err)
		}
		priceFloat = (priceFloat - marginMap[serverNumber]) / exchangeMap[serverNumber]
		priceStr := fmt.Sprintf("%.2f", priceFloat)
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://api.sms-activate.guru/stubs/handler_api.php?api_key=%s&action=getNumber&service=%s&operator=any&country=22&maxPrice=%s",
				apiKeyServer, data.Code, priceStr,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil

	case "9":
		return ApiRequest{
			URL: fmt.Sprintf(
				"http://www.phantomunion.com:10023/pickCode-api/push/buyCandy?token=%s&businessCode=%s&quantity=1&country=IN&effectiveTime=10",
				apiToken, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil
	case "10":
		return ApiRequest{
			URL: fmt.Sprintf(
				"https://sms-activation-service.pro/stubs/handler_api?api_key=%s&action=getNumber&service=%s&operator=any&country=22 ",
				apiKeyServer, data.Code,
			),
			Headers: map[string]string{}, // Empty headers
		}, nil
	case "11":
		if isMultiple == "true" {
			return ApiRequest{
				URL: fmt.Sprintf(
					"https://api.sms-man.com/control/get-number?token=%s&application_id=%s&country_id=14&hasMultipleSms=true",
					apiKeyServer, data.Code,
				),
				Headers: map[string]string{}, // Empty headers
			}, nil
		} else {
			return ApiRequest{
				URL: fmt.Sprintf(
					"https://api.sms-man.com/control/get-number?token=%s&application_id=%s&country_id=14&hasMultipleSms=false",
					apiKeyServer, data.Code,
				),
				Headers: map[string]string{}, // Empty headers
			}, nil
		}

	default:
		return ApiRequest{}, errors.New("invalid server value")
	}
}

func constructOtpUrl(server, apiKeyServer, token, id string) (ApiRequest, error) {
	switch server {
	case "1":
		return ApiRequest{
			URL:     fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "2":
		return ApiRequest{
			URL:     fmt.Sprintf("https://5sim.net/v1/user/check/%s", id),
			Headers: map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token), "Accept": "application/json"},
		}, nil
	case "3":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "4":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "5":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "6":
		return ApiRequest{
			URL:     fmt.Sprintf("https://tempnum.org/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "7":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "8":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-activate.guru/stubs/handler_api.php?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "9":
		return ApiRequest{
			URL:     fmt.Sprintf("http://www.phantomunion.com:10023/pickCode-api/push/sweetWrapper?token=%s&serialNumber=%s", token, id),
			Headers: map[string]string{},
		}, nil
	case "10":
		return ApiRequest{
			URL:     fmt.Sprintf("https://sms-activation-service.pro/stubs/handler_api?api_key=%s&action=getStatus&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "11":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-man.com/control/get-sms?token=%s&request_id=%s", token, id),
			Headers: map[string]string{},
		}, nil
	default:
		return ApiRequest{}, fmt.Errorf("ivalid server number")
	}
}

func ConstructNumberUrl(server, apiKeyServer, token, id, number string) (ApiRequest, error) {
	switch server {
	case "1":
		return ApiRequest{
			URL:     fmt.Sprintf("https://fastsms.su/stubs/handler_api.php?api_key=%s&action=setStatus&id=%s&status=8", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "2":
		return ApiRequest{
			URL:     fmt.Sprintf("https://5sim.net/v1/user/cancel/%s", id),
			Headers: map[string]string{"Authorization": fmt.Sprintf("Bearer %s", token), "Accept": "application/json"},
		}, nil
	case "3":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smshub.org/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "4":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.tiger-sms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "5":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.grizzlysms.com/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "6":
		return ApiRequest{
			URL:     fmt.Sprintf("https://tempnum.org/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "7":
		return ApiRequest{
			URL:     fmt.Sprintf("https://smsbower.online/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "8":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api.sms-activate.guru/stubs/handler_api.php?api_key=%s&action=setStatus&status=8&id=%s", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "9":
		return ApiRequest{
			URL:     fmt.Sprintf("https://php.paidsms.org/ccpay.php?type=cancel&number=%s", number),
			Headers: map[string]string{},
		}, nil
	case "10":
		return ApiRequest{
			URL:     fmt.Sprintf("https://sms-activation-service.pro/stubs/handler_api?api_key=%s&action=setStatus&id=%s&status=8", apiKeyServer, id),
			Headers: map[string]string{},
		}, nil
	case "11":
		return ApiRequest{
			URL:     fmt.Sprintf("https://api2.sms-man.com/control/set-status?token=%s&request_id=%s&status=reject", token, id),
			Headers: map[string]string{},
		}, nil
	default:
		return ApiRequest{}, fmt.Errorf("INVLAID_SERVER_CHOICE")
	}
}
