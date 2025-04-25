package gpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const openAIAPIKey = "YOUR_API_KEY" // Replace with your OpenAI API key
const openAIAPIURL = "https://api.openai.com/v1/chat/completions"

// ChatGPTRequest defines the structure of the request to ChatGPT
type ChatGPTRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// ChatGPTResponse defines the structure of the response from ChatGPT
type ChatGPTResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// handleSubchartImage handles the endpoint for subchart and umbrella chart name
func handleSubchartImage(w http.ResponseWriter, r *http.Request) {
	subchartName := r.URL.Query().Get("subchartName")
	umbrellaChartName := r.URL.Query().Get("umbrellaChartName")

	if subchartName == "" || umbrellaChartName == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	question := fmt.Sprintf("Give a recommendation for the image to use for a Deployment manifest in a subchart named '%s' for an umbrella chart named '%s'", subchartName, umbrellaChartName)
	response, err := queryChatGPT(question)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying ChatGPT: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(response))
}

// handleYamlConfig handles the endpoint for subchart name and YAML config file
func handleYamlConfig(w http.ResponseWriter, r *http.Request) {
	subchartName := r.URL.Query().Get("subchartName")
	yamlConfig := r.URL.Query().Get("yamlConfig")

	if subchartName == "" || yamlConfig == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	question := fmt.Sprintf("Survey a subchart template (subchart name: '%s') with the provided YAML configuration: %s", subchartName, yamlConfig)
	response, err := queryChatGPT(question)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying ChatGPT: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(response))
}

// queryChatGPT sends a query to the ChatGPT API and returns the response content
func queryChatGPT(question string) (string, error) {
	requestBody := ChatGPTRequest{
		Model: "gpt-4",
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: question},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", openAIAPIURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+openAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("non-200 response from ChatGPT: %s", string(body))
	}

	var chatResponse ChatGPTResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices found in response")
	}

	return chatResponse.Choices[0].Message.Content, nil
}

func main() {
	http.HandleFunc("/subchart-image", handleSubchartImage)
	http.HandleFunc("/yaml-config", handleYamlConfig)

	port := ":8080"
	fmt.Printf("Server started on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
