// Copyright (c) Deployment Theory.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// ManagedProvider owns clients that must be closed when Terraform stops its provider server.
type ManagedProvider interface {
	provider.Provider
	Close() error
}

// NewProtocol6Server serves the Framework provider with explicit client cleanup on StopProvider.
// Feature test factories use this adapter so in-process acceptance sessions are closed too.
func NewProtocol6Server(p ManagedProvider) tfprotov6.ProviderServer {
	return &protocol6Server{ProviderServer: providerserver.NewProtocol6(p)(), provider: p}
}

type protocol6Server struct {
	tfprotov6.ProviderServer
	provider ManagedProvider
}

// StopProvider cancels Framework operations before retiring the shared SDK session.
func (s *protocol6Server) StopProvider(ctx context.Context, req *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	response, err := s.ProviderServer.StopProvider(ctx, req)
	if closeErr := s.provider.Close(); closeErr != nil {
		if response == nil {
			response = &tfprotov6.StopProviderResponse{}
		}
		response.Error = "The directory session could not be closed safely."
	}
	if err != nil {
		return response, fmt.Errorf("stop provider protocol server: %w", err)
	}
	return response, nil
}
