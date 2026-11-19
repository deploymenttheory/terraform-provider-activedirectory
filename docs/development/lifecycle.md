# Directory identity and lifecycle outcomes

Directory object resources use a canonical nonzero object GUID as `id`. Import accepts only
a GUID; `identity.ImportGUID` normalizes supported textual GUID forms. Reads, updates and deletes
address that GUID even after an external rename or move. Relationships use directed
`owner_guid/member_guid` identities and manage one edge; the Terraform resource type identifies
the relationship kind.

Construct SDK requests only after rejecting unknown inputs. `lifecycle.PlannedString` keeps
omitted input distinct from an explicit empty string. Null update semantics remain the feature's
responsibility: construct an explicit SDK attribute clear when clearing a previously managed
value, and omit properties that the feature does not own. Ordinary writes leave
`ExpectedVersion` unset, matching the Active Directory PowerShell module's ordinary operations.

`ObserveString` retains the SDK retrieval state. Requested-but-absent, unloaded and ranged
attributes are unavailable observations; they do not prove an unset value. Update state only
from usable observations, retaining the previous value otherwise. A required identity/property
that cannot be observed needs a feature diagnostic rather than a fabricated default.

Call `ObserveMutation` even when the SDK returns an error. Persist its known GUID and relevant
partial observations before returning diagnostics. The helper retains an accepted creation's
GUID after failed readback or a later workflow failure, and refuses conflicting identities.
For accepted creation without a GUID, retain the returned recovery DN in feature state and
resolve that existing object before attempting another create. A recovery DN is a locator,
never a substitute GUID or authorization to adopt an unrelated object.

An acknowledged write classified by the SDK as unverifiable may retain accepted intent with a
warning. It does not claim readback proof. Other accepted/uncertain/partial failures require
reconciliation; no helper retries a mutation or rolls one back. Raw SDK errors stay out of
Terraform diagnostics because peer text may contain credentials or other sensitive values.

Only `ConfirmedAbsent` authorizes removing an object after a pure SDK not-found result with
no returned candidates. It rejects joined denial, incomplete, canceled and unknown failures.
`ConfirmedDeletion` additionally requires the SDK's verified deletion and a complete positive
same-GUID `isDeleted` observation. An accepted delete reply or empty live search is insufficient.

`Readback` permits up to five read-only observations within at most thirty seconds. It retains
every partial result and stops on denied, transport or unknown errors. Readiness is supplied by
the feature and must check the intended GUID and properties. This foundation has no directory
I/O itself; each consuming service supplies its real lab acceptance coverage.
