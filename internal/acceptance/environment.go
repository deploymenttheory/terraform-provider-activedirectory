// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package acceptance

import (
	"os"
	"strings"
	"testing"
)

// IsolateUnitEnvironment prevents Terraform-running unit tests from consuming live AD configuration.
// Tests using this helper must remain sequential because environment variables are process-wide.
func IsolateUnitEnvironment(t *testing.T) {
	t.Helper()
	for _, pair := range os.Environ() {
		name, _, _ := strings.Cut(pair, "=")
		if strings.HasPrefix(name, "AD_") || name == "TF_ACC_AD_LAB" || name == "TF_ACC" {
			t.Setenv(name, "")
		}
	}
	t.Setenv("TF_ACC", "")
	t.Setenv("TF_ACC_AD_LAB", "")
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "deploymenttheory")
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		t.Setenv("TF_ACC_TERRAFORM_VERSION", "1.17.2")
	}
}
