package cli

import (
	"testing"
)

func TestDataCommands_ArgsValidation(t *testing.T) {
	// 1. Тест команды Add (требует 1 аргумент)
	t.Run("Add Command Args", func(t *testing.T) {
		// Нет аргументов -> Ошибка
		err := addCmd.Args(addCmd, []string{})
		if err == nil {
			t.Error("Expected error when running 'add' without args, got nil")
		}

		// 1 аргумент -> OK
		err = addCmd.Args(addCmd, []string{"file.txt"})
		if err != nil {
			t.Errorf("Expected no error with 1 arg, got: %v", err)
		}
	})

	// 2. Тест команды Get (требует 1 аргумент)
	t.Run("Get Command Args", func(t *testing.T) {
		// Нет аргументов -> Ошибка
		err := getCmd.Args(getCmd, []string{})
		if err == nil {
			t.Error("Expected error when running 'get' without args, got nil")
		}

		// 1 аргумент -> OK
		err = getCmd.Args(getCmd, []string{"some-uuid"})
		if err != nil {
			t.Errorf("Expected no error with 1 arg, got: %v", err)
		}
	})
}

func TestGetToken_NilConfig(t *testing.T) {
	// Сбрасываем глобальный конфиг
	oldConfig := ConfigManager
	ConfigManager = nil
	defer func() { ConfigManager = oldConfig }()

	// Должна вернуться пустая строка, а не паника
	if got := getToken(); got != "" {
		t.Errorf("getToken() = %v, want empty string when ConfigManager is nil", got)
	}
}
