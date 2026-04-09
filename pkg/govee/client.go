package govee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://openapi.api.govee.com/router/api/v1"
)

// Client handles communication with the Govee API
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Govee API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: BaseURL,
	}
}

// Device represents a Govee device
type Device struct {
	DeviceID   string `json:"device"`
	DeviceName string `json:"deviceName"`
	Model      string `json:"model"`
	Controllable bool `json:"controllable"`
	Retrievable bool `json:"retrievable"`
	SupportCmds []string `json:"supportCmds"`
}

// DevicesResponse represents the response from the devices endpoint
type DevicesResponse struct {
	Code    int      `json:"code"`
	Message string   `json:"message"`
	Data    []Device `json:"data"`
}

// DeviceState represents the state of a device
type DeviceState struct {
	DeviceID string                 `json:"device"`
	Model    string                 `json:"model"`
	Properties []map[string]interface{} `json:"properties"`
}

// DeviceStateResponse represents the response from the device state endpoint
type DeviceStateResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    DeviceState `json:"data"`
}

// GetDevices retrieves a list of all devices
func (c *Client) GetDevices(ctx context.Context) ([]Device, error) {
	url := fmt.Sprintf("%s/user/devices", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Govee-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var devicesResp DevicesResponse
	if err := json.Unmarshal(body, &devicesResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if devicesResp.Code != 200 {
		return nil, fmt.Errorf("API returned error code %d: %s", devicesResp.Code, devicesResp.Message)
	}

	return devicesResp.Data, nil
}

// GetDeviceState retrieves the current state of a specific device
func (c *Client) GetDeviceState(ctx context.Context, deviceID string, model string) (*DeviceState, error) {
	url := fmt.Sprintf("%s/user/devices/state?device=%s&model=%s", c.baseURL, deviceID, model)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Govee-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var stateResp DeviceStateResponse
	if err := json.Unmarshal(body, &stateResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if stateResp.Code != 200 {
		return nil, fmt.Errorf("API returned error code %d: %s", stateResp.Code, stateResp.Message)
	}

	return &stateResp.Data, nil
}

