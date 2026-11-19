// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package acceptance_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/deploymenttheory/terraform-provider-activedirectory/internal/acceptance"
)

func labEnvironment() map[string]string {
	return map[string]string{
		"TF_ACC": "1", "TF_ACC_AD_LAB": "1", "AD_ACCEPTANCE_AUTHORIZED": "1",
		"AD_HOST": "adsdk25dc1.ad.sdk.test", "AD_ACCEPTANCE_HOSTS": "adsdk25dc1.ad.sdk.test",
		"AD_ACCEPTANCE_RUN_ID": "tfad-test-123", "AD_ACCEPTANCE_PARTITION": "DC=ad,DC=sdk,DC=test", "AD_ACCEPTANCE_SERVER_VERSION": "2025",
	}
}

func TestLabGuardRequiresEveryAuthorizationAndScopeField(t *testing.T) {
	t.Parallel()
	for name := range labEnvironment() {
		t.Run(name, func(t *testing.T) {
			env := labEnvironment()
			delete(env, name)
			_, err := acceptance.ReadTarget(func(key string) string { return env[key] }, "linux/amd64")
			if !errors.Is(err, acceptance.ErrUnsafeTarget) {
				t.Fatalf("missing %s was admitted", name)
			}
		})
	}
	for _, platform := range []string{"linux/amd64", "windows/amd64"} {
		env := labEnvironment()
		target, err := acceptance.ReadTarget(func(key string) string { return env[key] }, platform)
		if err != nil || target.Host != env["AD_HOST"] {
			t.Fatalf("explicit lab refused: %v", err)
		}
	}
}

func TestLabGuardRejectsMismatchesAndUntrustedRunners(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, value string }{
		{"AD_HOST", "outside.example.test"},
		{"AD_HOST", "ldaps://private@secret.example.test"},
		{"AD_ACCEPTANCE_RUN_ID", "../../escape"},
		{"AD_ACCEPTANCE_SERVER_VERSION", "2012"},
		{"GITHUB_EVENT_NAME", "pull_request"},
		{"GITHUB_EVENT_NAME", "pull_request_target"},
	}
	for _, test := range cases {
		env := labEnvironment()
		env[test.name] = test.value
		_, err := acceptance.ReadTarget(func(key string) string { return env[key] }, "linux/amd64")
		if !errors.Is(err, acceptance.ErrUnsafeTarget) || strings.Contains(err.Error(), test.value) {
			t.Fatal("unsafe metadata admitted or reflected into errors")
		}
	}
	for _, platform := range []string{"darwin/amd64", "linux/arm64"} {
		env := labEnvironment()
		if _, err := acceptance.ReadTarget(func(key string) string { return env[key] }, platform); err == nil {
			t.Fatal("unqualified live runtime admitted")
		}
	}
}

func TestFixtureNamesStayWithinRunBoundary(t *testing.T) {
	t.Parallel()
	first, err := acceptance.FixtureName("tfad-one-123", "TestAccOU")
	if err != nil {
		t.Fatal(err)
	}
	same, _ := acceptance.FixtureName("tfad-one-123", "TestAccOU")
	other, _ := acceptance.FixtureName("tfad-two-123", "TestAccOU")
	if first != same || first == other || len(first) > 64 || strings.ContainsAny(first, "/=,\\") {
		t.Fatal("fixture names are not safe and run-scoped")
	}
	if _, err := acceptance.FixtureName("../escape", "TestAccOU"); err == nil {
		t.Fatal("unsafe run identity admitted")
	}
}

type reporter struct {
	callbacks []func()
	failures  []string
}

func (*reporter) Helper()            {}
func (r *reporter) Cleanup(f func()) { r.callbacks = append(r.callbacks, f) }
func (r *reporter) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func TestCleanupErrorsAndPartialObservationsRemainFailures(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		verified    bool
		err         error
		wantFailure bool
	}{
		{false, nil, true}, {true, errors.New("private peer detail"), true}, {true, nil, false},
	} {
		r := &reporter{}
		acceptance.RegisterCleanup(r, "OU fixture", func(context.Context) (bool, error) { return test.verified, test.err })
		if len(r.callbacks) != 1 {
			t.Fatal("cleanup was not registered")
		}
		r.callbacks[0]()
		if (len(r.failures) != 0) != test.wantFailure {
			t.Fatal("cleanup failure was lost")
		}
		if strings.Contains(fmt.Sprint(r.failures), "private peer detail") {
			t.Fatal("cleanup diagnostics leaked peer text")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := acceptance.VerifyCleanup(ctx, func(context.Context) (bool, error) { return true, nil }); err == nil {
		t.Fatal("canceled verification was accepted as complete")
	}
}
