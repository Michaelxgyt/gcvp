package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"net/http"
	"os/exec"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	// "github.com/gorilla/mux" // Will be added if chosen for routing

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	// Assuming the generated proto path, adjust if different
	statsService "main/internal/v2rayapi/stats/command"
)

// Global variables
var (
	currentUsersConfig UsersConfig
	configMutex        = &sync.RWMutex{}
	v2rayCmd           *exec.Cmd
	v2rayRestartMutex  = &sync.Mutex{}
)

// V2Ray related structures
type Config struct {
	Log       LogEntry      `json:"log,omitempty"`
	API       *APIConfig    `json:"api,omitempty"`    // Pointer to allow omitting if not configured
	Policy    *PolicyConfig `json:"policy,omitempty"` // Pointer to allow omitting
	Routing   *RoutingConfig `json:"routing,omitempty"` // Pointer to allow omitting
	Inbounds  []Inbound     `json:"inbounds,omitempty"`
	Outbounds []Outbound    `json:"outbounds,omitempty"`
	// Add other fields like dns, transport as needed
}

type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"` // e.g., ["StatsService"]
}

type PolicyConfig struct {
	Levels map[string]LevelPolicy `json:"levels"`
	System SystemPolicy         `json:"system"`
}

type LevelPolicy struct {
	StatsUserUplink   bool `json:"statsUserUplink"`
	StatsUserDownlink bool `json:"statsUserDownlink"`
	// Add other policy fields as needed: handshake, connIdle, uplinkOnly, downlinkOnly, etc.
}

type SystemPolicy struct {
	StatsInboundUplink   bool `json:"statsInboundUplink"`
	StatsInboundDownlink bool `json:"statsInboundDownlink"`
	// Add other system policy fields
}

type RoutingConfig struct {
	Rules []RoutingRule `json:"rules"`
	// Other routing fields like domainStrategy, balancers can be added
}

type RoutingRule struct {
	Type        string   `json:"type"`              // "field"
	InboundTag  []string `json:"inboundTag,omitempty"` // Can be nil
	OutboundTag string   `json:"outboundTag"`
	Domain      []string `json:"domain,omitempty"`    // Can be nil
	Protocol    []string `json:"protocol,omitempty"`  // Can be nil
	// Add other rule fields: port, source, user, etc.
}


type LogEntry struct {
	Loglevel string `json:"loglevel"` // "warning", "error", etc.
	Access   string `json:"access"`   // Path to access log
	Error    string `json:"error"`    // Path to error log
}

type Inbound struct {
	Port           string          `json:"port"`
	Listen         string          `json:"listen,omitempty"` // Added for API inbound
	Protocol       string          `json:"protocol"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings StreamSettings  `json:"streamSettings,omitempty"` // omitempty for dokodemo-door
	Tag            string          `json:"tag,omitempty"`
}

type InboundSettings struct {
	Clients  []Client `json:"clients"`
	Decryption string `json:"decryption,omitempty"` // For VLESS
	Default *DefaultClient `json:"default,omitempty"`
}

type Client struct {
	ID      string `json:"id"` // UUID
	AlterID int    `json:"alterId"`
	Email   string `json:"email,omitempty"` // Optional
	Level   int    `json:"level,omitempty"`
}

type DefaultClient struct {
	AlterID int `json:"alterId"`
	Level int `json:"level"`
}

type StreamSettings struct {
	Network     string             `json:"network"`     // "ws", "tcp", "kcp", etc.
	Security    string             `json:"security"`    // "none", "tls"
	WSSettings  WebSocketSettings  `json:"wsSettings,omitempty"`
	// TCPSettings tcp.Config         `json:"tcpSettings,omitempty"`
	// KCPSettings kcp.Config         `json:"kcpSettings,omitempty"`
	// TLSSettings tls.Config         `json:"tlsSettings,omitempty"` // Usually handled by Cloud Run
}

type WebSocketSettings struct {
	Path    string            `json:"path"` // e.g., "/ws"
	Headers map[string]string `json:"headers,omitempty"`
}

type Outbound struct {
	Protocol string           `json:"protocol"`
	Settings OutboundSettings `json:"settings"`
	Tag      string           `json:"tag,omitempty"`
}

type OutboundSettings struct {
	// For "freedom" protocol, settings can be empty or define specific parameters
	// For other protocols like SOCKS, HTTP, etc., specific settings are needed
}


// User represents a user with traffic and time limits.
type User struct {
	ID             string    `json:"id"`
	TrafficLimitGB float64   `json:"traffic_limit_gb"`
	TimeLimitDays  int       `json:"time_limit_days"`
	CreatedAt      time.Time `json:"created_at"`
	TrafficUsedBytes int64   `json:"traffic_used_bytes"`
	IsActive       bool      `json:"is_active"`
}

// UsersConfig is a map of users, with User.ID as the key.
type UsersConfig map[string]User

// loadUsersConfig loads the user configuration from GCS.
// If the object is not found, it returns an empty UsersConfig and nil error.
func loadUsersConfig(bucketName, objectName string) (UsersConfig, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	rc, err := client.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		log.Printf("Object %s in bucket %s not found, returning empty config", objectName, bucketName)
		return make(UsersConfig), nil
	}
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", objectName, err)
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}

	var users UsersConfig
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %v", err)
	}
	return users, nil
}

// saveUsersConfig saves the user configuration to GCS.
func saveUsersConfig(bucketName, objectName string, users UsersConfig) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("json.MarshalIndent: %v", err)
	}

	wc := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	if _, err := wc.Write(data); err != nil {
		return fmt.Errorf("Writer.Write: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	log.Printf("Successfully saved config to gs://%s/%s", bucketName, objectName)
	return nil
}

// queryV2RayStats queries the V2Ray StatsService for a user's traffic.
// userEmailTag is the value set in client's Email field (e.g., "user_UUID").
// resetCounter determines if the counter should be reset after querying.
func queryV2RayStats(client statsService.StatsServiceClient, userEmailTag string, resetCounter bool) (uplink int64, downlink int64, err error) {
	if client == nil {
		return 0, 0, fmt.Errorf("StatsServiceClient is nil")
	}
	req := &statsService.GetStatsRequest{
		Name:  fmt.Sprintf("user>>>%s>>>traffic>>>uplink", userEmailTag), // Pattern for user uplink
		Reset: resetCounter,
	}
	respUp, err := client.GetStats(context.Background(), req)
	if err != nil {
		// It's possible the stat doesn't exist yet if user had no traffic
		// V2Ray might return an error or an empty response. Handle gracefully.
		// log.Printf("Debug: GetStats uplink for %s: err %v, resp %v", userEmailTag, err, respUp)
		// For now, assume error means no data or actual error
	}
	if respUp != nil && respUp.Stat != nil {
		uplink = respUp.Stat.Value
	}

	req.Name = fmt.Sprintf("user>>>%s>>>traffic>>>downlink", userEmailTag) // Pattern for user downlink
	req.Reset = resetCounter // Reset for downlink should be same as uplink

	respDown, err := client.GetStats(context.Background(), req)
	if err != nil {
		// log.Printf("Debug: GetStats downlink for %s: err %v, resp %v", userEmailTag, err, respDown)
	}
	if respDown != nil && respDown.Stat != nil {
		downlink = respDown.Stat.Value
	}

	// If there was an error for either, return it (uplink error takes precedence for simplicity)
	// A more robust error handling might combine errors or check specific error types from V2Ray.
	if err != nil { // Prioritizing uplink error, but downlink might also have one.
		return uplink, downlink, fmt.Errorf("failed to get stats for %s (uplink or downlink): %v", userEmailTag, err)
	}

	return uplink, downlink, nil
}

// startTrafficMonitoringLoop periodically checks user traffic and deactivates them if limits are exceeded.
func startTrafficMonitoringLoop(grpcApiAddress string, checkInterval time.Duration, gcsBucket, gcsObject, v2rayPort string) {
	log.Printf("Starting traffic monitoring loop. gRPC API: %s, Interval: %s", grpcApiAddress, checkInterval)

	// Setup gRPC connection
	// Note: This connection is long-lived. If it breaks, the loop will continuously fail.
	// Production systems might need more robust connection handling (e.g., retry dialing).
	conn, err := grpc.Dial(grpcApiAddress, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to V2Ray gRPC API at %s: %v", grpcApiAddress, err)
		return // Should not happen with WithBlock if server is up, but good practice.
	}
	defer conn.Close()
	statsClient := statsService.NewStatsServiceClient(conn)
	log.Printf("Successfully connected to V2Ray gRPC API at %s", grpcApiAddress)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Traffic monitoring tick: checking user stats...")
			configMutex.Lock() // Lock for the entire check cycle to prevent concurrent API modifications

			var configChanged bool = false
			var needsV2RayRestart bool = false

			usersToUpdate := make(map[string]User) // Store users that need updating in currentUsersConfig

			for userID, user := range currentUsersConfig {
				if !user.IsActive {
					continue
				}

				userTag := "user_" + user.ID
				// Query stats and reset them on V2Ray's side
				uplink, downlink, err := queryV2RayStats(statsClient, userTag, true)
				if err != nil {
					// Log error but continue, as other users might be fine.
					// V2Ray returns an error if the stat entry doesn't exist (e.g. user had no traffic yet)
					// This is normal, so we might want to reduce log spam for "not found" errors.
					// For now, logging all errors from queryV2RayStats.
					log.Printf("WARN: Error querying stats for user %s (tag: %s): %v", userID, userTag, err)
					continue
				}

				currentPeriodTraffic := uplink + downlink
				if currentPeriodTraffic > 0 {
					log.Printf("Traffic for user %s (tag: %s): Uplink=%d, Downlink=%d, Total Current Period=%d", userID, userTag, uplink, downlink, currentPeriodTraffic)
					user.TrafficUsedBytes += currentPeriodTraffic
					configChanged = true
					log.Printf("User %s (tag: %s) updated TrafficUsedBytes to %d", userID, userTag, user.TrafficUsedBytes)
				}

				// Check against limit (GB to Bytes: limit * 1024^3)
				if user.TrafficUsedBytes >= int64(user.TrafficLimitGB*1024*1024*1024) {
					user.IsActive = false
					log.Printf("INFO: User %s (tag: %s) DEACTIVATED due to traffic limit. Used: %d bytes, Limit: %.2f GB",
						userID, userTag, user.TrafficUsedBytes, user.TrafficLimitGB)
					configChanged = true
					needsV2RayRestart = true // V2Ray needs to be reconfigured to remove/disable user
				}

				// Time limit check (only if user is still active)
				if user.IsActive && user.TimeLimitDays > 0 {
					expirationTime := user.CreatedAt.AddDate(0, 0, user.TimeLimitDays)
					if time.Now().UTC().After(expirationTime) {
						log.Printf("INFO: User %s (tag: %s) DEACTIVATED due to time limit. Created: %s, Limit: %d days, Expires: %s",
							userID, userTag, user.CreatedAt.Format(time.RFC3339), user.TimeLimitDays, expirationTime.Format(time.RFC3339))
						user.IsActive = false
						configChanged = true
						needsV2RayRestart = true
					}
				}
				usersToUpdate[userID] = user
			}

			// Apply updates to currentUsersConfig
			for uid, u := range usersToUpdate {
				currentUsersConfig[uid] = u
			}

			if configChanged {
				log.Println("Configuration changed due to traffic updates/deactivations.")
				// Create a deep copy for saving, to avoid holding lock during GCS operation for too long
				configToSave := make(UsersConfig)
				for k, v := range currentUsersConfig {
					configToSave[k] = v
				}
				// Unlock before GCS save and potential V2Ray restart
				configMutex.Unlock()

				if err := saveUsersConfig(gcsBucket, gcsObject, configToSave); err != nil {
					log.Printf("ERROR: Failed to save user config to GCS after traffic update/deactivation: %v", err)
				} else {
					log.Println("Successfully saved updated user config to GCS.")
				}

				if needsV2RayRestart {
					log.Println("V2Ray restart needed due to user deactivation.")
					if err := handleRestartV2Ray(v2rayPort); err != nil {
						log.Printf("ERROR: Failed to restart V2Ray after deactivating user(s): %v", err)
					} else {
						log.Println("V2Ray restarted successfully after user deactivation(s).")
					}
				}
			} else {
				configMutex.Unlock() // No changes, just unlock
				log.Println("Traffic monitoring tick: no reportable traffic changes or deactivations.")
			}
		// TODO: Add a quit channel to gracefully stop this goroutine if needed.
		// case <-quitChannel:
		// 	 log.Println("Stopping traffic monitoring loop.")
		// 	 return
		}
	}
}

// --- HTTP Handlers ---

// --- HTTP Handlers ---

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
			// Cannot send error to client as headers are already sent
		}
	}
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	usersList := []User{}
	for _, user := range currentUsersConfig {
		usersList = append(usersList, user)
	}
	writeJSONResponse(w, http.StatusOK, usersList)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "User ID is required"})
		return
	}

	configMutex.RLock()
	defer configMutex.RUnlock()

	user, exists := currentUsersConfig[userID]
	if !exists {
		writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}
	writeJSONResponse(w, http.StatusOK, user)
}

func createUserHandler(gcsBucket, gcsObject, v2rayPort string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newUser User
		if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
			return
		}

		// Validate required fields (example: TrafficLimitGB and TimeLimitDays)
		// More specific validation can be added here.
		if newUser.TrafficLimitGB <= 0 || newUser.TimeLimitDays <= 0 {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "TrafficLimitGB and TimeLimitDays must be positive values"})
			return
		}

		newUser.ID = uuid.NewString()
		newUser.CreatedAt = time.Now().UTC()
		newUser.IsActive = true      // Default to active
		newUser.TrafficUsedBytes = 0 // Initialize traffic used

		configMutex.Lock()
		if currentUsersConfig == nil {
			currentUsersConfig = make(UsersConfig)
		}
		currentUsersConfig[newUser.ID] = newUser
		// Create a deep copy for saving, to avoid holding lock during GCS operation for too long
		configToSave := make(UsersConfig)
		for k, v := range currentUsersConfig {
			configToSave[k] = v
		}
		configMutex.Unlock() // Unlock before potentially long I/O operations

		if err := saveUsersConfig(gcsBucket, gcsObject, configToSave); err != nil {
			log.Printf("ERROR: Failed to save user config to GCS: %v", err)
			// Potentially revert currentUsersConfig change or handle inconsistency?
			// For now, log and proceed with restart, but the state might be inconsistent.
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save configuration: " + err.Error()})
			return
		}

		if err := handleRestartV2Ray(v2rayPort); err != nil {
			log.Printf("ERROR: Failed to restart V2Ray after creating user: %v", err)
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to restart V2Ray: " + err.Error()})
			return
		}

		writeJSONResponse(w, http.StatusCreated, newUser)
	}
}

func updateUserHandler(gcsBucket, gcsObject, v2rayPort string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "User ID is required in query parameters"})
			return
		}

		var updatedUserData User
		if err := json.NewDecoder(r.Body).Decode(&updatedUserData); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body: " + err.Error()})
			return
		}

		configMutex.Lock()
		existingUser, exists := currentUsersConfig[userID]
		if !exists {
			configMutex.Unlock()
			writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "User not found"})
			return
		}

		// Update fields (ID and CreatedAt should not change)
		// Only update if new value is provided or has a meaningful zero value for the type
		if updatedUserData.TrafficLimitGB > 0 {
			existingUser.TrafficLimitGB = updatedUserData.TrafficLimitGB
		}
		if updatedUserData.TimeLimitDays > 0 {
			existingUser.TimeLimitDays = updatedUserData.TimeLimitDays
		}
		// For IsActive, even if false, it's a valid update
		// We need a way to distinguish "not provided" from "false" if IsActive was a pointer or using a wrapper struct.
		// Assuming direct update for IsActive for now.
		existingUser.IsActive = updatedUserData.IsActive

		// TrafficUsedBytes is usually updated internally, not by client, but allow if needed.
		// For this example, let's assume it can be reset or adjusted via API.
		if updatedUserData.TrafficUsedBytes >= 0 { // Allow reset to 0
			existingUser.TrafficUsedBytes = updatedUserData.TrafficUsedBytes
		}


		currentUsersConfig[userID] = existingUser

		configToSave := make(UsersConfig)
		for k, v := range currentUsersConfig {
			configToSave[k] = v
		}
		configMutex.Unlock() // Unlock before I/O

		if err := saveUsersConfig(gcsBucket, gcsObject, configToSave); err != nil {
			log.Printf("ERROR: Failed to save user config to GCS during update: %v", err)
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save configuration: " + err.Error()})
			return
		}

		if err := handleRestartV2Ray(v2rayPort); err != nil {
			log.Printf("ERROR: Failed to restart V2Ray after updating user: %v", err)
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to restart V2Ray: " + err.Error()})
			return
		}
		writeJSONResponse(w, http.StatusOK, existingUser)
	}
}

func deleteUserHandler(gcsBucket, gcsObject, v2rayPort string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("id")
		if userID == "" {
			writeJSONResponse(w, http.StatusBadRequest, map[string]string{"error": "User ID is required in query parameters"})
			return
		}

		configMutex.Lock()
		_, exists := currentUsersConfig[userID]
		if !exists {
			configMutex.Unlock()
			writeJSONResponse(w, http.StatusNotFound, map[string]string{"error": "User not found"})
			return
		}

		delete(currentUsersConfig, userID)

		configToSave := make(UsersConfig)
		for k, v := range currentUsersConfig {
			configToSave[k] = v
		}
		configMutex.Unlock() // Unlock before I/O

		if err := saveUsersConfig(gcsBucket, gcsObject, configToSave); err != nil {
			log.Printf("ERROR: Failed to save user config to GCS after deleting user: %v", err)
			// If saving fails, the user is deleted in memory but not in GCS.
			// This leads to inconsistency. Consider how to handle this.
			// For now, we'll log the error and proceed with V2Ray restart based on in-memory state.
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save configuration: " + err.Error()})
			return
		}

		if err := handleRestartV2Ray(v2rayPort); err != nil {
			log.Printf("ERROR: Failed to restart V2Ray after deleting user: %v", err)
			writeJSONResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to restart V2Ray: " + err.Error()})
			return
		}
		writeJSONResponse(w, http.StatusNoContent, nil)
	}
}

func main() {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	objectName := os.Getenv("GCS_OBJECT_NAME")

	if bucketName == "" || objectName == "" {
		log.Fatal("GCS_BUCKET_NAME and GCS_OBJECT_NAME environment variables must be set")
	}

	log.Printf("Loading users from gs://%s/%s", bucketName, objectName)
	users, err := loadUsersConfig(bucketName, objectName)
	if err != nil {
		log.Fatalf("Failed to load users config: %v", err)
	}
	log.Printf("Loaded %d users", len(users))

	// Add a test user
	testUserID := uuid.NewString()
	users[testUserID] = User{
		ID:             testUserID,
		TrafficLimitGB: 10,
		TimeLimitDays:  30,
		CreatedAt:      time.Now(),
		IsActive:       true,
	}
	log.Printf("Added test user: %s", testUserID)

	log.Printf("Saving users to gs://%s/%s", bucketName, objectName)
	if err := saveUsersConfig(bucketName, objectName, users); err != nil {
		log.Fatalf("Failed to save users config: %v", err)
	}
	log.Println("Successfully saved users config.")

	// Get GCS and V2Ray port details from environment
	gcsBucketName := os.Getenv("GCS_BUCKET_NAME")
	gcsObjectName := os.Getenv("GCS_OBJECT_NAME")
	v2rayPort := os.Getenv("PORT") // PORT is for V2Ray service itself

	if gcsBucketName == "" || gcsObjectName == "" {
		log.Fatal("GCS_BUCKET_NAME and GCS_OBJECT_NAME environment variables must be set")
	}
	if v2rayPort == "" {
		v2rayPort = "8080" // Default V2Ray port
		log.Printf("PORT environment variable for V2Ray not set, using default %s", v2rayPort)
	}

	// Load initial users config
	log.Printf("Loading initial users config from gs://%s/%s", gcsBucketName, gcsObjectName)
	loadedUsers, err := loadUsersConfig(gcsBucketName, gcsObjectName)
	if err != nil {
		log.Fatalf("Failed to load initial users config: %v", err)
	}
	currentUsersConfig = loadedUsers // Assign to global
	log.Printf("Loaded %d users initially.", len(currentUsersConfig))

	// Initial V2Ray start
	go func() {
		log.Println("Starting initial V2Ray process...")
		if err := handleRestartV2Ray(v2rayPort); err != nil {
			log.Fatalf("FATAL: Failed to start initial V2Ray process: %v", err)
		}
	}()

	// Setup HTTP API server
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8000" // Default API port
		log.Printf("API_PORT environment variable not set, using default %s", apiPort)
	}

	// Using standard http.ServeMux and http.HandleFunc with closures
	mux := http.NewServeMux()
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUsersHandler(w, r)
		case http.MethodPost:
			createUserHandler(gcsBucketName, gcsObjectName, v2rayPort)(w, r)
		default:
			writeJSONResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		}
	})
	// For specific user, expecting /user?id=xxx
	// A more RESTful approach would be /users/{id}, requiring gorilla/mux or similar
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getUserHandler(w, r)
		case http.MethodPut:
			updateUserHandler(gcsBucketName, gcsObjectName, v2rayPort)(w, r)
		case http.MethodDelete:
			deleteUserHandler(gcsBucketName, gcsObjectName, v2rayPort)(w, r)
		default:
			writeJSONResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		}
	})

	log.Printf("Starting API server on port %s", apiPort)
	if err := http.ListenAndServe(":"+apiPort, mux); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	// Define V2Ray gRPC API address and traffic check interval
	v2rayGrpcApiAddress := "127.0.0.1:10085" // As configured in generateV2RayConfig

	trafficCheckIntervalStr := os.Getenv("TRAFFIC_CHECK_INTERVAL_SECONDS")
	trafficCheckInterval := 5 * time.Minute // Default
	if secs, err := time.ParseDuration(trafficCheckIntervalStr + "s"); err == nil && secs > 0 {
		trafficCheckInterval = secs
		log.Printf("Using custom traffic check interval: %s", trafficCheckInterval)
	} else if trafficCheckIntervalStr != "" {
		log.Printf("WARN: Invalid TRAFFIC_CHECK_INTERVAL_SECONDS value '%s', using default %s", trafficCheckIntervalStr, trafficCheckInterval)
	}


	// Start traffic monitoring loop
	go startTrafficMonitoringLoop(v2rayGrpcApiAddress, trafficCheckInterval, gcsBucketName, gcsObjectName, v2rayPort)

	log.Printf("Starting API server on port %s", apiPort)
	if err := http.ListenAndServe(":"+apiPort, mux); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
	// ListenAndServe is blocking, so main goroutine will stay alive.
}

// generateV2RayConfig creates a V2Ray JSON configuration.
func generateV2RayConfig(users UsersConfig, port string) ([]byte, error) {
	v2rayClients := []Client{}
	const userLevel = 0 // Define user level for policy

	activeUsers := 0
	for _, user := range users {
		if user.IsActive {
			v2rayClients = append(v2rayClients, Client{
				ID:      user.ID,
				AlterID: 0, // Standard AlterID for VMess
				Email:   "user_" + user.ID, // Tag for stats: "user_UUID"
				Level:   userLevel,
			})
			activeUsers++
		}
	}

	if activeUsers == 0 {
		log.Println("No active users found in config. Generating a default user for V2Ray.")
		defaultUserID := uuid.NewString()
		v2rayClients = append(v2rayClients, Client{
			ID:      defaultUserID,
			AlterID: 0,
			Email:   "user_" + defaultUserID, // Consistent email format for stats
			Level:   userLevel, // Use the defined userLevel
		})
		log.Printf("Default user ID: %s, Email for stats: user_%s", defaultUserID, defaultUserID)
	} else {
		log.Printf("Using %d active user(s) for V2Ray config.", len(v2rayClients))
	}

	apiTag := "API" // Tag for API inbound and routing

	config := Config{
		Log: LogEntry{
			Loglevel: "warning", // "debug" for more verbosity if needed
			Access:   "/dev/stdout",
			Error:    "/dev/stderr",
		},
		API: &APIConfig{
			Tag:      apiTag,
			Services: []string{"StatsService"}, // Enable StatsService
		},
		Policy: &PolicyConfig{
			Levels: map[string]LevelPolicy{
				fmt.Sprintf("%d", userLevel): { // Policy for userLevel (e.g., "0")
					StatsUserUplink:   true,
					StatsUserDownlink: true,
				},
			},
			System: SystemPolicy{ // System stats can be disabled
				StatsInboundUplink:   false,
				StatsInboundDownlink: false,
			},
		},
		Routing: &RoutingConfig{
			Rules: []RoutingRule{
				{
					Type:        "field",
					InboundTag:  []string{apiTag}, // Traffic from API inbound
					OutboundTag: apiTag,           // Route to API service itself
					// Domain and Protocol can be nil/empty for this rule
				},
				// Potentially other rules, e.g., for blocking ads or specific sites
			},
		},
		Inbounds: []Inbound{
			{
				Port:     port,
				Protocol: "vmess",
				Settings: InboundSettings{
					Clients: v2rayClients,
				},
				StreamSettings: StreamSettings{
					Network:  "ws",
					Security: "none", // TLS is handled by Cloud Run
					WSSettings: WebSocketSettings{
						Path: "/ws", // Standard WebSocket path
					},
				},
				Tag: "vmess-in", // Main inbound for user traffic
			},
			{ // Inbound for V2Ray API
				Port:     "10085", // Local port for API
				Listen:   "127.0.0.1",  // Listen on localhost only
				Protocol: "dokodemo-door",
				Settings: InboundSettings{ // Basic settings for dokodemo-door
					// Address should ideally be the API's intended listening address if it were external,
					// but for routing to internal services like StatsService, it's often 127.0.0.1.
					// V2Ray handles this internally based on the tag.
					// No clients needed for dokodemo-door API inbound
				},
				Tag: apiTag, // Tag this inbound as "API"
			},
		},
		Outbounds: []Outbound{
			{
				Protocol: "freedom",
				Settings: OutboundSettings{},
				Tag:      "direct-out", // Default outbound
			},
			{ // Outbound for API routing rule
				Protocol: "blackhole", // Can be blackhole as it's handled by API service
				Settings: OutboundSettings{}, // Empty settings for blackhole
				Tag:      apiTag,       // Must match outboundTag in API routing rule
			},
		},
	}

	configBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal V2Ray config: %v", err)
	}

	return configBytes, nil
}

// handleRestartV2Ray stops the current V2Ray process (if any) and starts a new one.
// It assumes configMutex is NOT held by the caller, as it will acquire it.
func handleRestartV2Ray(port string) error {
	v2rayRestartMutex.Lock() // Serialize V2Ray restarts
	defer v2rayRestartMutex.Unlock()

	// Stop existing V2Ray process
	if v2rayCmd != nil && v2rayCmd.Process != nil {
		log.Println("Stopping existing V2Ray process...")
		if err := v2rayCmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Failed to send interrupt signal to V2Ray process: %v. Attempting to kill.", err)
			if killErr := v2rayCmd.Process.Kill(); killErr != nil {
				log.Printf("Failed to kill V2Ray process: %v", killErr)
				// Continue, as we might still be ableto start a new one if the old one is defunct
			}
		}
		// Wait for a short period to allow the process to exit gracefully
		// This part can be tricky. A proper solution might involve a channel or cmd.Wait() in a goroutine.
		// For now, a small sleep, but this is not robust.
		// On second thought, cmd.Wait() is called in a goroutine below for the new process.
		// For the old process, we've sent a signal. If it doesn't die, the new one might fail to bind port.
		// This needs careful handling in a production system.
		// Let's assume Signal/Kill is usually effective.
		log.Println("Sent stop signal to V2Ray process.")
		// We'll rely on the OS to clean up if the process didn't exit immediately.
		// Or the new process might fail if the port is still in use.
	}

	configMutex.RLock()
	usersToConfigure := make(UsersConfig) // Create a deep copy for thread safety
	for k, v := range currentUsersConfig {
		usersToConfigure[k] = v
	}
	configMutex.RUnlock()

	log.Println("Generating new V2Ray config for restart...")
	v2rayConfigBytes, err := generateV2RayConfig(usersToConfigure, port)
	if err != nil {
		return fmt.Errorf("failed to generate V2Ray config for restart: %v", err)
	}

	v2rayConfigPath := "/tmp/v2ray_config.json" // Same path as used in main
	if err := ioutil.WriteFile(v2rayConfigPath, v2rayConfigBytes, 0644); err != nil {
		return fmt.Errorf("failed to write V2Ray config to %s for restart: %v", v2rayConfigPath, err)
	}
	log.Printf("New V2Ray config written to %s for restart", v2rayConfigPath)

	log.Println("Starting new V2Ray process...")
	newCmd := exec.Command("v2ray", "-config", v2rayConfigPath)
	newCmd.Stdout = log.Output()
	newCmd.Stderr = log.Output()

	if err := newCmd.Start(); err != nil {
		return fmt.Errorf("failed to start new V2Ray process: %v", err)
	}
	v2rayCmd = newCmd // Store the new command
	log.Printf("New V2Ray process started with PID: %d", v2rayCmd.Process.Pid)

	// Goroutine to wait for the command to complete and log its exit
	go func() {
		if v2rayCmd == nil || v2rayCmd.Process == nil {
			log.Println("V2Ray command or process is nil in Wait goroutine, cannot Wait.")
			return
		}
		processState, err := v2rayCmd.Process.Wait()
		pid := -1
		if processState != nil { // processState can be nil if Start() failed but Process was set
			pid = processState.Pid()
		} else if v2rayCmd.Process != nil { // Fallback to process if state is nil
		    pid = v2rayCmd.Process.Pid
		}


		if err != nil {
			log.Printf("V2Ray process (PID: %d) finished with error: %v. Status: %s", pid, err, processState)
		} else {
			log.Printf("V2Ray process (PID: %d) finished successfully. Status: %s", pid, processState)
		}
	}()

	return nil
}
