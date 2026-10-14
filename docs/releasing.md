# Release gates

This is a development deliverable, not a published distribution. Before release,
choose the application's license and review exact upstream notices: Cobra Apache-2.0,
sing-box GPL-3.0-or-later and bundled components. External process separation does
not waive redistribution responsibilities. Toolchains/cores are not vendored in
this repository. Inspect source/license/provenance and create an SBOM for actual
shipped artifacts, including optional native libraries when distributed.

Run unit, parser fuzz, real-core acceptance, race and native platform checks.
Establish the minimum Linux matrix, filesystem assumptions and resource measurements.
Sign/checksum application archives with reviewed provenance. Installation must not
connect or change networking, shell files or boot persistence. Never replace a running
core. Core downloads/updates remain explicit and independently versioned.

No release can advertise TUN, full tunnel, DNS leak protection, kill switch, universal
OS compatibility or benchmark targets until the corresponding specification gates
have actual evidence.
