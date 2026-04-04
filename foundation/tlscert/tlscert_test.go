package tlscert

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateCA_Deterministic(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")

	_, caCert1, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA #1: %v", err)
	}

	_, caCert2, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA #2: %v", err)
	}

	pub1Bytes, err := x509.MarshalPKIXPublicKey(caCert1.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey #1: %v", err)
	}

	pub2Bytes, err := x509.MarshalPKIXPublicKey(caCert2.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey #2: %v", err)
	}

	if !bytes.Equal(pub1Bytes, pub2Bytes) {
		t.Fatal("CA public keys differ for the same seed")
	}

	if string(caCert1.SubjectKeyId) != string(caCert2.SubjectKeyId) {
		t.Fatal("CA SubjectKeyId differs for the same seed")
	}
}

func TestCrossVerification(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")

	caTLS1, caCert1, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA (peer1): %v", err)
	}

	_, caCert2, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA (peer2): %v", err)
	}

	leafCert1, err := GenerateLeaf(caCert1, caTLS1.PrivateKey)
	if err != nil {
		t.Fatalf("GenerateLeaf (peer1): %v", err)
	}

	pool2 := x509.NewCertPool()
	pool2.AddCert(caCert2)

	leafX509, err := x509.ParseCertificate(leafCert1.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}

	_, err = leafX509.Verify(x509.VerifyOptions{
		Roots: pool2,
	})
	if err != nil {
		t.Fatalf("peer2 failed to verify peer1's leaf cert: %v", err)
	}
}

func TestTLSHandshake_CrossPeer(t *testing.T) {
	seed := []byte("0123456789abcdef0123456789abcdef")

	caTLS1, caCert1, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA (server): %v", err)
	}
	leafCert1, err := GenerateLeaf(caCert1, caTLS1.PrivateKey)
	if err != nil {
		t.Fatalf("GenerateLeaf (server): %v", err)
	}

	_, caCert2, err := GenerateCA(seed)
	if err != nil {
		t.Fatalf("GenerateCA (client): %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{leafCert1}}
	srv.StartTLS()
	defer srv.Close()

	pool := x509.NewCertPool()
	pool.AddCert(caCert2)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("TLS handshake failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
