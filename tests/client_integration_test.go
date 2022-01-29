package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/deislabs/go-bindle/client"
	"github.com/deislabs/go-bindle/keyring"
	"github.com/deislabs/go-bindle/types"

	"github.com/pelletier/go-toml"
)

const testAuthor = `Testy McTestface <"testy@test.face">`
const testAuthor2 = `Elon Testla <"elon@testla.com">`

var key string
var cert string

func TestMain(m *testing.M) {
	// Generate the cert first so it can be reused everywhere
	key, cert = generateSelfSignedCert()
	code := m.Run()
	os.RemoveAll(filepath.Dir(key))
	os.Exit(code)
}

type testController struct {
	Client client.Client
	// Don't know if we actually need this, but including it for now
	cmd exec.Cmd
}

func newTestController(t *testing.T) testController {
	t.Helper()
	serverBinaryPath, exists := os.LookupEnv("BINDLE_SERVER_PATH")
	if !exists {
		foundPath, err := exec.LookPath("bindle-server")
		if err != nil {
			t.Fatalf("Bindle server path was not specified and cannot find a bindle server in the PATH: %s", err)
		}
		serverBinaryPath = foundPath
	}

	tempdir, err := ioutil.TempDir("", "*")
	if err != nil {
		t.Fatalf("Unable to create tempdir for testing: %s", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempdir) })

	// Find an open port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Unable to find open port: %s", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	t.Log(key)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, serverBinaryPath,
		"-d", tempdir,
		"-i", address,
		"-c", cert,
		"-k", key,
		"--unauthenticated")

	t.Cleanup(cancel)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		t.Fatalf("Unable to start server process: %s", err)
	}

	bindleClient, err := client.New(fmt.Sprintf("https://%s/v1/", address), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the server to start up
	started := false
	for i := 0; i < 5; i++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			t.Logf("Bindle server not ready (attempt %d), will retry in 1s", i+1)
			time.Sleep(time.Second)
			continue
		} else {
			started = true
			conn.Close()
			break
		}
	}

	if !started {
		t.Fatal("Timed out waiting for bindle server to start")
	}

	return testController{
		Client: *bindleClient,
		cmd:    *cmd,
	}
}

// Because the http2 support in Go doesn't seem to allow you to use http with http2 without a bunch
// of config things that only apply here, we need a pair of certs to use for testing. Copied and
// modified from https://gist.github.com/samuel/8b500ddd3f6118d052b5e6bc16bc4c09
func generateSelfSignedCert() (keyPath string, certPath string) {
	tempdir, err := ioutil.TempDir("", "*")
	if err != nil {
		panic(fmt.Sprintf("Unable to create tempdir for certs: %s", err))
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("Unable to create self-signed cert")
	}
	b := x509.MarshalPKCS1PrivateKey(privKey)
	keyBlock := pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}

	keyPath = tempdir + "/key.pem"
	file, err := os.Create(keyPath)
	if err != nil {
		panic(fmt.Sprintf("Unable to create private key file: %s", err))
	}

	if err := pem.Encode(file, &keyBlock); err != nil {
		panic(fmt.Sprintf("Unable to encode private key: %s", err))
	}

	file.Close()

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to create certificate: %s", err))
	}

	certPath = tempdir + "/cert.pem"
	file, err = os.Create(certPath)
	if err != nil {
		panic(fmt.Sprintf("Unable to create cert file: %s", err))
	}

	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		panic(fmt.Sprintf("Unable to encode cert: %s", err))
	}
	file.Close()
	return
}

func scaffold_invoice_path(name string) string {
	return fmt.Sprintf("./scaffolds/%s/invoice.toml", name)
}

func scaffold_parcel_path(invoiceName string, parcelName string) string {
	return fmt.Sprintf("./scaffolds/%s/parcels/%s.dat", invoiceName, parcelName)
}

func load_scaffold_invoice(t *testing.T, name string) types.Invoice {
	t.Helper()
	file, err := os.Open(scaffold_invoice_path(name))
	if err != nil {
		t.Fatalf("Unable to open invoice file: %s", err)
	}
	var inv types.Invoice
	if err := toml.NewDecoder(file).Strict(true).Decode(&inv); err != nil {
		t.Fatalf("Unable to unmarshal invoice: %s", err)
	}
	return inv
}

func load_scaffold_parcel_data(t *testing.T, invoiceName string, parcelName string) []byte {
	t.Helper()
	raw, err := ioutil.ReadFile(scaffold_parcel_path(invoiceName, parcelName))
	if err != nil {
		t.Fatalf("Unable to open parcel file: %s", err)
	}
	return raw
}

//////////////////////////////////////////////////////////
/////////////////// BEGIN ACTUAL TESTS ///////////////////
//////////////////////////////////////////////////////////

func TestSuccessful(t *testing.T) {
	controller := newTestController(t)

	// First try creating an invoice
	inv := load_scaffold_invoice(t, "valid_v1")
	_, err := controller.Client.CreateInvoice(inv)
	if err != nil {
		t.Fatalf("Unable to create invoice: %s", err)
	}

	// Now create the parcel associated with that invoice
	data := load_scaffold_parcel_data(t, "valid_v1", "parcel")
	if err := controller.Client.CreateParcel(inv.Name(), inv.Parcel[0].Label.SHA256, data); err != nil {
		t.Fatalf("Unable to create parcel: %s", err)
	}

	// Now see if we get the parcel back from the server
	serverData, err := controller.Client.GetParcel(inv.Name(), inv.Parcel[0].Label.SHA256)
	if err != nil {
		t.Fatalf("Unable to fetch parcel from server: %s", err)
	}

	if !reflect.DeepEqual(data, serverData) {
		t.Fatalf("Did not get back valid data from the server\nExpected: %s\nGot: %s", data, serverData)
	}

	// Now try yanking the parcel and fetching it to make sure it gives us an error
	if err := controller.Client.YankInvoice(inv.Name()); err != nil {
		t.Fatalf("Unable to yank invoice: %s", err)
	}

	_, err = controller.Client.GetInvoice(inv.Name())
	if err == nil {
		t.Fatal("Shouldn't be able to get a yanked invoice")
	}

	// Get the yanked invoice and make sure it works
	_, err = controller.Client.GetYankedInvoice(inv.Name())
	if err != nil {
		t.Fatalf("Should be able to get a yanked invoice: %s", err)
	}
}

func TestStreamingSuccessful(t *testing.T) {
	controller := newTestController(t)

	// First try creating an invoice
	resp, err := controller.Client.CreateInvoiceFromFile(scaffold_invoice_path("valid_v1"))
	if err != nil {
		t.Fatalf("Unable to create invoice: %s", err)
	}

	inv := resp.Invoice

	// Now create the parcel associated with that invoice
	data := load_scaffold_parcel_data(t, "valid_v1", "parcel")
	if err := controller.Client.CreateParcelFromFile(inv.Name(), inv.Parcel[0].Label.SHA256, scaffold_parcel_path("valid_v1", "parcel")); err != nil {
		t.Fatalf("Unable to create parcel: %s", err)
	}

	// Now see if we get the parcel back from the server
	serverData, err := controller.Client.GetParcel(inv.Name(), inv.Parcel[0].Label.SHA256)
	if err != nil {
		t.Fatalf("Unable to fetch parcel from server: %s", err)
	}

	if !reflect.DeepEqual(data, serverData) {
		t.Fatalf("Did not get back valid data from the server\nExpected: %s\nGot: %s", data, serverData)
	}
}

func TestAlreadyCreated(t *testing.T) {
	controller := newTestController(t)

	// Create an invoice with two parcels
	inv := load_scaffold_invoice(t, "valid_v2")
	_, err := controller.Client.CreateInvoice(inv)
	if err != nil {
		t.Fatalf("Unable to create invoice: %s", err)
	}

	// Now create the parcels associated with that invoice
	data := load_scaffold_parcel_data(t, "valid_v2", "other")
	if err := controller.Client.CreateParcel(inv.Name(), inv.Parcel[0].Label.SHA256, data); err != nil {
		t.Fatalf("Unable to create parcel: %s", err)
	}

	data = load_scaffold_parcel_data(t, "valid_v2", "parcel")
	if err := controller.Client.CreateParcel(inv.Name(), inv.Parcel[1].Label.SHA256, data); err != nil {
		t.Fatalf("Unable to create parcel: %s", err)
	}

	// Now create another invoice that already has all parcels existing
	inv = load_scaffold_invoice(t, "valid_v1")
	resp, err := controller.Client.CreateInvoice(inv)
	if err != nil {
		t.Fatalf("Unable to create invoice: %s", err)
	}

	if resp.Missing != nil {
		t.Fatalf("Should have no missing parcels")
	}
}

func TestMissing(t *testing.T) {
	controller := newTestController(t)

	// Create an invoice with two parcels
	inv := load_scaffold_invoice(t, "valid_v2")
	_, err := controller.Client.CreateInvoice(inv)
	if err != nil {
		t.Fatalf("Unable to create invoice: %s", err)
	}

	missing, err := controller.Client.GetMissingParcels(inv.Name())
	if err != nil {
		t.Fatalf("Should have been able to get missing parcels: %s", err)
	}

	if len(missing.Missing) != len(inv.Parcel) {
		t.Fatalf("Expected to get %d missing parcels, got %d", len(inv.Parcel), len(missing.Missing))
	}
}

func TestSignVerify(t *testing.T) {
	sigKey, privKey, err := keyring.GenerateSignatureKey(testAuthor, types.RoleCreator)
	if err != nil {
		t.Error(err)
		return
	}

	data := []byte("something very important")

	importantParcel := types.NewParcel("importantfile", "application/important", data)

	invoice := &types.Invoice{
		BindleVersion: "1.0.0",
		Bindle: types.BindleSpec{
			Name:    "importantproj",
			Version: "0.1.0",
			Authors: []string{
				testAuthor,
			},
		},
		Parcel: []types.Parcel{
			importantParcel,
		},
	}

	if err := invoice.GenerateSignature(testAuthor, types.RoleCreator, sigKey, privKey); err != nil {
		t.Error(err)
		return
	}

	if err := invoice.VerifySignatures([]types.SignatureKey{*sigKey}); err != nil {
		t.Error(err)
		return
	}
}

func TestSignVerifyWrongKey(t *testing.T) {
	sigKey, privKey, err := keyring.GenerateSignatureKey(testAuthor, types.RoleCreator)
	if err != nil {
		t.Error(err)
		return
	}

	data := []byte("something very important")

	importantParcel := types.NewParcel("importantfile", "application/important", data)

	invoice := &types.Invoice{
		BindleVersion: "1.0.0",
		Bindle: types.BindleSpec{
			Name:    "importantproj",
			Version: "0.1.0",
			Authors: []string{
				testAuthor,
			},
		},
		Parcel: []types.Parcel{
			importantParcel,
		},
	}

	if err := invoice.GenerateSignature(testAuthor, types.RoleCreator, sigKey, privKey); err != nil {
		t.Error(err)
		return
	}

	// a second key with same author to ensure verification fails with something "spoofed"
	sigKey2, _, err := keyring.GenerateSignatureKey(testAuthor, types.RoleCreator)
	if err != nil {
		t.Error(err)
		return
	}

	if err := invoice.VerifySignatures([]types.SignatureKey{*sigKey2}); err == nil {
		t.Error(errors.New("did not get signing error, should have"))
		return
	}
}

func TestSignVerifyMissingKey(t *testing.T) {
	sigKey, privKey, err := keyring.GenerateSignatureKey(testAuthor, types.RoleCreator)
	if err != nil {
		t.Error(err)
		return
	}

	data := []byte("something very important")

	importantParcel := types.NewParcel("importantfile", "application/important", data)

	invoice := &types.Invoice{
		BindleVersion: "1.0.0",
		Bindle: types.BindleSpec{
			Name:    "importantproj",
			Version: "0.1.0",
			Authors: []string{
				testAuthor,
			},
		},
		Parcel: []types.Parcel{
			importantParcel,
		},
	}

	if err := invoice.GenerateSignature(testAuthor, types.RoleCreator, sigKey, privKey); err != nil {
		t.Error(err)
		return
	}

	// a key with a different author to ensure verification fails when the correct author's key isn't present
	sigKey2, _, err := keyring.GenerateSignatureKey(testAuthor2, types.RoleCreator)
	if err != nil {
		t.Error(err)
		return
	}

	if err := invoice.VerifySignatures([]types.SignatureKey{*sigKey2}); err == nil {
		t.Error(errors.New("did not get signing error, should have"))
		return
	}
}
