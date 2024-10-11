package chatbotapi_test
import(
	"testing"
	"fmt"
	"server/chatbotAPI"
)
func TestGetStreamingResponseFromModelAPI(t *testing.T){
	for token := range chatbotapi.GetStreamingResponseFromModelAPI("Nước bọt giúp tiêu hóa như thế nào?"){
		fmt.Print(token)
	}
	fmt.Println("Channel is closed, all data received!!")
}
// func TestCurl(t *testing.T){
// 	response, err := http.Get("https://9e5c-34-87-53-195.ngrok-free.app/ping")
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	defer response.Body.Close()

// 	// Read the response body
// 	body, err := io.ReadAll(response.Body) // Use io.ReadAll() instead of ioutil.ReadAll()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}

// 	// Print the response
// 	fmt.Println(string(body))
// }
// func TestPost(t *testing.T){
// 	data := map[string]string{
// 		"title":  "foo",
// 		"body":   "bar",
// 		"userId": "1",
// 	}

// 	// Convert data to JSON format
// 	jsonData, err := json.Marshal(data)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}

// 	// Send a POST request
// 	response, err := http.Post("https://7efb-34-87-53-195.ngrok-free.app/submit", "application/json", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	defer response.Body.Close()

// 	// Read the response body
// 	body, err := io.ReadAll(response.Body) // Use io.ReadAll() instead of ioutil.ReadAll()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}

// 	// Print the response
// 	fmt.Println(string(body))
// }