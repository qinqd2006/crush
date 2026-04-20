package adapter

import (
	"charm.land/catwalk/pkg/catwalk"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qinqd2006/crush/infra/pkg/commands"
	"github.com/qinqd2006/crush/kernel/pkg"
	"qinqd2006github.com/qinqd2006/crush/crush-agent/pkg/config"
)

// CatwalkProviderToKernel converts a catwalk.Provider to a kernel.Provider.
func CatwalkProviderToKernel(p catwalk.Provider) kernel.Provider {
	return kernel.Provider{
		ID:   string(p.ID),
		Name: p.Name,
		Type: string(p.Type),
	}
}

// ConfigSelectedModelToKernel converts a config.SelectedModel to a kernel.SelectedModel.
func ConfigSelectedModelToKernel(s config.SelectedModel) kernel.SelectedModel {
	return kernel.SelectedModel{
		Model:            s.Model,
		Provider:         s.Provider,
		ReasoningEffort:  s.ReasoningEffort,
		Think:            s.Think,
		MaxTokens:        s.MaxTokens,
		Temperature:      s.Temperature,
		TopP:             s.TopP,
		TopK:             s.TopK,
		FrequencyPenalty: s.FrequencyPenalty,
		PresencePenalty:  s.PresencePenalty,
		ProviderOptions:  s.ProviderOptions,
	}
}

// kernelToCatwalkProvider converts a kernel.Provider to a catwalk.Provider.
func kernelToCatwalkProvider(k kernel.Provider) catwalk.Provider {
	return catwalk.Provider{
		ID:          catwalk.InferenceProvider(k.ID),
		Name:        k.Name,
		Type:        catwalk.Type(k.Type),
		APIEndpoint: k.APIEndpoint,
	}
}

// kernelSelectedModelToConfig converts a kernel.SelectedModel to a config.SelectedModel.
func kernelSelectedModelToConfig(k kernel.SelectedModel) config.SelectedModel {
	return config.SelectedModel{
		Model:            k.Model,
		Provider:         k.Provider,
		ReasoningEffort:  k.ReasoningEffort,
		Think:            k.Think,
		MaxTokens:        k.MaxTokens,
		Temperature:      k.Temperature,
		TopP:             k.TopP,
		TopK:             k.TopK,
		FrequencyPenalty: k.FrequencyPenalty,
		PresencePenalty:  k.PresencePenalty,
		ProviderOptions:  k.ProviderOptions,
	}
}

// CommandsToKernel converts engine commands types to kernel types.
func CommandsToKernel(cc []commands.CustomCommand, mp []commands.MCPPrompt) ([]kernel.CustomCommand, []kernel.MCPPrompt) {
	kernelCC := make([]kernel.CustomCommand, len(cc))
	for i, c := range cc {
		kernelCC[i] = kernel.CustomCommand{
			ID:        c.ID,
			Name:      c.Name,
			Content:   c.Content,
			Arguments: ArgumentsToKernel(c.Arguments),
		}
	}

	kernelMP := make([]kernel.MCPPrompt, len(mp))
	for i, p := range mp {
		kernelMP[i] = kernel.MCPPrompt{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			PromptID:    p.PromptID,
			ClientID:    p.ClientID,
			Arguments:   ArgumentsToKernel(p.Arguments),
		}
	}

	return kernelCC, kernelMP
}

// ArgumentsToKernel converts engine Argument to kernel.Argument.
func ArgumentsToKernel(args []commands.Argument) []kernel.Argument {
	if args == nil {
		return nil
	}
	kernelArgs := make([]kernel.Argument, len(args))
	for i, a := range args {
		kernelArgs[i] = kernel.Argument{
			ID:          a.ID,
			Title:       a.Title,
			Description: a.Description,
			Required:    a.Required,
		}
	}
	return kernelArgs
}

// MCPToolToKernel converts a go-sdk mcp.Tool to a kernel.MCPTool.
func MCPToolToKernel(t *mcp.Tool) kernel.MCPTool {
	if t == nil {
		return kernel.MCPTool{}
	}
	return kernel.MCPTool{
		Name:        t.Name,
		Title:       t.Title,
		Description: t.Description,
		InputSchema: t.InputSchema,
	}
}

// MCPToolsToKernel converts a slice of go-sdk mcp.Tool to kernel.MCPTool.
func MCPToolsToKernel(tools []*mcp.Tool) []kernel.MCPTool {
	if tools == nil {
		return nil
	}
	result := make([]kernel.MCPTool, len(tools))
	for i, t := range tools {
		result[i] = MCPToolToKernel(t)
	}
	return result
}

// MCPPromptToKernel converts a go-sdk mcp.Prompt to a kernel.MCPPrompt.
func MCPPromptToKernel(p *mcp.Prompt) kernel.MCPPrompt {
	if p == nil {
		return kernel.MCPPrompt{}
	}
	kernelArgs := make([]kernel.Argument, len(p.Arguments))
	for i, a := range p.Arguments {
		kernelArgs[i] = kernel.Argument{
			ID:          a.Name,
			Title:       a.Title,
			Description: a.Description,
			Required:    a.Required,
		}
	}
	return kernel.MCPPrompt{
		ID:          p.Name,
		Title:       p.Title,
		Description: p.Description,
		Arguments:   kernelArgs,
	}
}

// MCPPromptsToKernel converts a slice of go-sdk mcp.Prompt to kernel.MCPPrompt.
func MCPPromptsToKernel(prompts []*mcp.Prompt) []kernel.MCPPrompt {
	if prompts == nil {
		return nil
	}
	result := make([]kernel.MCPPrompt, len(prompts))
	for i, p := range prompts {
		result[i] = MCPPromptToKernel(p)
	}
	return result
}

// MCPResourceToKernel converts a go-sdk mcp.Resource to a kernel.MCPResource.
func MCPResourceToKernel(r *mcp.Resource) kernel.MCPResource {
	if r == nil {
		return kernel.MCPResource{}
	}
	return kernel.MCPResource{
		Name:        r.Name,
		Title:       r.Title,
		Description: r.Description,
		MIMEType:    r.MIMEType,
	}
}

// MCPResourcesToKernel converts a slice of go-sdk mcp.Resource to kernel.MCPResource.
func MCPResourcesToKernel(resources []*mcp.Resource) []kernel.MCPResource {
	if resources == nil {
		return nil
	}
	result := make([]kernel.MCPResource, len(resources))
	for i, r := range resources {
		result[i] = MCPResourceToKernel(r)
	}
	return result
}
