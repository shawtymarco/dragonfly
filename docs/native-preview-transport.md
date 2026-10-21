# Native 1.26.60.30 Preview and transport integration

This isolated change permits the dual-transport proxy fixture to compile against
the native 1.26.60 candidate. Comparing gophertunnel `9e7f28092180` with
`8fe51a21de6d` shows that `protocol.Skin.PlayFabID` and its wire string were removed.
The two Dragonfly session conversion references are removed accordingly; other
appearance metadata and the input skin remain unchanged. Historical skin wire
differences remain owned by go-multiversion, not Dragonfly.

The `feat/all_26_u6` branch now pins upstream gophertunnel `28a5eec10daf` / protocol
2225 without a replacement. The matching transport fork is selected by consumers.
Exact .30 blocks, items, creative inventory and recipes come from go-multiversion's
verified `data/native2225` bundle. `server/world/native_snapshot.json` records source
and installed hashes; `tools/export_native2225` in that repository reproduces them.

The extracted block ordering is local, not vanilla numeric network ordering.
`server.NativeProtocol` converts the finalized registry's IDs to canonical network
hashes, including chunks before caching and item/event/actor block references.
Listener configuration installs it for native clients unless an explicit native
adapter is supplied. Historical adapters continue to own their numeric palettes.
Local air is reserved at ID zero to preserve the item-stack non-block sentinel.

Fire and redstone encode .30 connection properties. Loading older states merges
missing defaults without replacing existing age or power, then derives connections
from neighbours. Visual-only wire updates suppress gameplay propagation to avoid
altering redstone burnout or scheduling. Focused state/connection tests and the
complete Dragonfly race/vet suites cover these boundaries.

This is an exact Preview upgrade, not evidence for an unreleased final 26.60 build.
No production deployment or native Mojang-client validation is implied.
