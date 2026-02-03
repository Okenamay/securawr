package cli

import (
	"testing"
)

func TestRootCmd_Structure(t *testing.T) {
	if rootCmd.Use != "securawr" {
		t.Errorf("Expected Use: securawr, got: %s", rootCmd.Use)
	}

	// Проверяем наличие добавленных подкоманд
	commands := rootCmd.Commands()
	expectedCmds := map[string]bool{
		"auth":    false,
		"version": false,
		"add":     false,
		"list":    false,
		"get":     false,
	}

	for _, cmd := range commands {
		// У Cobra может быть много алиасов, берем первое слово из Use
		// но в данном случае у нас простые имена
		if _, exists := expectedCmds[cmd.Name()]; exists {
			expectedCmds[cmd.Name()] = true
		}
	}

	for name, found := range expectedCmds {
		if !found {
			t.Errorf("Expected command %q to be registered in rootCmd", name)
		}
	}
}

// Примечание: Мы не тестируем Execute() и initApp() здесь, так как они требуют
// моков файловой системы и Keyring (интеграционные тесты)
