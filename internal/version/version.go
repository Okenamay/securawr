package version

// Переменные, которые будут заполняться линкером при сборке (ldflags). Они
// должны быть доступны и клиенту, и серверу
var (
	Version   = "dev"
	BuildDate = "unknown"
)
