package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type PinataResponse struct {
	JWT             string `json:"JWT"`
	PinataApiKey    string `json:"pinata_api_key"`
	PinataApiSecret string `json:"pinata_api_secret"`
}

func GetSignedJWT(userId string) (string, error) {
	url := "https://api.pinata.cloud/v3/pinata/keys"
	payload := strings.NewReader(fmt.Sprintf("{\n  \"keyName\": \"key_%s\",\n  \"permissions\": {\n    \"admin\": false,\n    \"endpoints\": {\n      \"pinning\": {\n        \"pinFileToIPFS\": true\n      }\n    }\n  },\n  \"maxUses\": 1\n}", userId))
	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("Authorization", "Bearer "+os.Getenv("PINATA_JWT"))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var response PinataResponse
	err1 := json.Unmarshal([]byte(string(body)), &response)
	if err1 != nil {
		// Handle error
		return "", err1
	}
	return response.JWT, nil
}
