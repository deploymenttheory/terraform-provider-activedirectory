# Provider language

**Execution runtime**: The operating system and architecture running Terraform and its provider process. _Avoid_: target server.

**Directory target**: The selected Windows Server AD DS endpoint or directory scope being managed. _Avoid_: provider runtime.

**Managed object**: A directory object whose supported lifecycle and owned attributes Terraform reconciles.

**Adopted settings**: Existing directory settings Terraform takes responsibility for without assuming infrastructure provisioning or deletion ownership.

**Relationship**: One directly owned association between identified directory objects. _Avoid_: effective membership.

**Observation**: Returned directory information together with its scope, completeness and availability limits.

**Qualification evidence**: Retained observations of tests with identified implementation, client runtime, target environment and independent cleanup results. A planned test is not evidence.

**Reconciliation**: Establishing observable directory state and what remains before choosing another change. _Avoid_: blind retry.

**List resource**: Import-oriented discovery of instances of a corresponding managed resource type. _Avoid_: plural data source.

**Action**: An explicitly invoked operation whose intent does not fit ownership through a normal managed lifecycle.

**Delivery gate**: The evidence required before a work item is complete; writing code or raising a PR alone is insufficient.
