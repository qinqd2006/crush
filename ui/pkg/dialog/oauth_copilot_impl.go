package dialog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/qinqd2006/crush/kernel/pkg"
)

const (
	copilotClientID       = "Iv1.b507a08c87ecfe98"
	copilotUserAgent      = "GitHubCopilotChat/0.32.4"
	copilotDeviceCodeURL  = "https://github.com/login/device/code"
	copilotAccessTokenURL = "https://github.com/login/oauth/access_token"
	copilotTokenURL       = "https://api.github.com/copilot_internal/v2/token"
)

// copilotDeviceCode represents the device code response from GitHub.
type copilotDeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// copilotTokenResponse represents the token response from GitHub.
type copilotTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// requestCopilotDeviceCode initiates the device code flow with GitHub.
func requestCopilotDeviceCode(ctx context.Context) (*copilotDeviceCode, error) {
	data := url.Values{}
	data.Set("client_id", copilotClientID)
	data.Set("scope", "read:user")

	req, err := http.NewRequestWithContext(ctx, "POST", copilotDeviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", copilotUserAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed: %s - %s", resp.Status, string(body))
	}

	var dc copilotDeviceCode
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, err
	}
	return &dc, nil
}

// pollCopilotToken polls GitHub for the access token after user authorization.
func pollCopilotToken(ctx context.Context, dc *copilotDeviceCode) (*kernel.OAuthToken, error) {
	interval := max(dc.Interval, 5)
	deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		data := url.Values{}
		data.Set("client_id", copilotClientID)
		data.Set("device_code", dc.DeviceCode)
		data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequestWithContext(ctx, "POST", copilotAccessTokenURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", copilotUserAgent)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusBadRequest {
			var errResp struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description"`
			}
			if err := json.Unmarshal(body, &errResp); err != nil {
				return nil, err
			}
			if errResp.Error == "authorization_pending" {
				continue
			}
			if errResp.Error == "slow_down" {
				interval = min(interval+5, 60)
				ticker.Reset(time.Duration(interval) * time.Second)
				continue
			}
			return nil, fmt.Errorf("%s: %s", errResp.Error, errResp.ErrorDescription)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("token request failed: %s - %s", resp.Status, string(body))
		}

		var tokenResp copilotTokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return nil, err
		}

		// Exchange for copilot-specific token
		copilotToken, err := exchangeCopilotToken(ctx, tokenResp.AccessToken)
		if err != nil {
			return nil, err
		}

		return copilotToken, nil
	}

	return nil, fmt.Errorf("token request timed out")
}

// exchangeCopilotToken exchanges the GitHub token for a Copilot-specific token.
func exchangeCopilotToken(ctx context.Context, githubToken string) (*kernel.OAuthToken, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", copilotTokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("User-Agent", copilotUserAgent)
	req.Header.Set("Editor-Version", "vscode/1.105.1")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.32.4")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("copilot token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &kernel.OAuthToken{
		AccessToken: tokenResp.Token,
		ExpiresAt:   tokenResp.ExpiresAt,
		TokenType:   "Bearer",
	}, nil
}
