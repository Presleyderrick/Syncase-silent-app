package auth

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const credentialsFile = "user_credentials.json"

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PromptUserCredentials prompts from CLI, unless cached
func PromptUserCredentials() (string, string, error) {
	// Check if credentials are saved
	if email, password, err := readCachedCredentials(); err == nil {
		fmt.Println("🔐 Auto-login using saved credentials.")
		return email, password, nil
	}

	// Prompt for email/password
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("📧 Enter your email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("🔑 Enter your password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if email == "" || password == "" {
		return "", "", errors.New("email or password cannot be empty")
	}

	// Save credentials
	err := saveCredentials(email, password)
	if err != nil {
		fmt.Println("⚠️ Failed to save credentials:", err)
	}

	return email, password, nil
}

func saveCredentials(email, password string) error {
	creds := Credentials{Email: email, Password: password}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credentialsFile, data, 0600) // readable only by user
}

func readCachedCredentials() (string, string, error) {
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return "", "", err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", "", err
	}
	if creds.Email == "" || creds.Password == "" {
		return "", "", errors.New("invalid cached credentials")
	}
	return creds.Email, creds.Password, nil
}

// Optional helper to clear credentials (e.g., for logout)
func ClearSavedCredentials() error {
	return os.Remove(credentialsFile)
}