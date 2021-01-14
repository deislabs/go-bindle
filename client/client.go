package client

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/deislabs/go-bindle/types"

	"github.com/pelletier/go-toml"
	"golang.org/x/net/http2"
)

const invoiceEndpoint = "_i"
const queryEndpoint = "_q"
const relationshipEndpoint = "_r"
const tomlMimeType = "application/toml"

// Client is the struct that contains all necessary information for communicating with a Bindle
// Server
type Client struct {
	httpClient http.Client
	baseURL    *url.URL
}

// New returns a new Client configured to use the given baseURL. This URL should be the entire base
// part of your Bindle server. So if your Bindle server is namespaced (with something like v1), then
// the baseURL should contain that part of the URL (e.g. https://bindle.example.com/v1 instead of
// https://bindle.example.com). The tlsConfig parameter is optional and can be used if you have any
// specific TLS configuration options such as internally signed certificates
func New(baseURL string, tlsConfig *tls.Config) (*Client, error) {
	httpClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: tlsConfig,
		},
	}

	// Strip any trailing slashes first
	stripped := strings.TrimSuffix(baseURL, "/")
	// Validate the baseURL
	u, err := url.Parse(stripped)
	if err != nil {
		return nil, fmt.Errorf("Invalid base URL: %s", err)
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    u,
	}, nil
}

// RawRequests performs an HTTP request using the underlying HTTP client and base URL. The given
// path is appended to the URL and the data body is optional. If a body is specified, the
// contentType can be specified as well, otherwise contentType will be ignored
func (c *Client) RawRequest(path string, method string, data io.ReadCloser, contentType string) (*http.Response, error) {
	u := *c.baseURL
	// Parse as a URL so we can get the separate components
	parsedPath, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u.Path = u.Path + parsedPath.Path
	u.RawQuery = parsedPath.RawQuery

	req := &http.Request{
		Method: method,
		URL:    &u,
		Body:   data,
		Header: http.Header{
			"Content-Type": []string{contentType},
		},
	}
	return c.httpClient.Do(req)
}

func (c *Client) requestAndUnmarshal(path string, method string, data io.ReadCloser, contentType string, v interface{}) error {
	resp, err := c.RawRequest(path, method, data, contentType)
	if err != nil {
		return err
	}
	if err := unmarshalResponse(resp, v); err != nil {
		return err
	}
	return nil
}

// GetInvoice returns an `Invoice` with the given ID. This will return an error if the invoice is
// yanked
func (c *Client) GetInvoice(id string) (*types.Invoice, error) {
	var inv types.Invoice
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s", invoiceEndpoint, id), http.MethodGet, nil, "", &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetYankedInvoice is the same as `GetInvoice`, but allows you to return an invoice that has been
// yanked
func (c *Client) GetYankedInvoice(id string) (*types.Invoice, error) {
	var inv types.Invoice
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s?yanked=true", invoiceEndpoint, id), http.MethodGet, nil, "", &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

// CreateInvoice from the given `Invoice` object. Returns a response containing the newly created
// invoice and a list of any missing parcels that need to be uploaded
func (c *Client) CreateInvoice(inv types.Invoice) (*types.InvoiceCreateResponse, error) {
	body, err := encodeToBuffer(&inv)
	if err != nil {
		return nil, err
	}

	var invResp types.InvoiceCreateResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s", invoiceEndpoint), http.MethodPost, body, tomlMimeType, &invResp); err != nil {
		return nil, err
	}

	return &invResp, nil
}

// CreateInvoiceFromFile is the same as `CreateInvoice`, but instead takes a path to an invoice TOML
// file to send to the server
func (c *Client) CreateInvoiceFromFile(path string) (*types.InvoiceCreateResponse, error) {
	body, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var invResp types.InvoiceCreateResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s", invoiceEndpoint), http.MethodPost, body, tomlMimeType, &invResp); err != nil {
		return nil, err
	}
	return &invResp, nil
}

// QueryInvoices allows you to search and filter invoices from the Bindle server. Bindle servers are
// allowed to implement their query engines differently, so the number of matches may vary depending
// on the server, particularly in their use of `strict` mode. Returns a `Matches` object containing
// information about the query, pagination data, and the list of responses
func (c *Client) QueryInvoices(opts types.QueryOptions) (*types.Matches, error) {
	var matches types.Matches
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s%s", queryEndpoint, opts.QueryString()), http.MethodGet, nil, tomlMimeType, &matches); err != nil {
		return nil, err
	}
	return &matches, nil
}

// YankInvoice allows you to "yank," or mark as no longer available, an invoice of the given ID.
// Invoices cannot be deleted from a Bindle server, so this notifies users that it should not be
// consumed
func (c *Client) YankInvoice(id string) error {
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s", invoiceEndpoint, id), http.MethodDelete, nil, "", nil); err != nil {
		return err
	}
	return nil
}

// Performs the request against the parcel endpoint and handles any http errors, returning the HTTP body
func (c *Client) doParcelRequest(bindleID string, sha string, method string, body io.ReadCloser) (io.ReadCloser, error) {
	resp, err := c.RawRequest(fmt.Sprintf("/%s/%s@%s", invoiceEndpoint, bindleID, sha), method, body, "")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, unmarshalResponse(resp, nil)
	}
	return resp.Body, nil
}

// GetParcel returns the parcel identified by the Bindle ID and parcel SHA. This loads the data into
// memory as a byte array and is not recommended for use with larger parcels. For larger parcels (or
// when writing directly to another source), use the `GetParcelReader` function instead
func (c *Client) GetParcel(bindleID string, sha string) ([]byte, error) {
	body, err := c.doParcelRequest(bindleID, sha, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	return ioutil.ReadAll(body)
}

// GetParcelReader is similar to `GetParcel` but returns the parcel as a reader (for streaming
// purposes). This will be more efficient for larger files
func (c *Client) GetParcelReader(bindleID string, sha string) (io.ReadCloser, error) {
	return c.doParcelRequest(bindleID, sha, http.MethodGet, nil)
}

// CreateParcel uploads a parcel for the given `bindleID`. The `sha` value must match the SHA256 sum
// of the data or the server will reject the parcel. This function takes the parcel data as a raw
// byte array. For larger parcels, it is recommended to use `CreateParcelFromFile` or
// `CreateParcelFromReader` to avoid loading them into memory.
//
// Please note that for best efficiency, consumers should only upload parcels that do not already
// exist as indicated by the server (either in the `InvoiceCreateResponse` or by using the
// `GetMissingParcels` function)
func (c *Client) CreateParcel(bindleID string, sha string, data []byte) error {
	_, err := c.doParcelRequest(bindleID, sha, http.MethodPost, ioutil.NopCloser(bytes.NewReader(data)))
	return err
}

// CreateParcelFromFile is the same as `CreateParcel` but takes a path to a file to upload for a
// parcel. This file will be streamed to the server and not loaded into memory
func (c *Client) CreateParcelFromFile(bindleID string, sha string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = c.doParcelRequest(bindleID, sha, http.MethodPost, file)
	return err
}

// CreateParcelFromReader is the same as `CreateParcel` but takes anything that is a `ReadCloser` to
// use for the parcel. This function will stream the data from the reader to the server and then
// close the reader
func (c *Client) CreateParcelFromReader(bindleID string, sha string, data io.ReadCloser) error {
	_, err := c.doParcelRequest(bindleID, sha, http.MethodPost, data)
	defer data.Close()
	return err
}

// GetMissingParcels checks with the server if there are any missing parcels for the given Bindle
// ID. Returns a response containing the list of missing parcels, if any
func (c *Client) GetMissingParcels(id string) (*types.MissingParcelsResponse, error) {
	var missing types.MissingParcelsResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/missing/%s", relationshipEndpoint, id), http.MethodGet, nil, "", &missing); err != nil {
		return nil, err
	}
	return &missing, nil
}

func unmarshalResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		var errorInfo types.ErrorResponse
		var err error
		// Try to get an error message. Not all errors will have them, so do not error out if it
		// fails
		if toml.NewDecoder(resp.Body).Strict(true).Decode(&errorInfo) == nil {
			err = fmt.Errorf("Error making request (HTTP status code %v): %s", resp.StatusCode, errorInfo.Error)
		} else {
			err = fmt.Errorf("Error making request (HTTP status code %v)", resp.StatusCode)
		}
		return err
	}
	// Sometimes we want to try and unmarshal the error above, but not handle the body
	if v != nil {
		if err := toml.NewDecoder(resp.Body).Strict(true).Decode(v); err != nil {
			return err
		}
	}
	return nil
}

func encodeToBuffer(v interface{}) (io.ReadCloser, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return ioutil.NopCloser(&buf), nil
}
