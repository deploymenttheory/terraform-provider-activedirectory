# Provider registrations

Production registrations are explicit constructor slices in `internal/provider/resources.go`,
`datasources.go`, `list_resources.go` and `actions.go`. Add a constructor to the matching file
when its implementation is ready. Constructors must create fresh service instances and leave
directory connections to provider configuration.

The core provider owns schema and configuration. Its `Registry` supports feature tests without
importing the production wrapper or unrelated services. Each core instance copies the constructor
slices on creation and when returning registrations, so provider aliases and test factories cannot
change each other's registry entries.

Place a `provider_factories_test.go` beside each feature's tests. Use the shared core with only
the feature and any explicit fixture dependencies:

```go
package organizational_unit_test

import (
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-go/tfprotov6"

    providercore "github.com/deploymenttheory/terraform-provider-activedirectory/internal/provider/core"
    organizationalunit "github.com/deploymenttheory/terraform-provider-activedirectory/internal/services/resources/organizational_units/organizational_unit"
)

// Add this factory with the OU implementation; its service package is planned.
func testProviderFactory() (tfprotov6.ProviderServer, error) {
    p := providercore.NewWithRegistry("test", providercore.Registry{
        Resources: []func() resource.Resource{organizationalunit.NewResource},
    })
    return providerserver.NewProtocol6WithError(p)()
}
```

Data source and action tests register their constructors in the corresponding fields. List
resource tests must also register the matching managed resource because Framework validates
that discovery and resource type names agree. Register fixture dependencies explicitly.

The SDK client foundation (FND-03) owns authentication, client construction and publishing the
same configured bundle to resources, data sources, list resources and actions. Registry
construction performs no client initialization. No directory acceptance applies to registration
alone; feature acceptance must exercise the real Terraform CLI against the applicable lab targets.
