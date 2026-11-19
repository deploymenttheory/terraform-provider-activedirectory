// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package acceptance

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var (
	ErrUnsafeTarget = errors.New("explicit lab authorization, target and fixture identity are required")
	runIDPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{5,39}$`)
)

// Target identifies the selected lab endpoint and fixture boundary for one acceptance invocation.
type Target struct {
	Host          string
	Partition     string
	RunID         string
	ServerVersion string
}

// ReadTarget validates explicit authorization and target metadata without contacting the directory.
// Error text never includes environment values, credentials or peer diagnostics.
func ReadTarget(lookup func(string) string, platform string) (Target, error) {
	if lookup("TF_ACC") != "1" || lookup("TF_ACC_AD_LAB") != "1" || lookup("AD_ACCEPTANCE_AUTHORIZED") != "1" {
		return Target{}, ErrUnsafeTarget
	}
	if platform != "linux/amd64" && platform != "windows/amd64" {
		return Target{}, ErrUnsafeTarget
	}
	if lookup("GITHUB_EVENT_NAME") == "pull_request" || lookup("GITHUB_EVENT_NAME") == "pull_request_target" {
		return Target{}, ErrUnsafeTarget
	}
	target := Target{
		Host:          lookup("AD_HOST"),
		Partition:     lookup("AD_ACCEPTANCE_PARTITION"),
		RunID:         lookup("AD_ACCEPTANCE_RUN_ID"),
		ServerVersion: lookup("AD_ACCEPTANCE_SERVER_VERSION"),
	}
	if target.Host == "" || target.Partition == "" || !runIDPattern.MatchString(target.RunID) {
		return Target{}, ErrUnsafeTarget
	}
	if strings.ContainsAny(target.Host, "/\\:@ \t\r\n") {
		return Target{}, ErrUnsafeTarget
	}
	switch target.ServerVersion {
	case "2016", "2019", "2022", "2025":
	default:
		return Target{}, ErrUnsafeTarget
	}
	for _, allowed := range strings.Split(lookup("AD_ACCEPTANCE_HOSTS"), ",") {
		if strings.EqualFold(target.Host, strings.TrimSpace(allowed)) {
			return target, nil
		}
	}
	return Target{}, ErrUnsafeTarget
}

// PreCheck refuses live acceptance unless the caller explicitly selects an authorized lab and fixture run.
func PreCheck(t *testing.T) Target {
	t.Helper()
	target, err := ReadTarget(os.Getenv, runtime.GOOS+"/"+runtime.GOARCH)
	if err != nil {
		t.Fatal("live acceptance refused: explicit lab authorization, allowlisted host, partition, server version and run ID are required")
	}
	return target
}

// FixtureName produces a safe, deterministic name scoped to the externally unique run identity.
func FixtureName(runID, testName string) (string, error) {
	if !runIDPattern.MatchString(runID) || testName == "" {
		return "", ErrUnsafeTarget
	}
	digest := sha256.Sum256([]byte(runID + "\x00" + testName))
	return fmt.Sprintf("tfad-%s-%x", runID, digest[:6]), nil
}
