// Package client is a fully featured client for talking to a Bindle server. Underneath the hood,
// Bindle uses HTTP/2 for communicating with the Bindle server. This enables a consumer to make
// multiple parallel requests for parcels using the same underlying connection
//
// Bindle IDs and SHAs
//
// Many of the client functions take `bindleID` and `sha` parameters. Bindle IDs are arbitrarily
// pathy names (e.g. example.com/foo/bar) plus a strict semver version (e.g. 1.0.0), resulting in an
// ID of "example.com/foo/bar/1.0.0". The sha parameter is a SHA256 sum of a given parcel and should
// match the SHA of the data you are sending, otherwise it will be rejected
package client
