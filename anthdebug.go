package anthropic

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

var (
    isDebugEnabled bool
    debugLogFile   *os.File
    debugMutex     sync.Mutex
    sessionID      string
)

// EnableDebug turns on debug logging and creates a new log file for the session
func EnableDebug() error {
    debugMutex.Lock()
    defer debugMutex.Unlock()

    isDebugEnabled = true
    return initDebugLogFile()
}

// DisableDebug turns off debug logging and closes the current log file
func DisableDebug() error {
    debugMutex.Lock()
    defer debugMutex.Unlock()

    isDebugEnabled = false
    return closeDebugLogFile()
}

// IsDebugEnabled returns the current debug state
func IsDebugEnabled() bool {
    return isDebugEnabled
}

// initDebugLogFile creates a new log file for the current session
func initDebugLogFile() error {
    // Close existing log file if any
    if debugLogFile != nil {
        debugLogFile.Close()
    }

    // Create logs directory if it doesn't exist
    if err := os.MkdirAll("logs", 0755); err != nil {
        return fmt.Errorf("failed to create logs directory: %w", err)
    }

    // Generate unique session ID using timestamp
    sessionID = time.Now().Format("20060102-150405")
    logPath := filepath.Join("logs", fmt.Sprintf("anthropic-debug-%s.log", sessionID))

    var err error
    debugLogFile, err = os.Create(logPath)
    if err != nil {
        return fmt.Errorf("failed to create log file: %w", err)
    }

    // Write session start marker
    timestamp := time.Now().Format("2006-01-02 15:04:05")
    _, err = fmt.Fprintf(debugLogFile, "=== Session Started: %s ===\n\n", timestamp)
    return err
}

// closeDebugLogFile closes the current log file
func closeDebugLogFile() error {
    if debugLogFile != nil {
        timestamp := time.Now().Format("2006-01-02 15:04:05")
        _, err := fmt.Fprintf(debugLogFile, "\n=== Session Ended: %s ===\n", timestamp)
        if err != nil {
            return err
        }
        return debugLogFile.Close()
    }
    return nil
}

// Internal debug logging functions

// debugLog writes a message to the debug log if debugging is enabled
func debugLog(format string, args ...interface{}) {
    if !isDebugEnabled || debugLogFile == nil {
        return
    }

    debugMutex.Lock()
    defer debugMutex.Unlock()

    timestamp := time.Now().Format("2006-01-02 15:04:05.000")
    message := fmt.Sprintf(format, args...)
    fmt.Fprintf(debugLogFile, "[%s] %s\n", timestamp, message)
}

// debugLogJSON writes a formatted JSON object to the debug log if debugging is enabled
func debugLogJSON(prefix string, v interface{}) {
    if !isDebugEnabled || debugLogFile == nil {
        return
    }

    debugMutex.Lock()
    defer debugMutex.Unlock()

    timestamp := time.Now().Format("2006-01-02 15:04:05.000")
    jsonBytes, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        fmt.Fprintf(debugLogFile, "[%s] Error marshaling JSON for %s: %v\n", timestamp, prefix, err)
        return
    }

    fmt.Fprintf(debugLogFile, "[%s] === %s ===\n%s\n\n", timestamp, prefix, string(jsonBytes))
}

// GetSessionID returns the current debug session ID
func GetSessionID() string {
    return sessionID
}
