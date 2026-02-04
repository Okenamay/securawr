package cli

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unicode"

	"github.com/Okenamay/securawr/internal/client/grpcclient"
	"github.com/Okenamay/securawr/internal/client/storage"
	"github.com/Okenamay/securawr/internal/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands (register, login)",
}

// validatePassword проверяет сложность пароля
func validatePassword(pass string) error {
	if len(pass) < 8 {
		return fmt.Errorf("password too short (min 8 chars)")
	}
	var hasLetter, hasDigit bool
	for _, c := range pass {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must contain both letters and digits")
	}
	return nil
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := grpcclient.NewClient(ConfigManager.GetServerAddress(), ConfigManager.GetCertFile())
		if err != nil {
			fmt.Printf("Error connecting to server: %v\n", err)
			return
		}
		defer client.Close()

		// 1. Ввод логина и пароля
		login := prompt("Enter login: ")
		if len(login) < 4 {
			fmt.Println("login is too short (min 4 chars)")
			return
		}

		password := promptPassword("Enter password: ")
		if login == "" || password == "" {
			fmt.Println("login and password cannot be empty")
			return
		}

		// Проверка сложности пароля
		if err := validatePassword(password); err != nil {
			fmt.Printf("Invalid password: %v\n", err)
			return
		}

		// 2. Генерация солей на клиенте
		authSalt, err := crypto.GenerateRandomBytes(16)
		if err != nil {
			fmt.Printf("Crypto error (auth salt): %v\n", err)
			return
		}
		encSalt, err := crypto.GenerateRandomBytes(16)
		if err != nil {
			fmt.Printf("Crypto error (enc salt): %v\n", err)
			return
		}

		// 3. Вычисление Auth_Key = Argon2(pass, auth_salt)
		authKey, err := crypto.DeriveKey([]byte(password), authSalt)
		if err != nil {
			fmt.Printf("KDF error: %v\n", err)
			return
		}

		// 4. Отправка на сервер
		err = client.Register(context.Background(), login, authKey, authSalt, encSalt)
		if err != nil {
			fmt.Printf("Registration error: %v\n", err)
			return
		}

		fmt.Println("Registration successful! You can now login.")
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the system",
	Run: func(cmd *cobra.Command, args []string) {
		// Используем ConfigManager вместо cfg
		client, err := grpcclient.NewClient(ConfigManager.GetServerAddress(), ConfigManager.GetCertFile())
		if err != nil {
			fmt.Printf("Error connecting to server: %v\n", err)
			return
		}
		defer client.Close()

		// 1. Ввод логина
		login := prompt("Enter login: ")

		// 2. Запрос AuthSalt у сервера
		fmt.Print("Fetching auth params... ")
		authSalt, err := client.GetAuthParams(context.Background(), login)
		if err != nil {
			fmt.Printf("\nError getting auth params: %v\n", err)
			return
		}
		fmt.Println("OK")

		// 3. Ввод пароля и вычисление ключа
		password := promptPassword("Enter password: ")
		authKey, err := crypto.DeriveKey([]byte(password), authSalt)
		if err != nil {
			fmt.Printf("KDF error: %v\n", err)
			return
		}

		// 4. Логин
		token, encSalt, err := client.Login(context.Background(), login, authKey)
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		// 5. Сохранение токена в Keyring
		if err := storage.SaveToken(token); err != nil {
			fmt.Printf("Warning: failed to save token to system keyring: %v\n", err)
		} else {
			fmt.Println("Token saved to secure system storage.")
		}

		if err := ConfigManager.SetEncryptionSalt(hex.EncodeToString(encSalt)); err != nil {
			fmt.Printf("Failed to save encryption salt: %v\n", err)
			return
		}

		fmt.Printf("Login successful!\nToken: %s...\n", token[:10])
		fmt.Printf("Encryption Salt received and saved.\n")
	},
}

func init() {
	authCmd.AddCommand(registerCmd)
	authCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(authCmd)
}

// Вспомогательные функции для ввода

func prompt(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func promptPassword(label string) string {
	fmt.Print(label)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}
	fmt.Println() // Перенос строки после ввода пароля
	return string(bytePassword)
}
