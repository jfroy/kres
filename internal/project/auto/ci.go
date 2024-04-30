// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"fmt"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/project/common"
)

// DetectCI checks the ci settings.
func (builder *builder) DetectCI() (bool, error) {
	var ci CI

	if err := builder.meta.Config.Load(&ci); err != nil {
		return false, err
	}

	if ci.Provider == "" {
		ci.Provider = config.CIProviderGitHubActions
	}

	switch ci.Provider {
	case config.CIProviderDrone:
	case config.CIProviderGitHubActions:
	default:
		return false, fmt.Errorf("unknown ci provider: %s", ci.Provider)
	}

	builder.meta.CIProvider = ci.Provider
	builder.meta.CompileGithubWorkflowsOnly = ci.CompileGHWorkflowsOnly

	return true, nil
}

// BuildCI builds the ci settings.
func (builder *builder) BuildCI() error {
	if builder.meta.CompileGithubWorkflowsOnly {
		repository := common.NewRepository(builder.meta)
		sops := common.NewSOPS(builder.meta)
		ghworkflow := common.NewGHWorkflow(builder.meta)

		builder.proj.AddTarget(repository, sops, ghworkflow)
	}

	return nil
}
