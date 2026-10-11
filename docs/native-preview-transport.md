# Native Preview compile bridge

This isolated change permits the dual-transport proxy fixture to compile against
the native 1.26.60 candidate. Comparing gophertunnel `9e7f28092180` with
`8fe51a21de6d` shows that `protocol.Skin.PlayFabID` and its wire string were removed.
The two Dragonfly session conversion references are removed accordingly; other
appearance metadata and the input skin remain unchanged. Historical skin wire
differences remain owned by go-multiversion, not Dragonfly.

No gophertunnel replacement or native registry/data upgrade is committed here.
Full tests and vet use disposable module files selecting the isolated candidate.
This branch must be integrated only alongside the selected native dependency;
it is not a standalone change for the outgoing stable skin codec. Blocks, items,
recipes and creative inventory are still the existing data and do not establish
Preview support. No production deployment or client validation was performed.
