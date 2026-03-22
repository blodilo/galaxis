package economy

// PlayerZeroID is the hardcoded player ID used in the MVP before Keycloak
// authentication is integrated (AP3). All economy handlers fall back to this
// value when the X-Player-ID header is absent.
//
// AP3 note: replace header extraction with JWT sub claim; this constant can then
// be removed.
const PlayerZeroID = "00000000-0000-0000-0000-000000000001"
