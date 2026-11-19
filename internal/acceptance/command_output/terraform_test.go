// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package commandoutput_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
)

const outputBytes = 128 * 1024

// TestMain provides a synthetic CLI subprocess; it never loads providers or directory credentials.
func TestMain(m *testing.M) {
	if os.Getenv("TFEXEC_QUERY_OUTPUT_HELPER") != "1" {
		os.Exit(m.Run())
	}
	if len(os.Args) < 2 {
		os.Exit(3)
	}
	var output string
	exitCode := 0
	switch os.Args[1] {
	case "version":
		output = "{\"terraform_version\":\"1.17.2\",\"platform\":\"synthetic\"}\n"
	case "query":
		output = "{\"@level\":\"info\",\"@message\":\"synthetic query\",\"@module\":\"terraform.ui\",\"type\":\"version\",\"terraform\":\"1.17.2\",\"ui\":\"1.0\"}\n"
	case "plan":
		output = strings.Repeat(strings.Repeat("x", 128)+"\n", outputBytes/128)
		exitCode = 2
	default:
		os.Exit(3)
	}
	if _, err := io.WriteString(os.Stdout, output); err != nil {
		os.Exit(4)
	}
	os.Exit(exitCode)
}

func TestQueryPreservesSubsequentLargePlanOutput(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cli, err := tfexec.NewTerraform(t.TempDir(), executable)
	if err != nil {
		t.Fatal(err)
	}
	// The helper receives no inherited AD/cloud credentials or Terraform configuration.
	environment := map[string]string{"TFEXEC_QUERY_OUTPUT_HELPER": "1"}
	if root := os.Getenv("SystemRoot"); root != "" {
		environment["SystemRoot"] = root
	}
	if err := cli.SetEnv(environment); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	cli.SetStdout(&stdout)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	messages := 0
	query, err := cli.QueryJSON(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for message := range query {
		if message.Err != nil {
			t.Fatal(message.Err)
		}
		if message.Msg != nil {
			messages++
		}
	}
	if messages != 1 {
		t.Fatalf("query returned %d messages; expected one", messages)
	}
	changes, err := cli.Plan(ctx)
	if err != nil {
		t.Fatalf("large plan after query did not complete: %v", err)
	}
	if !changes || bytes.Count(stdout.Bytes(), []byte("x")) != outputBytes {
		t.Fatal("query replaced the caller's output destination or lost subsequent plan output")
	}
}
