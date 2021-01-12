package types

import (
	"fmt"
	"path"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// ID represents the canonical ID of a Bindle
type ID struct {
	name    string
	version semver.Version
}

// ParseID takes a raw string and attempts to parse it as an `ID`. A valid Bindle ID is an
// arbitrarily pathy name (e.g. example.com/foo/bar) and a strict semver version (e.g. 1.0.0) in the
// format "example.com/foo/bar/1.0.0"
func ParseID(raw string) (ID, error) {
	name := path.Dir(raw)
	version, err := semver.StrictNewVersion(path.Base(raw))
	if err != nil {
		return ID{}, err
	}
	return ID{name, *version}, nil
}

func (i ID) String() string {
	return path.Join(i.name, i.version.String())
}

// Invoice is the main structure for a Bindle invoice.
//
// The invoice describes a specific version of a bindle. For example, the bindle `foo/bar/1.0.0`
// would be represented as an Invoice with the `BindleSpec` name set to `foo/bar` and version set to
// `1.0.0`.
//
// Most fields on this struct are singular to best represent the specification. There, fields like
// `group` and `parcel` are singular due to the conventions of TOML.
type Invoice struct {
	BindleVersion string
	Yanked        *bool
	Bindle        BindleSpec
	Annotations   map[string]string
	Parcel        []Parcel
	Group         []Group
}

// BindleSpec contains the data to identify a bindle as well as additional metadata describing it
type BindleSpec struct {
	// TODO: Figure out how to flatten this to name/version
	ID          ID
	Description *string
	Authors     []string
}

// Parcel is a description of a stored parcel file
//
// A parcel file can be an arbitrary "blob" of data. This could be binary or text files. This
// object contains the metadata and associated conditions for using a parcel. For more information,
// see the [Bindle Spec](https://github.com/deislabs/bindle/blob/master/docs/bindle-spec.md)
type Parcel struct {
	Label      Label
	Conditions *Condition
}

// Label is the metadata of a stored parcel
//
// See the [Label Spec](https://github.com/deislabs/bindle/blob/master/docs/label-spec.md) for more
// detailed information
type Label struct {
	SHA256      string
	MediaType   string
	Name        string
	Size        uint64
	Annotations map[string]string
	Feature     map[string]map[string]string
}

/// Condition associate parcels to `Group`s
type Condition struct {
	MemberOf []string
	Requires []string
}

// Group is a top-level organization object that may contain zero or more parcels. Every parcel
// belongs to at least one group, but may belong to others.
type Group struct {
	Name        string
	Required    *bool
	SatisfiedBy *string
}

// InvoiceCreateResponse is returned by a Bindle server when creating an invoice. It contains the
// created invoice and an optional slice of labels indicating which parcels are missing in storage
type InvoiceCreateResponse struct {
	Invoice Invoice
	Missing []Label
}

// MissingParcelsResponse is a response to a missing parcels request. TOML doesn't support top level arrays, so they
// must be embedded in a table
type MissingParcelsResponse struct {
	Missing []Label
}

// ErrorResponse is a string error message returned from the server
type ErrorResponse struct {
	Error string
}

// QueryOptions represents available options for the query API
type QueryOptions struct {
	// #[serde(alias = "q")]
	Query *string
	// #[serde(alias = "v")]
	Version *string
	// #[serde(alias = "o")]
	Offset *uint64
	// #[serde(alias = "l")]
	Limit  *uint8
	Strict *bool
	Yanked *bool
}

// Returns a query string suitable to use in a URL
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
	Query string
	// Whether the search engine used strict mode
	Strict bool
	// The offset of the first result in the matches
	Offset uint64
	// The maximum number of results this query would have returned
	Limit uint8
	// The total number of matches the search engine located
	//
	// In many cases, this will not match the number of results returned on this query
	Total uint64
	// Whether there are more results than the ones returned here
	More bool
	// Whether this list includes potentially yanked invoices
	Yanked bool
	// The list of invoices returned as this part of the query
	//
	// The length of this Vec will be less than or equal to the limit.
	Invoices []Invoice
}
