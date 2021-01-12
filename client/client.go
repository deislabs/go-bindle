package client

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/deislabs/go-bindle/types"

	"golang.org/x/net/http2"
)

type Client struct {
	httpClient http.Client
	baseURL    string
}

func New(baseURL string) (*Client, error) {
	httpClient := http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
		},
	}
	// Validate the baseURL
	_, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid base URL: %s", err)
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

func (c *Client) GetInvoice(id string) (types.Invoice, error) {
	return types.Invoice{}, nil
}

func (c *Client) CreateInvoice(inv types.Invoice) (types.InvoiceCreateResponse, error) {
	return types.InvoiceCreateResponse{}, nil
}

func (c *Client) CreateInvoiceFromFile(path string) (types.InvoiceCreateResponse, error) {
	return types.InvoiceCreateResponse{}, nil
}
