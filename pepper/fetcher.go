package main

import (
	"AlgoTN/common"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func FetchProblemInfo(url string) (*common.ProblemInfo, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add the secret to the request header
	req.Header.Add("x-auth-secret-key", Secret)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var problemInfo common.ProblemInfo
	err = json.NewDecoder(resp.Body).Decode(&problemInfo)
	if err != nil {
		return nil, err
	}

	return &problemInfo, nil
}

func DownloadAsText(url string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add the secret to the request header
	req.Header.Add("x-auth-secret-key", Secret)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(fileBytes), nil
}
