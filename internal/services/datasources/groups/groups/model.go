// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package groups

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/groups"

// GroupsDataSourceModel contains explicit query inputs; result objects use the shared group projection.
type GroupsDataSourceModel = common.SearchModel
