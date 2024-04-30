// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Release provides common release target.
type Release struct {
	meta *meta.Options
	dag.BaseNode

	// List of file patterns relative to the ArtifactsPath to include in the release.
	//
	// If not specified, defaults to the auto-detected commands.
	Artifacts []string `yaml:"artifacts"`
}

// NewRelease initializes Release.
func NewRelease(m *meta.Options) *Release {
	return &Release{
		BaseNode: dag.NewBaseNode("release"),

		meta: m,

		Artifacts: xslices.Map(m.Commands, func(cmd meta.Command) string {
			return cmd.Name + "-*"
		}),
	}
}

// CompileDrone implements drone.Compiler.
func (release *Release) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep("release-notes").
		OnlyOnTag().
		DependsOn(dag.GatherMatchingInputNames(release, drone.HasDroneOutput())...),
	)

	output.Step(drone.CustomStep(release.Name()).
		Image("plugins/github-release").
		PublishArtifacts(
			filepath.Join(release.meta.ArtifactsPath, "RELEASE_NOTES.md"),
			xslices.Map(release.Artifacts, func(artifact string) string {
				return filepath.Join(release.meta.ArtifactsPath, artifact)
			})...,
		).
		OnlyOnTag().
		DependsOn("release-notes"),
	)

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (release *Release) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	steps := []*ghworkflow.JobStep{}

	releaseStep := ghworkflow.Step("Release").
		SetUses("crazy-max/ghaction-github-release@"+config.ReleaseActionVersion).
		SetWith("body_path", filepath.Join(release.meta.ArtifactsPath, "RELEASE_NOTES.md")).
		SetWith("draft", "true")

	if len(release.meta.Commands) > 0 {
		artifacts := xslices.Map(release.Artifacts, func(artifact string) string {
			return filepath.Join(release.meta.ArtifactsPath, artifact)
		})

		checkSumCommands := []string{
			fmt.Sprintf("sha256sum %s > %s", strings.Join(artifacts, " "), filepath.Join(release.meta.ArtifactsPath, "sha256sum.txt")),
			fmt.Sprintf("sha512sum %s > %s", strings.Join(artifacts, " "), filepath.Join(release.meta.ArtifactsPath, "sha512sum.txt")),
		}

		checkSumStep := ghworkflow.Step("Generate Checksums").
			SetCommand(strings.Join(checkSumCommands, "\n"))

		releaseStep.SetWith("files", strings.Join(artifacts, "\n")+"\n"+filepath.Join(release.meta.ArtifactsPath, "sha*.txt"))

		if err := checkSumStep.SetConditions("only-on-tag"); err != nil {
			return err
		}

		steps = append(steps, checkSumStep)
	}

	if err := releaseStep.SetConditions("only-on-tag"); err != nil {
		return err
	}

	releaseNotesStep := ghworkflow.Step("release-notes").
		SetMakeStep("release-notes")

	if err := releaseNotesStep.SetConditions("only-on-tag"); err != nil {
		return err
	}

	steps = append(
		steps,
		releaseNotesStep,
		releaseStep,
	)

	output.AddStep(
		"default",
		steps...,
	)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (release *Release) CompileMakefile(output *makefile.Output) error {
	output.Target("release-notes").
		Depends("$(ARTIFACTS)").
		Script("@ARTIFACTS=$(ARTIFACTS) ./hack/release.sh $@ $(ARTIFACTS)/RELEASE_NOTES.md $(TAG)").
		Phony()

	return nil
}
