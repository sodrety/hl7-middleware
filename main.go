package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Version information (will be set during build)
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// Constants for HL7 message structure
const (
	SegmentSeparator      = "\r"
	FieldSeparator        = "|"
	ComponentSeparator    = "^"
	SubComponentSeparator = "&"
	RepetitionSeparator   = "~"
	EscapeCharacter       = "\\"
)

// HL7Message represents a complete HL7 message
type HL7Message struct {
	Segments []Segment
}

// Segment represents an HL7 segment
type Segment struct {
	Type   string
	Fields []string
}

// HTTPResponse represents the standard API response
type HTTPResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    *HL7Message `json:"data,omitempty"`
}

// NewHL7Message creates a new HL7 message
func NewHL7Message() *HL7Message {
	return &HL7Message{
		Segments: make([]Segment, 0),
	}
}

// AddSegment adds a new segment to the message
func (m *HL7Message) AddSegment(segmentType string, fields ...string) {
	segment := Segment{
		Type:   segmentType,
		Fields: fields,
	}
	m.Segments = append(m.Segments, segment)
}

// GenerateMessage converts the HL7Message to a string
func (m *HL7Message) GenerateMessage() string {
	var messageBuilder strings.Builder

	for _, segment := range m.Segments {
		messageBuilder.WriteString(segment.Type)
		for _, field := range segment.Fields {
			messageBuilder.WriteString(FieldSeparator)
			messageBuilder.WriteString(field)
		}
		messageBuilder.WriteString(SegmentSeparator)
	}

	return messageBuilder.String()
}

// ParseHL7Message parses an HL7 message string into an HL7Message struct
func ParseHL7Message(messageStr string) (*HL7Message, error) {
	message := NewHL7Message()

	scanner := bufio.NewScanner(strings.NewReader(messageStr))
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), SegmentSeparator); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	for scanner.Scan() {
		segmentStr := scanner.Text()
		if len(segmentStr) == 0 {
			continue
		}

		fields := strings.Split(segmentStr, FieldSeparator)
		if len(fields) < 1 {
			return nil, errors.New("invalid segment format")
		}

		segment := Segment{
			Type:   fields[0],
			Fields: fields[1:],
		}
		message.Segments = append(message.Segments, segment)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return message, nil
}

// HTTP Server handlers

// handleParseHL7 handles POST requests to parse HL7 messages
func handleParseHL7(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, false, "Method not allowed", nil, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendJSONResponse(w, false, "Error reading request body", nil, http.StatusBadRequest)
		return
	}

	message, err := ParseHL7Message(string(body))
	if err != nil {
		sendJSONResponse(w, false, fmt.Sprintf("Error parsing HL7 message: %v", err), nil, http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, true, "HL7 message parsed successfully", message, http.StatusOK)
}

// handleGenerateHL7 handles GET requests to generate sample HL7 messages
func handleGenerateHL7(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, false, "Method not allowed", nil, http.StatusMethodNotAllowed)
		return
	}

	message := GenerateSampleMessage()
	sendJSONResponse(w, true, "HL7 message generated successfully", message, http.StatusOK)
}

// Helper function to send JSON responses
func sendJSONResponse(w http.ResponseWriter, success bool, message string, data *HL7Message, statusCode int) {
	response := HTTPResponse{
		Success: success,
		Message: message,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Helper function to generate a sample HL7 message
func GenerateSampleMessage() *HL7Message {
	message := NewHL7Message()

	currentTime := time.Now().Format("20060102150405")
	message.AddSegment("MSH",
		"^~\\&",
		"SENDING_APP",
		"SENDING_FACILITY",
		"RECEIVING_APP",
		"RECEIVING_FACILITY",
		currentTime,
		"",
		"ADT^A01",
		"MSG00001",
		"P",
		"2.5")

	message.AddSegment("PID",
		"",
		"12345",
		"",
		"",
		"Doe^John",
		"",
		"19800101",
		"M")

	return message
}

// HTTP Client functions

// SendHL7Message sends an HL7 message to a server
func SendHL7Message(url string, message *HL7Message) (*HTTPResponse, error) {
	hl7String := message.GenerateMessage()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(hl7String))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	var response HTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &response, nil
}

// GetSampleHL7Message retrieves a sample HL7 message from a server
func GetSampleHL7Message(url string) (*HTTPResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting sample message: %v", err)
	}
	defer resp.Body.Close()

	var response HTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &response, nil
}

// Health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	sendJSONResponse(w, true, "Service is healthy", nil, http.StatusOK)
}

// Version information endpoint
func handleVersion(w http.ResponseWriter, r *http.Request) {
	info := struct {
		Version   string `json:"version"`
		BuildDate string `json:"buildDate"`
	}{
		Version:   Version,
		BuildDate: BuildDate,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	// Print version information
	fmt.Printf("HL7 Processor v%s (Built: %s)\n", Version, BuildDate)

	// Allow port configuration via environment variable
	port := os.Getenv("HL7_PORT")
	if port == "" {
		port = "8080"
	}
	// Set up HTTP server routes
	http.HandleFunc("/parse", handleParseHL7)
	http.HandleFunc("/generate", handleGenerateHL7)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/version", handleVersion)

	// Server example
	go func() {
		fmt.Println("Starting server on :8080...")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Keep the main thread running
	select {}
}
