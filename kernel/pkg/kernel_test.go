package kernel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessage_Content(t *testing.T) {
	t.Run("returns text content when present", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello world"},
			},
		}
		assert.Equal(t, "hello world", m.Content().Text)
	})

	t.Run("returns empty when no text content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ToolCallContent{ID: "1", Name: "bash"},
			},
		}
		assert.Equal(t, "", m.Content().Text)
	})

	t.Run("returns empty for empty parts", func(t *testing.T) {
		m := Message{}
		assert.Equal(t, "", m.Content().Text)
	})
}

func TestMessage_ToolCalls(t *testing.T) {
	t.Run("extracts tool calls", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "I'll run a command"},
				ToolCallContent{ID: "call_1", Name: "bash", Input: `{"cmd": "ls"}`},
				ToolResultContent{ToolCallID: "call_1", Content: "file1\nfile2"},
			},
		}
		calls := m.ToolCalls()
		assert.Len(t, calls, 1)
		assert.Equal(t, "call_1", calls[0].ID)
		assert.Equal(t, "bash", calls[0].Name)
	})

	t.Run("returns empty when no tool calls", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello"},
			},
		}
		assert.Empty(t, m.ToolCalls())
	})
}

func TestMessage_ToolResults(t *testing.T) {
	t.Run("extracts tool results", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ToolCallContent{ID: "call_1", Name: "bash"},
				ToolResultContent{ToolCallID: "call_1", Content: "done", IsError: false},
			},
		}
		results := m.ToolResults()
		assert.Len(t, results, 1)
		assert.Equal(t, "call_1", results[0].ToolCallID)
		assert.False(t, results[0].IsError)
	})

	t.Run("extracts multiple tool results", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ToolResultContent{ToolCallID: "call_1", Content: "result1", IsError: false},
				ToolResultContent{ToolCallID: "call_2", Content: "error", IsError: true},
			},
		}
		results := m.ToolResults()
		assert.Len(t, results, 2)
		assert.True(t, results[1].IsError)
	})
}

func TestMessage_FinishPart(t *testing.T) {
	t.Run("returns finish content when present", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				FinishContent{Reason: FinishReasonEndTurn, Message: "done"},
			},
		}
		fp := m.FinishPart()
		assert.NotNil(t, fp)
		assert.Equal(t, FinishReasonEndTurn, fp.Reason)
		assert.Equal(t, "done", fp.Message)
	})

	t.Run("returns nil when no finish content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello"},
			},
		}
		assert.Nil(t, m.FinishPart())
	})
}

func TestMessage_FinishReason(t *testing.T) {
	tests := []struct {
		name     string
		parts    []ContentPart
		expected FinishReason
	}{
		{
			name:     "returns end_turn reason",
			parts:    []ContentPart{FinishContent{Reason: FinishReasonEndTurn}},
			expected: FinishReasonEndTurn,
		},
		{
			name:     "returns tool_use reason",
			parts:    []ContentPart{FinishContent{Reason: FinishReasonToolUse}},
			expected: FinishReasonToolUse,
		},
		{
			name:     "returns empty when no finish",
			parts:    []ContentPart{TextContent{Text: "hello"}},
			expected: "",
		},
		{
			name:     "returns empty for empty message",
			parts:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{Parts: tt.parts}
			assert.Equal(t, tt.expected, m.FinishReason())
		})
	}
}

func TestMessage_IsThinking(t *testing.T) {
	t.Run("returns true when reasoning content present", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ReasoningContent{Thinking: "let me think..."},
			},
		}
		assert.True(t, m.IsThinking())
	})

	t.Run("returns false when no reasoning content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello"},
			},
		}
		assert.False(t, m.IsThinking())
	})

	t.Run("returns false for empty parts", func(t *testing.T) {
		m := Message{}
		assert.False(t, m.IsThinking())
	})
}

func TestMessage_ReasoningContent(t *testing.T) {
	t.Run("extracts reasoning content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ReasoningContent{Thinking: "analysis...", Signature: "sig123", StartedAt: 1000, FinishedAt: 1050},
			},
		}
		rc := m.ReasoningContent()
		assert.Equal(t, "analysis...", rc.Thinking)
		assert.Equal(t, "sig123", rc.Signature)
		assert.Equal(t, int64(1000), rc.StartedAt)
		assert.Equal(t, int64(1050), rc.FinishedAt)
	})

	t.Run("returns empty for no reasoning", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{TextContent{Text: "hello"}},
		}
		rc := m.ReasoningContent()
		assert.Equal(t, "", rc.Thinking)
	})
}

func TestMessage_ImageURLContents(t *testing.T) {
	t.Run("extracts image URL contents", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "here is an image"},
				ImageURLContent{URL: "https://example.com/img.png", Detail: "high"},
			},
		}
		imgs := m.ImageURLContents()
		assert.Len(t, imgs, 1)
		assert.Equal(t, "https://example.com/img.png", imgs[0].URL)
		assert.Equal(t, "high", imgs[0].Detail)
	})

	t.Run("returns empty when no image URLs", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{TextContent{Text: "hello"}},
		}
		assert.Empty(t, m.ImageURLContents())
	})
}

func TestMessage_BinaryContents(t *testing.T) {
	t.Run("extracts binary contents", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				BinaryContent{Path: "/tmp/img.png", MIMEType: "image/png", Data: []byte{0x89, 0x50, 0x4E}},
			},
		}
		bin := m.BinaryContents()
		assert.Len(t, bin, 1)
		assert.Equal(t, "/tmp/img.png", bin[0].Path)
	})

	t.Run("returns empty when no binary", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{TextContent{Text: "hello"}},
		}
		assert.Empty(t, m.BinaryContents())
	})
}

func TestMessage_IsImage(t *testing.T) {
	t.Run("returns true for image URL content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ImageURLContent{URL: "https://example.com/img.png"},
			},
		}
		assert.True(t, m.IsImage())
	})

	t.Run("returns true for binary image content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				BinaryContent{Path: "/tmp/img.png", MIMEType: "image/png", Data: []byte{0x89}},
			},
		}
		assert.True(t, m.IsImage())
	})

	t.Run("returns false for non-image binary content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				BinaryContent{Path: "/tmp/doc.pdf", MIMEType: "application/pdf", Data: []byte{0x25}},
			},
		}
		assert.False(t, m.IsImage())
	})

	t.Run("returns false for text content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello"},
			},
		}
		assert.False(t, m.IsImage())
	})

	t.Run("returns false for empty parts", func(t *testing.T) {
		m := Message{}
		assert.False(t, m.IsImage())
	})
}

func TestMessage_IsText(t *testing.T) {
	t.Run("returns true when text content present", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "hello world"},
			},
		}
		assert.True(t, m.IsText())
	})

	t.Run("returns false when no text content", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ToolCallContent{ID: "1", Name: "bash"},
			},
		}
		assert.False(t, m.IsText())
	})

	t.Run("returns false for empty parts", func(t *testing.T) {
		m := Message{}
		assert.False(t, m.IsText())
	})
}

func TestMessage_IsFinished(t *testing.T) {
	t.Run("returns true when finish part present", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				FinishContent{Reason: FinishReasonEndTurn},
			},
		}
		assert.True(t, m.IsFinished())
	})

	t.Run("returns false when no finish part", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				TextContent{Text: "thinking..."},
			},
		}
		assert.False(t, m.IsFinished())
	})
}

func TestMessage_ThinkingDuration(t *testing.T) {
	t.Run("calculates duration from started to finished", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ReasoningContent{StartedAt: 1000, FinishedAt: 1050},
			},
		}
		assert.Equal(t, 50*time.Second, m.ThinkingDuration())
	})

	t.Run("returns zero when no started time", func(t *testing.T) {
		m := Message{
			Parts: []ContentPart{
				ReasoningContent{Thinking: "..."},
			},
		}
		assert.Equal(t, time.Duration(0), m.ThinkingDuration())
	})

	t.Run("returns zero for empty parts", func(t *testing.T) {
		m := Message{}
		assert.Equal(t, time.Duration(0), m.ThinkingDuration())
	})
}

func TestAttachment_IsImage(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"image/png", "image/png", true},
		{"image/jpeg", "image/jpeg", true},
		{"image/gif", "image/gif", true},
		{"text/plain", "text/plain", false},
		{"application/json", "application/json", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Attachment{MimeType: tt.mimeType}
			assert.Equal(t, tt.expected, a.IsImage())
		})
	}
}

func TestAttachment_IsText(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"text/plain", "text/plain", true},
		{"text/html", "text/html", true},
		{"text/x-shellscript", "text/x-shellscript", true},
		{"image/png", "image/png", false},
		{"application/json", "application/json", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Attachment{MimeType: tt.mimeType}
			assert.Equal(t, tt.expected, a.IsText())
		})
	}
}

func TestNewInfoMsg(t *testing.T) {
	msg := NewInfoMsg("hello")
	assert.Equal(t, InfoTypeInfo, msg.Type)
	assert.Equal(t, "hello", msg.Msg)
}

func TestNewWarnMsg(t *testing.T) {
	msg := NewWarnMsg("warning")
	assert.Equal(t, InfoTypeWarn, msg.Type)
	assert.Equal(t, "warning", msg.Msg)
}

func TestNewErrorMsg(t *testing.T) {
	err := assert.AnError
	msg := NewErrorMsg(err)
	assert.Equal(t, InfoTypeError, msg.Type)
	assert.Equal(t, err.Error(), msg.Msg)
}

func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("created"), EventCreated)
	assert.Equal(t, EventType("updated"), EventUpdated)
	assert.Equal(t, EventType("deleted"), EventDeleted)
}

func TestFinishReasons(t *testing.T) {
	assert.Equal(t, FinishReason("end_turn"), FinishReasonEndTurn)
	assert.Equal(t, FinishReason("max_tokens"), FinishReasonMaxTokens)
	assert.Equal(t, FinishReason("tool_use"), FinishReasonToolUse)
	assert.Equal(t, FinishReason("canceled"), FinishReasonCanceled)
	assert.Equal(t, FinishReason("error"), FinishReasonError)
	assert.Equal(t, FinishReason("permission_denied"), FinishReasonPermissionDenied)
}

func TestMessageRoles(t *testing.T) {
	assert.Equal(t, MessageRole("assistant"), RoleAssistant)
	assert.Equal(t, MessageRole("user"), RoleUser)
	assert.Equal(t, MessageRole("system"), RoleSystem)
	assert.Equal(t, MessageRole("tool"), RoleTool)
	assert.Equal(t, RoleUser, User)
	assert.Equal(t, RoleAssistant, Assistant)
	assert.Equal(t, RoleSystem, System)
	assert.Equal(t, RoleTool, Tool)
}

func TestScopes(t *testing.T) {
	assert.Equal(t, Scope("global"), ScopeGlobal)
	assert.Equal(t, Scope("project"), ScopeProject)
	assert.Equal(t, Scope("session"), ScopeSession)
}

func TestModelTypes(t *testing.T) {
	assert.Equal(t, ModelType("main"), ModelTypeMain)
	assert.Equal(t, ModelType("small"), ModelTypeSmall)
}
