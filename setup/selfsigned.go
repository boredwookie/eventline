// Source: https://golang.org/src/crypto/tls/generate_cert.go
//		BSD licensed
package setup

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

// Takes a comma delimited list of FQDNs that the certificate should be valid for
//		Generates 'cert.pem' and 'key.pem'
func GenerateSelfSignedCert(fqdns string){
	var notBefore = time.Now()
	notAfter := notBefore.Add(366*24*time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Eventline Local Server"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(fqdns, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// This is a self-signed certificate
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	// P-256 (see FIPS 186-3, section D.2.3)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	// Write out the certificate
	certOut, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("failed to write data to cert.pem: %s", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("error closing cert.pem: %s", err)
	}
	log.Print("wrote cert.pem\n")

	// Write out the key
	keyOut, err := os.OpenFile("key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("failed to open key.pem for writing:", err)
		return
	}
	if err := pem.Encode(keyOut, pemBlockForKey(priv)); err != nil {
		log.Fatalf("failed to write data to key.pem: %s", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("error closing key.pem: %s", err)
	}
	log.Print("wrote key.pem\n")
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}