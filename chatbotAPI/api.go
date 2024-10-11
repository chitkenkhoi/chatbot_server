package chatbotapi
import (
    "bufio"
    "fmt"
    "net/http"
    "strings"
    "encoding/json"
    "os"
)

func GetStreamingResponseFromModelAPI(message string) <-chan string {
    // Create a channel to send tokens
    tokenChan := make(chan string)

    go func() {
        // Close the channel when the function returns
        defer close(tokenChan)

        // Prepare the request body
        reqBody := map[string]string{"conversation": message}
        jsonBody, err := json.Marshal(reqBody)
        if err != nil {
            fmt.Println("Error marshalling JSON:", err)
            return
        }

        // Create a new request
        req, err := http.NewRequest("POST", os.Getenv("GOOGLE_COLAB_API_URL"), strings.NewReader(string(jsonBody)))
        if err != nil {
            fmt.Println("Error creating request:", err)
            return
        }

        // Set headers
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("ngrok-skip-browser-warning","hello")

        // Send the request
        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            fmt.Println("Error sending request:", err)
            return
        }
        defer resp.Body.Close()

        // Check the response status
        if resp.StatusCode != http.StatusOK {
            fmt.Println("Unexpected status code:", resp.StatusCode)
            return
        }

        // Read the response body line by line
        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            token := scanner.Text()
            tokenChan <- token
        }

        if err := scanner.Err(); err != nil {
            fmt.Println("Error reading response:", err)
        }
    }()

    return tokenChan
}