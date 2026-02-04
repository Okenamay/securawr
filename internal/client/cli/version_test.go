package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	// Сохраняем старый stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Запускаем команду
	// Вызываем Run напрямую, чтобы не триггерить PersistentPreRun root-команды
	versionCmd.Run(versionCmd, []string{})

	// Закрываем writer и восстанавливаем stdout
	w.Close()
	os.Stdout = oldStdout

	// Читаем вывод
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Проверки
	expectedSubstrings := []string{
		"SecuRawr Client",
		"Version:",
		"Build Date:",
		"Go Version:",
		"OS/Arch:",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("Output missing substring %q. Got:\n%s", sub, output)
		}
	}
}
