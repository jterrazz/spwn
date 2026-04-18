package compile

import "spwn.sh/packages/compile/internal/dockerfile"

// GenerateOpts configures Dockerfile generation.
type GenerateOpts = dockerfile.GenerateOpts

// ToolInput is the data needed from each tool to generate a Dockerfile.
type ToolInput = dockerfile.ToolInput

// GenerateDockerfile composes a final Dockerfile from a base Dockerfile and tool inputs.
// Tools must be in topological order (dependencies first).
func GenerateDockerfile(baseDockerfile []byte, tools []ToolInput, imageVersion string, opts ...GenerateOpts) []byte {
	return dockerfile.Generate(baseDockerfile, tools, imageVersion, opts...)
}

// ToolsToInputs converts resolved Tool interfaces to ToolInput structs
// for use with GenerateDockerfile.
func ToolsToInputs(tools []Tool) []ToolInput {
	inputs := make([]ToolInput, len(tools))
	for i, t := range tools {
		spec := t.Install()
		inputs[i] = ToolInput{
			Name:         t.Name(),
			Kind:         string(t.Kind()),
			AptPackages:  spec.AptPackages,
			Commands:     spec.Commands,
			UserCommands: spec.UserCommands,
			Env:          spec.Env,
			Files:        spec.Files,
			HasSkills:    t.Skills() != nil,
		}
	}
	return inputs
}
