// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

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
	server := providerserver.NewProtocol6(p)()
	return &protocol6Server{frameworkProtocol6Server: server.(frameworkProtocol6Server), provider: p}
}

// frameworkProtocol6Server retains all four Framework surfaces while the adapter overrides only shutdown.
// The pinned Framework implementation supplies these optional interfaces even when no services are registered.
type frameworkProtocol6Server interface {
	tfprotov6.ProviderServer
	tfprotov6.ListResourceServer
	tfprotov6.ActionServer
}

type protocol6Server struct {
	frameworkProtocol6Server
	provider ManagedProvider
}

// StopProvider cancels Framework operations before retiring the shared SDK session.
func (s *protocol6Server) StopProvider(ctx context.Context, req *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	response, err := s.frameworkProtocol6Server.StopProvider(ctx, req)
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
