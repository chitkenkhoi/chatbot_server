package chatbotapi_test
import(
	"testing"
	"fmt"
	"server/chatbotAPI"
)
func TestGetStreamingResponseFromModelAPI(t *testing.T){
	for token := range chatbotapi.GetStreamingResponseFromModelAPI("Hello"){
		fmt.Println(token)
	}
	fmt.Println("Channel is closed, all data received!")
}