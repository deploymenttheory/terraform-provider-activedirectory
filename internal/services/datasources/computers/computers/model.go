// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package computers

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/computers"

// ComputersDataSourceModel contains explicit query inputs; result objects use the shared computer projection.
type ComputersDataSourceModel = common.SearchModel
