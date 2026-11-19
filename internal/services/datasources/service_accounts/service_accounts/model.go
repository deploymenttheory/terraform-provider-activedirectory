// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package service_accounts

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/service_accounts"

// ServiceAccountsDataSourceModel contains explicit query inputs; result objects use the shared service account projection.
type ServiceAccountsDataSourceModel = common.SearchModel
