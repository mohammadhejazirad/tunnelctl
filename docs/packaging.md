# Packaging contract and toolchain

Command/package name is `tunnelctl`, matching the module and repository. There is
no separate registered final product name. No core, private data, driver or service
is bundled. sing-box 1.14.3 must be installed separately and match the approved
binary digest; currently only Linux amd64 has that approval. Xray/TUN are not
implemented. Other targets are experimental installation/metadata builds, not
functional VPN support.

## Tools and official references

- Go 1.27.2 from go.mod; release CGO=0, separate race CGO=1.
- [GoReleaser OSS 2.18.3](https://github.com/goreleaser/goreleaser/tree/v2.18.3),
  embedded [nFPM 2.47.0](https://github.com/goreleaser/nfpm/tree/v2.47.0).
  Exact-tag docs: `www/content/customization/package/{archives,nfpm}.md` and
  `www/content/customization/publish/snapshots.md`. GoReleaser uses its snapshot
  engine **only to package with a mapped native version**, never to publish.
  Our independent clean-tree/exact-tag gate controls official builds; ldflags and
  filenames retain actual SemVer. `release.disable` is always true.
- [appimagetool 1.9.1](https://github.com/AppImage/appimagetool/tree/1.9.1),
  `--runtime-file` supplies separately SHA-256 pinned type2 runtimes from upstream
  commit `8f39b89e2ac31e1640b3d3f7e9a5108e6ce805fa`. Upstream offers a moving
  `continuous` URL: changed bytes fail closed; review/update pins before retrying,
  never accept a new digest automatically. Build extraction avoids needing FUSE.
  Each image includes full static-library notices, the exact runtime source/build
  scripts/patch, complete libfuse 3.15.0 source and LGPL modification/relinking
  instructions under `usr/share/doc/tunnelctl/APPIMAGE_RUNTIME`. Runtime source
  is MIT; libfuse is LGPL 2.1, separate from the application's Apache-2.0.
- [WiX 5.0.2](https://github.com/wixtoolset/wix/releases/tag/v5.0.2), schema v4,
  Windows .NET 8 SDK. Environment table appends/removes its own machine PATH
  entry; no script overwrites PATH. MajorUpgrade family UpgradeCode is permanent;
  product/package codes change with rebuilds. Public assets must never be rebuilt
  under the same tag. WiX tool is installed from NuGet at the exact version.
  [Environment](https://docs.firegiant.com/wix/schema/wxs/environment/),
  [MajorUpgrade](https://docs.firegiant.com/wix/schema/wxs/majorupgrade/).
  The inspected NuGet runtimeconfig targets net6.0 with rollForward=Major, allowing
  the runner's supported .NET 8+ runtime without installing EOL .NET 6. Existing
  unmanaged machine PATH entries referencing the fixed destination are refused
  rather than claimed and later removed by MSI.
- Native macOS `pkgbuild`, `productbuild`, `pkgutil` from the pinned runner OS;
  CI records versions. Xcode tools are OS components, not downloaded scripts.

## Version conversion

All native managers use `MAJOR.MINOR.(PATCH*1000+rank)` with package release 1
where required. alpha.N rank=N; beta.N=200+N; rc.N=400+N; final=999.
N must be 1..199; major/minor <=255, patch <=64. Other valid SemVer prerelease
forms are rejected for packaging, with no partial release. This deliberately
restricted mapping satisfies MSI's three fields (255/255/65535 limits) and orders
alpha < beta < rc < final < next-patch-alpha identically in deb/rpm/pacman/apk/pkg.
Example: binary `0.2.0-rc.1` -> native version `0.2.401`; final -> `0.2.999`.
Do not compare native version text to binary SemVer as if they were identical.
Do not silently switch this mapping after publication: use a reviewed migration.
Application, core, storage schema and installer revision are independent.

## Assets and boundaries

For VERSION without `v`, filenames are `tunnelctl_VERSION_OS_ARCH.EXT`.
Linux amd64/arm64: tar.gz, deb, rpm, pkg.tar.zst, apk, AppImage and PKGBUILD.
Darwin amd64/arm64: tar.gz and pkg. Windows amd64: zip and msi.
There is no universal macOS binary or Windows arm64 artifact.
Native packages install `/usr/bin/tunnelctl`; macOS owns
`/Library/Application Support/tunnelctl` plus a checked symlink `/usr/local/bin/tunnelctl`;
MSI owns Program Files/tunnelctl and its machine PATH entry. Unix packages and
MSI refuse unmanaged destination collisions. Alpine's apk continues unpacking
after a failed preinstall script: its dedicated hook instead moves an unmanaged
destination intact into a root-private `/var/backups/tunnelctl-unmanaged.XXXXXXXX`
directory and prints an explicit warning. Removal preserves that backup; review
it before explicitly restoring/migrating it. No shell startup files are edited.
macOS checks that payload ancestors are root-owned and not writable by other
users. It refuses unsafe ancestors instead of changing their permissions or
ownership. This avoids placing a root-invoked uninstaller under a potentially
Homebrew-writable `/usr/local/lib` tree.

Public staging uses only named documents/examples and linked-module license
texts; every other source path is excluded. The generated GoReleaser configuration
lists each staged file individually with explicit 0755 directory/0644 document
modes, avoiding recursive-glob flattening and source-filesystem mode inheritance.
Docker context remains deny-all;
package test containers receive only the staged asset directory read-only.
`links_and_subs.txt` is never read by packaging or public tests. The aggregate
gate rejects unexpected files and requires all 23 named payload/metadata assets
before writing checksums. SHA-256 does not substitute for signatures/provenance.

The generated PKGBUILD references the exact release tarball and its SHA-256;
`makepkg` works with a locally cached tarball for dry-runs. It does not publish
anything to AUR. Package-manager destination collision checks and ordinary file
ownership prevent silently replacing another install. No deb/rpm/apk signature
is claimed. Unsigned apk installation requires explicit `--allow-untrusted` for
this local file only; it does not configure an untrusted repository.

## Local build

The orchestration/bootstrap host is Linux amd64, matching the CI build job.
The generated binaries target five OS/architecture combinations; MSI/pkg are
added on native Windows/macOS runners rather than emulated on that host.

```sh
bash packaging/tools.sh /tmp/tunnelctl-package-tools
export PATH="/tmp/tunnelctl-package-tools:$PATH"
go run ./cmd/packaging --tag v0.2.0-rc.1 --dry-run --out dist/packages
bash packaging/appimage/build.sh dist/packages /tmp/tunnelctl-package-tools 0.2.0-rc.1
# MSI/pkg are built/tested natively by packaging.yml, not emulated locally.
```

`dist/packages/assets` is not a complete release until native jobs succeed.
`--mode finalize` requires the whole set; `--mode verify-assets` rechecks it.
The legacy `cmd/release` remains for archive-only regression diagnostics.
Transient `--appimage-extract-and-run` cannot forward directed signals with the
pinned upstream runtime; shared application services refuse sessions in that mode.
Use stable `--appimage-extract` plus direct AppRun for the tested no-FUSE fallback.
Mounted mode execs AppRun directly in upstream source, but needs native FUSE
runtime evidence before claiming it was tested in this environment.
See [installation](install.md), [release workflow](releasing.md), and the
[execution checklist](packaging-checklist.md) for evidence and limitations.
