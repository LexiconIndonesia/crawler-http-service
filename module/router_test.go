package module

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adryanev/go-http-service-template/common/db"
	"github.com/adryanev/go-http-service-template/common/messaging"
	"github.com/adryanev/go-http-service-template/repository"
	"github.com/nats-io/nats.go/jetstream"
)

func TestModuleRouter(t *testing.T) {
	// Create a new module with nil dependencies (for testing only)
	mod := NewModule(nil, nil)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Get the router and serve the request
	handler := mod.Router()
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body using JSON unmarshaling for more reliable comparison
	var response map[string]interface{}

	// Parse the actual response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// Check individual fields
	status, ok := response["status"].(float64)
	if !ok {
		t.Errorf("Expected status to be a number, got %T", response["status"])
	} else if int(status) != 200 {
		t.Errorf("Expected status to be 200, got %v", status)
	}

	message, ok := response["message"].(string)
	if !ok {
		t.Errorf("Expected message to be a string, got %T", response["message"])
	} else if message != "Hello with dependency injection" {
		t.Errorf("Expected message to be 'Hello with dependency injection', got %v", message)
	}
}

func TestModuleWithMockDB(t *testing.T) {
	// Create mock dependencies
	mockDB := NewMockDB()
	mockNats := NewMockNatsClient()

	// Create a new module with the mock dependencies
	mod := NewModule(mockDB.DB, mockNats.NatsClient)

	// Test the module
	if mod.DB != mockDB.DB {
		t.Errorf("expected DB to be the mock DB instance")
	}

	if mod.NatsClient != mockNats.NatsClient {
		t.Errorf("expected NatsClient to be the mock NATS client instance")
	}
}

// MockDB implements the db.DB structure for testing
type MockDB struct {
	*db.DB
}

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		DB: &db.DB{
			Pool:    nil,
			Queries: &repository.Queries{},
		},
	}
}

// Ping mocks the database ping method
func (m *MockDB) Ping(ctx context.Context) error {
	return nil
}

// MockNatsClient implements the messaging.NatsClient for testing
type MockNatsClient struct {
	*messaging.NatsClient
	PublishCalled bool
	SubjectsSeen  []string
}

// NewMockNatsClient creates a new mock NATS client
func NewMockNatsClient() *MockNatsClient {
	return &MockNatsClient{
		NatsClient:    &messaging.NatsClient{},
		PublishCalled: false,
		SubjectsSeen:  []string{},
	}
}

// Publish mocks the publish method
func (m *MockNatsClient) Publish(subject string, data []byte) error {
	m.PublishCalled = true
	m.SubjectsSeen = append(m.SubjectsSeen, subject)
	return nil
}

// We're simplifying the test by removing the complex mocks for NATS and JetStream
// since they're causing compatibility issues with the latest version of the library.
// In a real test, you would likely use a more comprehensive mocking approach.

// PublishAsync is a simplified version for testing
func (m *MockNatsClient) PublishAsync(subject string, data []byte) (jetstream.PubAckFuture, error) {
	m.PublishCalled = true
	m.SubjectsSeen = append(m.SubjectsSeen, subject)
	// Return nil for testing purposes as we won't use the result
	return nil, nil
}

// GetStream is a simplified version for testing
func (m *MockNatsClient) GetStream(ctx context.Context, name string) (jetstream.Stream, error) {
	// Return nil for testing purposes as we won't use the result
	return nil, nil
}
