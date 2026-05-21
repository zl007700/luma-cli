package commands

import "github.com/luma-cli/lumer-cli/project"

func recordProjectArtifact(artifactType, path, step string) {
	proj, _ := project.GetActiveProject()
	if proj == nil {
		return
	}
	_ = proj.AddArtifact(project.Artifact{
		Type:    artifactType,
		Path:    path,
		Step:    step,
		Ability: step,
		Outputs: []string{path},
		Status:  "completed",
	})
}
