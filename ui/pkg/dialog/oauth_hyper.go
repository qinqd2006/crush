package dialog

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/qinqd2006/crush/kernel/pkg"
	"github.com/qinqd2006/crush/ui/pkg/common"
)

func NewOAuthHyper(
	com *common.Common,
	isOnboarding bool,
	provider kernel.Provider,
	model kernel.SelectedModel,
	modelType kernel.SelectedModelType,
) (*OAuth, tea.Cmd) {
	return newOAuth(com, isOnboarding, provider, model, modelType, &OAuthHyper{})
}

type OAuthHyper struct {
	cancelFunc func()
}

var _ OAuthProvider = (*OAuthHyper)(nil)

func (m *OAuthHyper) name() string {
	return "Hyper"
}

func (m *OAuthHyper) initiateAuth() tea.Msg {
	minimumWait := 750 * time.Millisecond
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	authResp, err := initiateHyperDeviceAuth(ctx)

	ellapsed := time.Since(startTime)
	if ellapsed < minimumWait {
		time.Sleep(minimumWait - ellapsed)
	}

	if err != nil {
		return ActionOAuthErrored{fmt.Errorf("failed to initiate device auth: %w", err)}
	}

	return ActionInitiateOAuth{
		DeviceCode:      authResp.DeviceCode,
		UserCode:        authResp.UserCode,
		ExpiresIn:       authResp.ExpiresIn,
		VerificationURL: authResp.VerificationURL,
	}
}

func (m *OAuthHyper) startPolling(deviceCode string, expiresIn int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		refreshToken, err := pollHyperToken(ctx, deviceCode, expiresIn)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return ActionOAuthErrored{err}
		}

		token, err := exchangeHyperToken(ctx, refreshToken)
		if err != nil {
			return ActionOAuthErrored{fmt.Errorf("token exchange failed: %w", err)}
		}

		introspect, err := introspectHyperToken(ctx, token.AccessToken)
		if err != nil {
			return ActionOAuthErrored{fmt.Errorf("token introspection failed: %w", err)}
		}
		if !introspect.Active {
			return ActionOAuthErrored{fmt.Errorf("access token is not active")}
		}

		return ActionCompleteOAuth{token}
	}
}

func (m *OAuthHyper) stopPolling() tea.Msg {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	return nil
}
