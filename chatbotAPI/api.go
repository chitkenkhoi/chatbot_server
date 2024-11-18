package chatbotapi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"os"
	"strings"
)
var test = os.Getenv("MODEL_API_URL")
func GetStreamingResponseFromModelAPIDemo() <-chan string {
	tokenChan := make(chan string)

	go func() {
		// Close the channel when the function returns
		defer close(tokenChan)

		// Prepare the request body
		// reqBody := map[string]string{"conversation": message}
		// jsonBody, err := json.Marshal(reqBody)
		// if err != nil {
		//     fmt.Println("Error marshalling JSON:", err)
		//     return
		// }

		// Create a new request
		req, err := http.NewRequest("GET", os.Getenv("MODEL_API_URL_DEMO"), strings.NewReader("abc"))
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("ngrok-skip-browser-warning", "hello")

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
func GetStreamingResponseFromModelAPI(message,mode string, id string, isFirst bool,cid string) <-chan string {
	if mode != "1" && mode != "2" {
		mode = "1"
	}
	// Create a channel to send tokens
	tokenChan := make(chan string)

	go func() {
		// Close the channel when the function returns
		defer close(tokenChan)

		// Prepare the request body
		reqBody := map[string]string{"query": message, "conversation_id": id, "is_first": fmt.Sprintf("%t", isFirst),"mode":mode,"cid":cid}
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			tokenChan <- "Sorry, something went wrong while processing your request"
			return
		}

		// Create a new request
		req, err := http.NewRequest("POST", os.Getenv("MODEL_API_URL"), strings.NewReader(string(jsonBody)))
		if err != nil {
			tokenChan <- "Sorry, there was an error connecting to the service"
			return
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("ngrok-skip-browser-warning", "hello")

		// Send the request
		client := &http.Client{
			Timeout: 60 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			if strings.Contains(err.Error(), "timeout") {
				tokenChan <- "Sorry, the request timed out. Please try again"
			} else {
				tokenChan <- "Sorry, there was a network error. Please check your connection"
			}
			return
		}
		defer resp.Body.Close()

		// Check the response status
		if resp.StatusCode != http.StatusOK {
			fmt.Println("Unexpected status code:", resp.StatusCode)
			tokenChan <- fmt.Sprintf("Sorry, received unexpected response (Status: %d)", resp.StatusCode)
			return
		}

		// Read the response body line by line
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			token := scanner.Text()
			if token != "" {
				tokenChan <- token + "\n"
			}
			
			
			time.Sleep(100 * time.Millisecond)
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading response:", err)
			tokenChan <- "Sorry, there was an error reading the response"
		}
	}()

	return tokenChan
}
