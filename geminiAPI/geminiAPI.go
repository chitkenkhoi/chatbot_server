package geminiapi

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

//	func printResponse(resp *genai.GenerateContentResponse) {
//		for _, cand := range resp.Candidates {
//			if cand.Content != nil {
//				for _, part := range cand.Content.Parts {
//					fmt.Println(part)
//				}
//			}
//		}
//		fmt.Println("---")
//	}
func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Println(part)
			}
		}
	}
	fmt.Println("---")
}
func GetTopic(userQuestion string, isStream bool) (string, error) {
	ctx := context.Background()
	// Create the request body
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GENAI_API_KEY")))
	if err != nil {
		return "", fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()
	model := client.GenerativeModel("gemini-1.5-flash")
	if isStream {
		iter := model.GenerateContentStream(ctx, genai.Text("Write a story about a magic backpack."))
		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			printResponse(resp)
		}
	}
	text := fmt.Sprintf(`
Dựa trên câu hỏi của người dùng sau đây, hãy cho tôi biết chủ đề mà người dùng muốn hỏi là gì, 
nhớ rằng tôi chỉ cần biết chủ đề, không cần biết câu trả lời cụ thể.
Hãy trả lời thật ngắn gọn nhất có thể (trong khoảng 4 từ đến 9 từ).
Một ví dụ:
Câu hỏi: "Tôi cần API LLM miễn phí cho một tác vụ rất đơn giản, bạn có thể giúp tôi không?"
Câu trả lời bạn nên cho tôi: "Tìm API LLM miễn phí cho một tác vụ đơn giản"
                                  
Câu hỏi từ người dùng:
"%s"`, userQuestion)
	resp, err := model.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}
	return fmt.Sprint(resp.Candidates[0].Content.Parts[0]), nil
}
