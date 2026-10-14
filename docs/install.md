# Manual installation lifecycle

Archives contain application only; core is external. No installer connects,
changes host networking/shell files or installs boot persistence. No native package
manager/service installers are offered without install/uninstall gates.

Linux: verify checksums/available provenance, inspect archive paths/types, extract
into a new directory, install exact binary mode 0755 in an owned location. Use
permission-supporting storage. `start` refuses writable-by-others binaries. Preserve
notices/docs for offline recovery. Core must match the approved version/digest.

Update: stop, privately back up data, verify new archive, keep old executable, replace
only installed application. Never replace active core. Core updates need a new
tested adapter/manifest; other versions are refused. No automatic updater exists.
Schema migration happens on successful writes, not arbitrary read commands.

Uninstall: stop with installed binary, remove that exact executable. Profiles,
sources/backups remain. Purge only specifically inspected private application files;
never recursively remove a broad XDG/home root. Core uninstall is separate after
all sessions stop. There is no boot service to disable in this release.

Artifact tests install into clean private temporary storage, run version/help/init/
status/stop, replace the executable and uninstall while asserting unchanged data.
Extracted-binary integration additionally tests real proxy lifecycle. This is manual
archive evidence, not package-manager or native macOS/Windows certification. Actual
upgrade tests also use the 0.1 development baseline executable: migration, profile
preservation, old decoder refusal and real upgraded traffic. No public old release
is assumed.
