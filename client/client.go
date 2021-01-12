package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/deislabs/go-bindle/types"

	"github.com/pelletier/go-toml"
	"golang.org/x/net/http2"
)

const invoiceEndpoint = "_i"
const queryEndpoint = "_q"
const relationshipEndpoint = "_r"

type Client struct {
	httpClient http.Client
	baseURL    *url.URL
}

func New(baseURL string) (*Client, error) {
	httpClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
		},
	}
	// Validate the baseURL
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid base URL: %s", err)
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    u,
	}, nil
}

// RawRequests performs an HTTP request using the underlying HTTP client and base URL. The given
// path is appended to the URL and the data body is optional
func (c *Client) RawRequest(path string, method string, data io.ReadCloser) (*http.Response, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, err
	}
	req := &http.Request{
		Method: method,
		URL:    u,
		Body:   data,
	}
	return c.httpClient.Do(req)
}

func (c *Client) requestAndUnmarshal(path string, method string, data io.ReadCloser, v interface{}) error {
	resp, err := c.RawRequest(path, method, data)
	if err != nil {
		return err
	}
	if err := unmarshalResponse(resp, v); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetInvoice(id string) (*types.Invoice, error) {
	var inv types.Invoice
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s", invoiceEndpoint, id), http.MethodGet, nil, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (c *Client) GetYankedInvoice(id string) (*types.Invoice, error) {
	var inv types.Invoice
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s?yanked=true", invoiceEndpoint, id), http.MethodGet, nil, &inv); err != nil {
		return nil, err
	}
	return &inv, nil
}

func (c *Client) CreateInvoice(inv types.Invoice) (*types.InvoiceCreateResponse, error) {
	body, err := encodeToBuffer(&inv)
	if err != nil {
		return nil, err
	}

	var invResp types.InvoiceCreateResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s", invoiceEndpoint), http.MethodPost, body, &invResp); err != nil {
		return nil, err
	}

	return &invResp, nil
}

func (c *Client) CreateInvoiceFromFile(path string) (*types.InvoiceCreateResponse, error) {
	body, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var invResp types.InvoiceCreateResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s", invoiceEndpoint), http.MethodPost, body, &invResp); err != nil {
		return nil, err
	}
	return &invResp, nil
}

func (c *Client) QueryInvoices(opts types.QueryOptions) (*types.Matches, error) {
	var matches types.Matches
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s%s", queryEndpoint, opts.QueryString()), http.MethodGet, nil, &matches); err != nil {
		return nil, err
	}
	return &matches, nil
}

func (c *Client) YankInvoice(id string) error {
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/%s", invoiceEndpoint, id), http.MethodDelete, nil, nil); err != nil {
		return err
	}
	return nil
}

// Performs the request against the parcel endpoint and handles any http errors, returning the HTTP body
func (c *Client) doParcelRequest(bindleID string, sha string, method string, body io.ReadCloser) (io.ReadCloser, error) {
	resp, err := c.RawRequest(fmt.Sprintf("/%s/%s@%s", invoiceEndpoint, bindleID, sha), method, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return nil, unmarshalResponse(resp, nil)
	}
	return resp.Body, nil
}

func (c *Client) GetParcel(bindleID string, sha string) ([]byte, error) {
	body, err := c.doParcelRequest(bindleID, sha, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	defer body.Close()
	return ioutil.ReadAll(body)
}

// Returns the parcel as a reader (for streaming purposes). This will be more efficient for larger
// files
func (c *Client) GetParcelReader(bindleID string, sha string) (io.ReadCloser, error) {
	return c.doParcelRequest(bindleID, sha, http.MethodGet, nil)
}

func (c *Client) CreateParcel(bindleID string, sha string, data []byte) error {
	_, err := c.doParcelRequest(bindleID, sha, http.MethodPost, ioutil.NopCloser(bytes.NewReader(data)))
	return err
}

func (c *Client) CreateParcelFromFile(bindleID string, sha string, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = c.doParcelRequest(bindleID, sha, http.MethodPost, file)
	return err
}

func (c *Client) CreateParcelFromReader(bindleID string, sha string, data io.ReadCloser) error {
	_, err := c.doParcelRequest(bindleID, sha, http.MethodPost, data)
	return err
}

func (c *Client) GetMissingParcels(id string) (*types.MissingParcelsResponse, error) {
	var missing types.MissingParcelsResponse
	if err := c.requestAndUnmarshal(fmt.Sprintf("/%s/missing/%s", relationshipEndpoint, id), http.MethodGet, nil, &missing); err != nil {
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
