# Provider checks and builds

Pull requests run native unit/protocol tests and vet on Linux, Windows and macOS. Those jobs
do not qualify live AD access. The offline Terraform foundation check exercises the real CLI;
directory acceptance remains an explicit authorized lab invocation from Windows/Linux amd64
against each applicable Server 2016/2019/2022/2025 profile.

The existing required lint check names are retained. Both lint workflows fetch `origin/main`,
use the local Git merge base, run without fixes and fail on findings. They do not request the
GitHub PR file-diff API or write comments. Formatting checks read Go sources and Terraform
examples without modifying them. The documentation workflow owns authored-input coverage,
four-surface generation and generated drift; see [documentation](documentation.md).

Every supported Linux/Windows/macOS amd64/arm64 target is cross-built with CGO disabled,
trimmed paths, fixed version/commit metadata and no build ID or VCS metadata. The build script
builds twice into different output directories and compares bytes before recording SHA256.
CI retains each verified binary and checksum as a target-specific artifact. Cross-build success
does not prove architecture-native runtime or lab acceptance.

```shell
bash scripts/build-provider.sh linux amd64 dev "$(git rev-parse HEAD)"
```

Release Please uses its Go release strategy for provider versioning rather than the inherited
Terraform-module strategy. These CI artifacts are development builds. Registry signing,
release archives, publication and final qualification remain the release implementation's
responsibility; building or merging a foundation PR does not establish that release gate.
