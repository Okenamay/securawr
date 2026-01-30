package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/client/grpcclient"
)

var (
	login    string
	password string
)

// Вспомогательная функция для получения токена из конфига
func getToken() string {
	if ConfigManager != nil {
		return ConfigManager.GetToken()
	}
	return ""
}

// registerCmd представляет команду регистрации нового пользователя
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	Long:  `Create a new account in SecuRawr system. Usage: securawr register -u <login> -p <password>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Валидация входных данных
		if login == "" || password == "" {
			return fmt.Errorf("login and password are required")
		}

		// 2. Получаем адрес сервера из конфига (загружен в root.go)
		serverAddr := ConfigManager.GetServerAddress()
		fmt.Printf("Connecting to server at %s...\n", serverAddr)

		// 3. Создаём соединение, передаем провайдер токена
		conn, err := grpcclient.NewClient(serverAddr, getToken)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)

		// 4. Выполняем запрос с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.RegisterRequest{
			Login:          login,
			AuthKey:        []byte(password),
			AuthSalt:       []byte{}, // Пока пусто
			EncryptionSalt: []byte{}, // Пока пусто
		}

		resp, err := client.Register(ctx, req)
		if err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}

		if resp.Success {
			fmt.Println("Success!")
			fmt.Println(resp.Message)
		} else {
			fmt.Println("Registration failed: " + resp.Message)
		}
		return nil
	},
}

// loginCmd представляет команду входа в систему
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to the system",
	Long:  `Authenticate with the server and save session token. Usage: securawr login -u <login> -p <password>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Валидация
		if login == "" || password == "" {
			return fmt.Errorf("login and password are required")
		}

		// 2. Подключение
		serverAddr := ConfigManager.GetServerAddress()
		fmt.Printf("Connecting to server at %s...\n", serverAddr)

		conn, err := grpcclient.NewClient(serverAddr, getToken)
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)

		// 3. Вызов метода Login
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.LoginRequest{
			Login:   login,
			AuthKey: []byte(password),
		}

		resp, err := client.Login(ctx, req)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// 4. Сохранение токена
		if err := ConfigManager.SetToken(resp.Token); err != nil {
			return fmt.Errorf("failed to save token to config: %w", err)
		}

		fmt.Println("Login successful! Session token saved.")
		return nil
	},
}

func init() {
	// Регистрируем команду registerCmd
	rootCmd.AddCommand(registerCmd)

	// Настраиваем флаги
	registerCmd.Flags().StringVarP(&login, "user", "u", "", "Username (login)")
	registerCmd.Flags().StringVarP(&password, "password", "p", "", "Password")
	registerCmd.MarkFlagRequired("user")
	registerCmd.MarkFlagRequired("password")

	// Регистрируем команду loginCmd
	rootCmd.AddCommand(loginCmd)

	// Переиспользуем переменные login/password, так как команды не запускаются
	// одновременно
	loginCmd.Flags().StringVarP(&login, "user", "u", "", "Username (login)")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Password")
	loginCmd.MarkFlagRequired("user")
	loginCmd.MarkFlagRequired("password")
}
