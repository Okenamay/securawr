package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	pb "github.com/Okenamay/securawr/gen/proto"
	"github.com/Okenamay/securawr/internal/client/grpcclient"
	"github.com/Okenamay/securawr/internal/client/storage"
)

var (
	description string
)

// Структура для формирования JSON метаданных
type FileMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// addCmd загружает файл на сервер
var addCmd = &cobra.Command{
	Use:   "add <file_path>",
	Short: "Upload a file to secure storage",
	Long:  `Read a file from local disk and upload it to the server securely.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		// 1. Проверяем токен
		token := getToken()
		if token == "" {
			return fmt.Errorf("you are not logged in. Run 'securawr login' first")
		}

		// Проверка размера файла перед чтением
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		if fileInfo.Size() > ConfigManager.GetMaxFileSize() {
			return fmt.Errorf("file size (%d bytes) exceeds the limit of %d bytes", fileInfo.Size(), ConfigManager.GetMaxFileSize())
		}

		// 2. Читаем файл
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		filename := filepath.Base(filePath)

		// Формируем JSON метаданные
		meta := FileMeta{
			Name:        filename,
			Description: description,
		}
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("failed to encode metadata: %w", err)
		}

		// Сохранение в локальный кеш (LRU)
		// Используем StoragePath и LocalCacheSize из конфигурации
		kvStore, err := storage.NewStorage(ConfigManager.GetStoragePath(), ConfigManager.GetLocalCacheSize())
		if err != nil {
			// Логируем ошибку, но не прерываем работу (кеш опционален для upload)
			fmt.Printf("Warning: failed to open local cache: %v\n", err)
		} else {
			defer kvStore.Close()

			record := storage.LocalRecord{
				ID:            filename,
				Type:          int32(pb.DataType_BINARY),
				Name:          filename,
				Description:   description,
				CreatedAt:     time.Now(),
				EncryptedData: data,
			}

			err = kvStore.Save(record)
			if err != nil {
				fmt.Printf("Warning: failed to save to local cache: %v\n", err)
			} else {
				fmt.Println("File cached locally.")
			}
		}

		// 3. Подключаемся к серверу
		serverAddr := ConfigManager.GetServerAddress()
		// Передаем функцию getToken, чтобы клиент добавил заголовок авторизации
		conn, err := grpcclient.NewClient(serverAddr, ConfigManager.GetCertFile())
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer conn.Close()

		// 4. Отправляем данные
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Uploading '%s' (%d bytes)...\n", filename, len(data))

		req := &pb.AddDataRequest{
			Type:          pb.DataType_BINARY,
			EncryptedData: data,
			MetaInfo:      string(metaJSON),
		}

		resp, err := conn.AddData(ctx, req)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}

		fmt.Println("Success!")
		fmt.Printf("File saved with ID: %s\n", resp.Id)
		return nil
	},
}

// listCmd выводит список файлов
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored files",
	Long:  `Retrieve and display a list of all files stored on the server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := getToken()
		if token == "" {
			return fmt.Errorf("you are not logged in. Run 'securawr login' first")
		}

		serverAddr := ConfigManager.GetServerAddress()
		conn, err := grpcclient.NewClient(serverAddr, ConfigManager.GetCertFile())
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req := &pb.ListDataRequest{
			TypeFilter: pb.DataType_UNKNOWN,
		}

		resp, err := conn.ListData(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to list data: %w", err)
		}

		if len(resp.Items) == 0 {
			fmt.Println("No files found.")
			return nil
		}

		// Форматированный вывод таблицы
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tNAME\tCREATED AT\tDESCRIPTION")
		for _, item := range resp.Items {
			// Разбираем метаданные для отображения имени
			var meta FileMeta
			_ = json.Unmarshal([]byte(item.MetaInfo), &meta)

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				item.Id,
				item.Type.String(),
				meta.Name,
				item.CreatedAt,
				meta.Description,
			)
		}
		w.Flush()

		return nil
	},
}

// getCmd скачивает файл по ID
var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Download a file by ID",
	Long:  `Download the file content from the server and save it to the current directory.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		token := getToken()
		if token == "" {
			return fmt.Errorf("you are not logged in. Run 'securawr login' first")
		}

		serverAddr := ConfigManager.GetServerAddress()
		conn, err := grpcclient.NewClient(serverAddr, ConfigManager.GetCertFile())
		if err != nil {
			return fmt.Errorf("failed to connect to server: %w", err)
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Downloading file %s...\n", id)

		resp, err := conn.GetData(ctx, &pb.GetDataRequest{Id: id})
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		var meta FileMeta
		if err := json.Unmarshal([]byte(resp.MetaInfo), &meta); err != nil {
			// Если метаданные битые, используем ID
			meta.Name = resp.Id
		}

		if meta.Name == "" {
			meta.Name = resp.Id
		}

		// Сохраняем файл в текущую директорию
		if err := os.WriteFile(meta.Name, resp.EncryptedData, 0644); err != nil {
			return fmt.Errorf("failed to save file to disk: %w", err)
		}

		fmt.Printf("Success! Saved as '%s' (%d bytes)\n", meta.Name, len(resp.EncryptedData))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)

	// Флаги для команды add
	addCmd.Flags().StringVarP(&description, "desc", "d", "", "Description of the file")
}

// getToken возвращает токен из конфигурации
func getToken() string {
	if ConfigManager == nil {
		return ""
	}
	return ConfigManager.GetToken()
}
