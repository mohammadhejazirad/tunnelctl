# 0004: GoReleaser and native installer boundaries

The primary release path now uses GoReleaser OSS 2.18.3 and its embedded nFPM
2.47.0 instead of the original archive-only Go builder. This implements the
requested package-manager lifecycle while retaining the original public-file
allowlist, module-license inventory and archive acceptance tests. The old builder
remains an offline diagnostic path, not the GitHub publication implementation.

Release binaries use CGO=0. Race builds remain separate CGO=1 builds. No core,
privileged helper, service, driver or private user data is bundled. Installer tests
do not certify unsupported networking backends. macOS/Windows and Linux arm64
installers are explicitly experimental until their runtime adapters are approved.

GoReleaser builds without publishing; native jobs add MSI/pkg/AppImage; a separate
least-privilege job verifies the complete allowlisted set, creates a draft, verifies
uploaded hashes and only then promotes it. Manual runs cannot reach publication.
No APT/DNF/AUR/Homebrew/WinGet repository is created.

macOS stores its system payload in `/Library/Application Support/tunnelctl`,
not `/usr/local/lib`, whose ancestors may be writable by a Homebrew user. The
installer checks root ownership and non-writable ancestor modes before installing
the binary and administrator-invoked uninstaller. Only its managed command
symlink lives in `/usr/local/bin`; no shell profile is modified.
