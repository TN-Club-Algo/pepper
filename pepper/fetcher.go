package main

import (
	"AlgoTN/common"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func DownloadAndSaveFile(url, savePath, extension string) error {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Add the secret to the request header
	req.Header.Add("x-auth-secret-key", Secret)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// Determine the file extension based on the content type
	contentType := resp.Header.Get("Content-Type")
	isArchive := strings.HasPrefix(contentType, "application/") && strings.Contains(contentType, "zip")

	// Get the original file name without the extension
	originalFileName := strings.TrimSuffix(filepath.Base(url), filepath.Ext(url))

	// Create the full save path if it doesn't exist
	if err := os.MkdirAll(savePath, os.ModePerm); err != nil {
		return err
	}

	// Create or truncate the file for writing
	var finalFileName string
	if isArchive {
		finalFileName = originalFileName
	} else {
		finalFileName = originalFileName + extension
	}

	file, err := os.Create(fmt.Sprintf("%s/%s", savePath, finalFileName))
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy the content from the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
