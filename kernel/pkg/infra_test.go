package kernel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeConstants(t *testing.T) {
	assert.Equal(t, Scope("global"), ScopeGlobal)
	assert.Equal(t, Scope("project"), ScopeProject)
	assert.Equal(t, Scope("session"), ScopeSession)
}

func TestModelTypeConstants(t *testing.T) {
	assert.Equal(t, ModelType("main"), ModelTypeMain)
	assert.Equal(t, ModelType("small"), ModelTypeSmall)
}

func TestSkillEventTypeConstants(t *testing.T) {
	assert.Equal(t, SkillEventType("loaded"), SkillEventLoaded)
	assert.Equal(t, SkillEventType("unloaded"), SkillEventUnloaded)
}

func TestMCPEventTypeConstants(t *testing.T) {
	assert.Equal(t, MCPEventType("state_changed"), MCPEventStateChanged)
	assert.Equal(t, MCPEventType("tools_list_changed"), MCPEventToolsListChanged)
	assert.Equal(t, MCPEventType("prompts_list_changed"), MCPEventPromptsListChanged)
	assert.Equal(t, MCPEventType("resources_list_changed"), MCPEventResourcesListChanged)
}

func TestLSPEventTypeConstants(t *testing.T) {
	assert.Equal(t, LSPEventType("state_changed"), LSPEventStateChanged)
	assert.Equal(t, LSPEventType("diagnostics_changed"), LSPEventDiagnosticsChanged)
}

func TestNotificationTypeConstants(t *testing.T) {
	assert.Equal(t, NotificationType("agent_finished"), NotificationAgentFinished)
	assert.Equal(t, NotificationType("re_authenticate"), NotificationReAuthenticate)
}

func TestMCPStateConstants(t *testing.T) {
	assert.Equal(t, MCPState(0), MCPStateDisabled)
	assert.Equal(t, MCPState(1), MCPStateStarting)
	assert.Equal(t, MCPState(2), MCPStateConnected)
	assert.Equal(t, MCPState(3), MCPStateError)
}

func TestLSPStateConstants(t *testing.T) {
	assert.Equal(t, LSPState(0), LSPStateUnstarted)
	assert.Equal(t, LSPState(1), LSPStateStarting)
	assert.Equal(t, LSPState(2), LSPStateRunning)
	assert.Equal(t, LSPState(3), LSPStateError)
}

func TestSkillEvent(t *testing.T) {
	evt := SkillEvent{
		Type: SkillEventLoaded,
		Skill: Skill{
			Name:        "test-skill",
			Description: "A test skill",
			Path:        "/path/to/skill",
			Commands: []SkillCommand{
				{Name: "cmd1", Description: "command 1"},
			},
		},
	}
	assert.Equal(t, SkillEventLoaded, evt.Type)
	assert.Equal(t, "test-skill", evt.Skill.Name)
	assert.Len(t, evt.Skill.Commands, 1)
}

func TestOAuthToken(t *testing.T) {
	token := OAuthToken{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		ExpiresAt:    1234567890,
		TokenType:    "Bearer",
	}
	assert.Equal(t, "access123", token.AccessToken)
	assert.Equal(t, "refresh456", token.RefreshToken)
	assert.Equal(t, int64(1234567890), token.ExpiresAt)
	assert.Equal(t, "Bearer", token.TokenType)
}

func TestProgramInterface(t *testing.T) {
	// Program is an interface - verify it exists and has Send method
	var p Program = &testProgram{}
	assert.NotNil(t, p)
}

type testProgram struct{}

func (t *testProgram) Send(msg any) {}

func TestSkill(t *testing.T) {
	s := Skill{
		Name:        "my-skill",
		Description: "Test skill",
		Path:        "/skills/my-skill",
		Commands: []SkillCommand{
			{Name: "hello", Description: "Say hello", Args: []string{"name"}},
		},
	}
	assert.Equal(t, "my-skill", s.Name)
	assert.Equal(t, "Test skill", s.Description)
	assert.Len(t, s.Commands, 1)
	assert.Equal(t, "hello", s.Commands[0].Name)
}

func TestSkillCommand(t *testing.T) {
	cmd := SkillCommand{
		Name:        "greet",
		Description: "Greet someone",
		Args:        []string{"name", "greeting"},
	}
	assert.Equal(t, "greet", cmd.Name)
	assert.Equal(t, "Greet someone", cmd.Description)
	assert.Len(t, cmd.Args, 2)
}
