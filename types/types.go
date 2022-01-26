// Package types contains type defintions for all Bindle API objects
package types

import (
	"fmt"
	"strings"
)

// Invoice is the main structure for a Bindle invoice. The invoice describes a specific version of a
// bindle. For example, the bindle `foo/bar/1.0.0` would be represented as an Invoice with the
// `BindleSpec` name set to `foo/bar` and version set to `1.0.0`.
//
// Most fields on this struct are singular to best represent the specification. There, fields like
// `group` and `parcel` are singular due to the conventions of TOML.
type Invoice struct {
	BindleVersion string            `toml:"bindleVersion"`
	Yanked        *bool             `toml:"yanked"`
	Bindle        BindleSpec        `toml:"bindle"`
	Annotations   map[string]string `toml:"annotations,omitempty"`
	Signature     []Signature       `toml:"signature,omitempty"`
	Parcel        []Parcel          `toml:"parcel,omitempty"`
	Group         []Group           `toml:"group,omitempty"`
}

// Name returns the full name of the bindle (name + version)
func (i Invoice) Name() string {
	return fmt.Sprintf("%s/%s", i.Bindle.Name, i.Bindle.Version)
}

// NOTE: I tried to create an embedded ID type as we do in Rust so we can validate semver, but the
// TOML library isn't flexible enough to flatten the data

// BindleSpec contains the data to identify a bindle as well as additional metadata describing it
type BindleSpec struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description *string  `toml:"description"`
	Authors     []string `toml:"authors,omitempty"`
}

// Parcel is a description of a stored parcel file. A parcel file can be an arbitrary "blob" of
// data. This could be binary or text files. This object contains the metadata and associated
// conditions for using a parcel. For more information, see the Bindle Spec:
// https://github.com/deislabs/bindle/blob/master/docs/bindle-spec.md
type Parcel struct {
	Label      Label      `toml:"label"`
	Conditions *Condition `toml:"conditions"`
}

// Label is the metadata of a stored parcel. See the Label Spec for more detailed information:
// https://github.com/deislabs/bindle/blob/master/docs/label-spec.md
type Label struct {
	SHA256      string                       `toml:"sha256"`
	MediaType   string                       `toml:"mediaType"`
	Name        string                       `toml:"name"`
	Size        uint64                       `toml:"size"`
	Annotations map[string]string            `toml:"annotations,omitempty"`
	Feature     map[string]map[string]string `toml:"feature,omitempty"`
}

/// Condition is used to associate parcels to `Group`s
type Condition struct {
	MemberOf []string `toml:"memberOf,omitempty"`
	Requires []string `toml:"requires,omitempty"`
}

// Group is a top-level organization object that may contain zero or more parcels. Every parcel
// belongs to at least one group, but may belong to others.
type Group struct {
	Name        string  `toml:"name"`
	Required    *bool   `toml:"required"`
	SatisfiedBy *string `toml:"satisfiedBy"`
}

// Signature is a (default Ed25519) signature of the bindle based on the spec:
// https://github.com/deislabs/bindle/blob/main/docs/signing-spec.md
type Signature struct {
	By        string `toml:"by"`
	Signature string `toml:"signature"`
	Key       string `toml:"key"`
	Role      string `toml:"role"`
	At        int64  `toml:"at"`
}

// InvoiceCreateResponse is returned by a Bindle server when creating an invoice. It contains the
// created invoice and an optional slice of labels indicating which parcels are missing in storage
type InvoiceCreateResponse struct {
	Invoice Invoice `toml:"invoice"`
	Missing []Label `toml:"missing,omitempty"`
}

// MissingParcelsResponse is a response to a missing parcels request. TOML doesn't support top level arrays, so they
// must be embedded in a table
type MissingParcelsResponse struct {
	Missing []Label `toml:"missing"`
}

// ErrorResponse is a string error message returned from the server
type ErrorResponse struct {
	Error string `toml:"error"`
}

// QueryOptions represents available options for the query API
type QueryOptions struct {
	Query   *string `toml:"q"`
	Version *string `toml:"v"`
	Offset  *uint64 `toml:"o"`
	Limit   *uint8  `toml:"l"`
	Strict  *bool   `toml:"strict"`
	Yanked  *bool   `toml:"yanked"`
}

// QueryString returns a query string suitable for use in a URL (including the starting `?`) using
// the configured parameters of a `QueryOptions`
func (q *QueryOptions) QueryString() string {
	var pairs []string
	if q.Query != nil {
		pairs = append(pairs, fmt.Sprintf("q=%s", *q.Query))
	}
	if q.Version != nil {
		pairs = append(pairs, fmt.Sprintf("v=%s", *q.Version))
	}
	if q.Offset != nil {
		pairs = append(pairs, fmt.Sprintf("o=%d", q.Offset))
	}
	if q.Limit != nil {
		pairs = append(pairs, fmt.Sprintf("l=%d", q.Limit))
	}
	if q.Strict != nil {
		pairs = append(pairs, fmt.Sprintf("strict=%v", q.Strict))
	}
	if q.Yanked != nil {
		pairs = append(pairs, fmt.Sprintf("yanked=%v", q.Yanked))
	}

	return "?" + strings.Join(pairs, "&")
}

// Matches describes the matches that are returned from a query
type Matches struct {
	// The query used to find this match set
	Query string `toml:"query"`
	// Whether the search engine used strict mode
	Strict bool `toml:"strict"`
	// The offset of the first result in the matches
	Offset uint64 `toml:"offset"`
	// The maximum number of results this query would have returned
	Limit uint8 `toml:"limit"`
	// The total number of matches the search engine located
	//
	// In many cases, this will not match the number of results returned on this query
	Total uint64 `toml:"total"`
	// Whether there are more results than the ones returned here
	More bool `toml:"more"`
	// Whether this list includes potentially yanked invoices
	Yanked bool `toml:"yanked"`
	// The list of invoices returned as this part of the query
	//
	// The length of this Vec will be less than or equal to the limit.
	Invoices []Invoice `toml:"invoices"`
}
