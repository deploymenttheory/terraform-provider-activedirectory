// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package group_member

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

type labSecretPlan struct{ secrets []string }

// CheckPlan checks the entire Terraform JSON plan for bind and fixture passwords, including ephemeral reader input.
func (check labSecretPlan) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	encoded, err := json.Marshal(req.Plan)
	if err != nil {
		resp.Error = errLabOwnership
		return
	}
	for _, secret := range check.secrets {
		if secret != "" && strings.Contains(string(encoded), secret) {
			resp.Error = errLabOwnership
			return
		}
	}
}
