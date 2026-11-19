// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package users

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/users"

// UsersDataSourceModel contains explicit query inputs; result objects use the shared user projection.
type UsersDataSourceModel = common.SearchModel
