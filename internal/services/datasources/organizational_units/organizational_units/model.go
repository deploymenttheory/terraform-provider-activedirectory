// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package organizationalUnits

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/organizational_units"

// OrganizationalUnitsDataSourceModel contains explicit query inputs; result objects use the shared OU projection.
type OrganizationalUnitsDataSourceModel = common.SearchModel
