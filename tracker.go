package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const GEMINI_API_URL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-pro:generateContent"

type RequestPayload struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type ResponsePayload struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func generateAIResponse(query string) (string, error) {
	apiKey := "AIzaSyDAG1Hlm4Ge_ou5czvTHP-tyokvYhdE8wA"
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	payload := RequestPayload{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{Text: query},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(fmt.Sprintf("%s?key=%s", GEMINI_API_URL, apiKey), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response ResponsePayload
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if len(response.Candidates) > 0 && len(response.Candidates[0].Content.Parts) > 0 {
		return response.Candidates[0].Content.Parts[0].Text, nil
	}

	return "No response from Gemini.", nil
}

func handler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("query") // Get the query parameter from the request URL
    if query == "" { // Check if the query is empty
        http.Error(w, "Query parameter is required", http.StatusBadRequest)
        return
    }

    response, err := generateAIResponse(query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"response": response})
}


func main() {
	http.HandleFunc("/ask", handler)
	fmt.Println("Server running on port 8080...")
	http.ListenAndServe("127.0.0.1:8080", nil)
}
