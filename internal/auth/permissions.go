package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PermissionClient calls the creaminds Permission Service for object-level checks.
type PermissionClient struct {
	baseURL string
	http    *http.Client
}

// NewPermissionClient creates a client for the Permission Service.
// Returns nil when serviceURL is empty (feature disabled).
func NewPermissionClient(serviceURL string) *PermissionClient {
	if serviceURL == "" {
		return nil
	}
	return &PermissionClient{
		baseURL: serviceURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type checkRequest struct {
	UserID       string `json:"user_id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Action       string `json:"action"`
}

type checkResponse struct {
	Allowed bool `json:"allowed"`
}

// Check returns whether the user may perform action on resource_type/resource_id.
//
// When the client is nil (Permission Service not configured) it returns true
// — galaxy access is public by default (OE-1 decision: public default).
func (c *PermissionClient) Check(
	ctx context.Context,
	bearerToken string,
	userID, resourceType, resourceID, action string,
) (bool, error) {
	if c == nil {
		return true, nil // public by default
	}

	body, err := json.Marshal(checkRequest{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
	})
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/permissions/check", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("permission service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("permission service: status %d", resp.StatusCode)
	}

	var result checkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Allowed, nil
}
