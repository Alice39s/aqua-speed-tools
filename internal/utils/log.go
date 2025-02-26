package utils

var IsDebug bool

func LogDebug(format string, args ...any) {
	if IsDebug {
		Gray.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func LogInfo(format string, args ...any) {
	Blue.Printf("[INFO] "+format+"\n", args...)
}

func LogSuccess(format string, args ...any) {
	Green.Printf("[DONE] "+format+"\n", args...)
}

func LogWarning(format string, args ...any) {
	Yellow.Printf("[WARN] "+format+"\n", args...)
}

func LogError(format string, args ...any) {
	Red.Printf("[ERROR] "+format+"\n", args...)
}
