// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package objects

import common "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/common/objects"

// ObjectsDataSourceModel contains explicit query inputs; result objects use the shared object projection.
type ObjectsDataSourceModel = common.SearchModel
