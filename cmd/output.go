package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type cliError struct {
	Code     string      `json:"code"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	ExitCode int         `json:"-"`
	Err      error       `json:"-"`
}

func (e *cliError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil || e.Err.Error() == "" || e.Err.Error() == e.Message {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *cliError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func newCLIError(code, message string, exitCode int, err error) *cliError {
	return &cliError{
		Code:     code,
		Message:  message,
		ExitCode: exitCode,
		Err:      err,
	}
}

func newCLIErrorWithDetails(code, message string, exitCode int, details interface{}, err error) *cliError {
	return &cliError{
		Code:     code,
		Message:  message,
		Details:  details,
		ExitCode: exitCode,
		Err:      err,
	}
}

func isJSONOutput() bool {
	if outputJSON || strings.EqualFold(outputFormat, "json") {
		return true
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--json" {
			return true
		}
		if strings.HasPrefix(arg, "--output=") {
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--output="))
			return strings.EqualFold(value, "json")
		}
		if arg == "--output" && i+1 < len(args) {
			return strings.EqualFold(strings.TrimSpace(args[i+1]), "json")
		}
	}
	return false
}

func logWriter() *os.File {
	if isJSONOutput() {
		return os.Stderr
	}
	return os.Stdout
}

func logf(format string, args ...interface{}) {
	fmt.Fprintf(logWriter(), format, args...)
}

func logln(args ...interface{}) {
	fmt.Fprintln(logWriter(), args...)
}

type successEnvelope struct {
	Success bool        `json:"success"`
	Command string      `json:"command"`
	Data    interface{} `json:"data,omitempty"`
}

func emitSuccess(command string, data interface{}) error {
	if !isJSONOutput() {
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(successEnvelope{
		Success: true,
		Command: command,
		Data:    data,
	})
}

type errorEnvelope struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	} `json:"error"`
}

func HandleError(err error) int {
	if err == nil {
		return 0
	}

	code, message, details, exitCode := classifyError(err)
	if isJSONOutput() {
		enc := json.NewEncoder(os.Stderr)
		enc.SetEscapeHTML(false)
		payload := errorEnvelope{Success: false}
		payload.Error.Code = code
		payload.Error.Message = message
		payload.Error.Details = details
		_ = enc.Encode(payload)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	return exitCode
}

func classifyError(err error) (code string, message string, details interface{}, exitCode int) {
	var cliErr *cliError
	if errors.As(err, &cliErr) {
		return cliErr.Code, cliErr.Error(), cliErr.Details, cliErr.ExitCode
	}

	message = strings.TrimSpace(err.Error())
	if message == "" {
		message = "unknown error"
	}

	lowerMsg := strings.ToLower(message)
	switch {
	case strings.Contains(lowerMsg, "build failed"), strings.Contains(lowerMsg, "build timeout"), strings.Contains(lowerMsg, "unknown build status"):
		return "build_failed", message, nil, 3
	case strings.Contains(lowerMsg, "publish"):
		return "publish_failed", message, nil, 4
	case strings.Contains(lowerMsg, "api error"), strings.Contains(lowerMsg, "request failed"):
		return "api_error", message, nil, 2
	default:
		return "general_error", message, nil, 1
	}
}
