package dialog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qinqd2006/crush/kernel/pkg"
)

const (
	hyperBaseURLConst = "https://hyper.charm.land"
	hyperUserAgent    = "crush"
)

// hyperDeviceAuthResponse contains the response from the device authorization endpoint.
type hyperDeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
}

// hyperTokenResponse contains the response from the polling endpoint.
type hyperTokenResponse struct {
	RefreshToken     string `json:"refresh_token,omitempty"`
	UserID           string `json:"user_id"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// hyperAccessTokenResponse contains the response from the token exchange endpoint.
type hyperAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
}

// hyperIntrospectResponse contains the response from the token introspection endpoint.
type hyperIntrospectResponse struct {
	Active bool   `json:"active"`
	Sub    string `json:"sub,omitempty"`
	OrgID  string `json:"org_id,omitempty"`
	Exp    int64  `json:"exp,omitempty"`
	Iat    int64  `json:"iat,omitempty"`
	Iss    string `json:"iss,omitempty"`
	Jti    string `json:"jti,omitempty"`
}

// hyperBaseURL returns the base URL for Hyper API.
func hyperBaseURL() string {
	if url := os.Getenv("HYPER_URL"); url != "" {
		return url
	}
	return hyperBaseURLConst
}

// hyperDeviceName returns the device name for Hyper OAuth.
func hyperDeviceName() string {
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return "Crush (" + hostname + ")"
	}
	return "Crush"
}

// initiateHyperDeviceAuth calls the /device/auth endpoint to start the device flow.
func initiateHyperDeviceAuth(ctx context.Context) (*hyperDeviceAuthResponse, error) {
	url := hyperBaseURL() + "/device/auth"

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, url,
		strings.NewReader(fmt.Sprintf(`{"device_name":%q}`, hyperDeviceName())),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", hyperUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth failed: status %d, body %q", resp.StatusCode, string(body))
	}

	var authResp hyperDeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &authResp, nil
}

// pollHyperToken polls the /device/token endpoint until authorization is complete.
func pollHyperToken(ctx context.Context, deviceCode string, expiresIn int) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(expiresIn)*time.Second)
	defer cancel()

	d := 5 * time.Second
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			result, err := pollHyperTokenOnce(ctx, deviceCode)
			if err != nil {
				return "", err
			}
			if result.RefreshToken != "" {
				return result.RefreshToken, nil
			}
			if result.Error == "authorization_pending" {
				continue
			}
			if result.Error != "" {
				return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDescription)
			}
		}
	}
}

func pollHyperTokenOnce(ctx context.Context, deviceCode string) (hyperTokenResponse, error) {
	var result hyperTokenResponse
	url := fmt.Sprintf("%s/device/auth/%s", hyperBaseURL(), deviceCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", hyperUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return result, fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("unmarshal response: %w: %s", err, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("token request failed: status %d body %q", resp.StatusCode, string(body))
	}

	return result, nil
}

// exchangeHyperToken exchanges a refresh token for an access token.
func exchangeHyperToken(ctx context.Context, refreshToken string) (*kernel.OAuthToken, error) {
	reqBody := map[string]string{
		"refresh_token": refreshToken,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := hyperBaseURL() + "/token/exchange"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", hyperUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d body %q", resp.StatusCode, string(body))
	}

	var tokenResp hyperAccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &kernel.OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    tokenResp.ExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

// introspectHyperToken validates an access token using the introspection endpoint.
func introspectHyperToken(ctx context.Context, accessToken string) (*hyperIntrospectResponse, error) {
	reqBody := map[string]string{
		"token": accessToken,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := hyperBaseURL() + "/token/introspect"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", hyperUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token introspection failed: status %d body %q", resp.StatusCode, string(body))
	}

	var result hyperIntrospectResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
