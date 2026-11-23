// Package registry owns static miniapp registration: manifest
// validation, implementation binding, duplicate rejection, and
// release-envelope verification. Registration is explicit (no
// init-time magic, no hot-loading); the service binary registers
// each miniapp with :version identity. Resolution is exact-match
// only — there is no "latest".
package registry
