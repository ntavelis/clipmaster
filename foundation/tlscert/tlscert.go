// Package tlscert generates in-memory TLS certificates for clipmaster peers.
package tlscert

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"golang.org/x/crypto/hkdf"
)

// deterministicP256Key derives a P-256 ECDSA private key from an HKDF reader.
// ecdsa.GenerateKey is not deterministic even with a deterministic reader (Go injects
// internal randomness), so we derive the scalar directly from HKDF bytes.
func deterministicP256Key(reader io.Reader) (*ecdsa.PrivateKey, error) {
	keyBytes := make([]byte, 32)
	if _, err := io.ReadFull(reader, keyBytes); err != nil {
		return nil, err
	}

	n := elliptic.P256().Params().N
	d := new(big.Int).SetBytes(keyBytes)
	d.Mod(d, new(big.Int).Sub(n, big.NewInt(1)))
	d.Add(d, big.NewInt(1))

	dBytes := make([]byte, 32)
	dRaw := d.Bytes()
	copy(dBytes[32-len(dRaw):], dRaw)

	ecdhKey, err := ecdh.P256().NewPrivateKey(dBytes)
	if err != nil {
		return nil, err
	}

	der, err := x509.MarshalPKCS8PrivateKey(ecdhKey)
	if err != nil {
		return nil, err
	}

	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}

	ecdsaKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unexpected key type %T", parsed)
	}

	return ecdsaKey, nil
}

// GenerateCA derives a deterministic ECDSA P-256 CA certificate from seed.
// All peers that share the same passphrase will derive the same CA and can
// therefore verify each other's leaf certificates without InsecureSkipVerify.
func GenerateCA(seed []byte) (tls.Certificate, *x509.Certificate, error) {
	caKey, err := deterministicP256Key(hkdf.New(sha256.New, seed, []byte("clipmaster-ca-v1"), nil))
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "clipmaster-ca"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &caKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	caCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	tlsCert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
	if err != nil {
		return tls.Certificate{}, nil, err
	}

	return tlsCert, caCert, nil
}

// GenerateLeaf creates a random ECDSA P-256 server certificate signed by the given CA.
// The certificate includes SANs for all local IPs and localhost.
func GenerateLeaf(caCert *x509.Certificate, caKey crypto.PrivateKey) (tls.Certificate, error) {
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "clipmaster"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  localIPs(),
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
	)
}

func localIPs() []net.IP {
	ips := []net.IP{net.ParseIP("127.0.0.1")}

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					ips = append(ips, ip4)
				}
			}
		}
	}

	return ips
}
