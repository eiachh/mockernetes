package auth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// NewTLSConfig creates a TLS configuration that enforces full mutual TLS (mTLS) authentication.
// It loads the server certificate/key for serving and the CA certificate to verify client certificates.
// This replaces Gin's RunTLS which accepts certs without verification.
// The certificate checks (loading, pool, extraction) are performed here as per requirements.
func NewTLSConfig(serverCert, serverKey, caCertPath string) (*tls.Config, error) {
	// Load server TLS certificate and key pair
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert/key: %w", err)
	}

	// Load CA cert for verifying client certificates (enables mTLS)
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	// Return TLS config with:
	// - Server cert
	// - RequireAndVerifyClientCert + ClientCAs for full mTLS (client cert must be present and valid wrt CA)
	// - Custom VerifyPeerCertificate to check cert details using ExtractUser (as instructed)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		// Additional cert check in auth.go: runs after CA verification
		VerifyPeerCertificate: verifyPeerCertificate,
	}, nil
}

// verifyPeerCertificate performs additional checks on the client certificate chain.
// This is where we "check the certs" using ExtractUser from the same auth.go file.
func verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// verifiedChains provided by Go's TLS stack after CA validation
	if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
		return fmt.Errorf("no verified client certificate chain provided")
	}

	// leaf cert is the client's cert
	leafCert := verifiedChains[0][0]

	// Use ExtractUser to get identity (implements the TODO)
	user := ExtractUser(leafCert)
	if user == "" {
		return fmt.Errorf("could not extract valid user identity from client certificate")
	}

	// In Kubernetes, CN=admin (from client.crt), O=system:masters
	// Here we accept any non-empty CN; extend as needed for authz
	// e.g., could check for "system:masters" in orgs

	return nil
}

// ExtractUser extracts the user identity from a TLS client certificate.
// Follows Kubernetes client cert auth convention: use CommonName (CN) as username.
// E.g., for client.crt with /CN=admin , returns "admin"
// Proper type: *x509.Certificate (updated from interface{} TODO)
func ExtractUser(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	// Could enhance: combine with Organization for groups, but keep minimal for mock
	return cert.Subject.CommonName
}

// VerifyTLSCertificates checks if the TLS certificates exist and are not expired.
// Returns an error with details if any check fails, nil on success.
// On failure, logs the specific issue. On success, does not log anything.
func VerifyTLSCertificates(serverCert, serverKey, caCertPath string) error {
	// Check if server certificate file exists
	if _, err := os.Stat(serverCert); os.IsNotExist(err) {
		return fmt.Errorf("server certificate file not found: %s", serverCert)
	}

	// Check if server key file exists
	if _, err := os.Stat(serverKey); os.IsNotExist(err) {
		return fmt.Errorf("server key file not found: %s", serverKey)
	}

	// Check if CA certificate file exists
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		return fmt.Errorf("CA certificate file not found: %s", caCertPath)
	}

	// Load and validate server certificate
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return fmt.Errorf("failed to load server certificate/key pair: %w", err)
	}

	// Parse the server certificate to check expiration
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Check if server certificate is expired
	now := time.Now()
	if now.Before(x509Cert.NotBefore) {
		return fmt.Errorf("server certificate is not yet valid (valid from: %s)", x509Cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(x509Cert.NotAfter) {
		return fmt.Errorf("server certificate has expired (expired on: %s)", x509Cert.NotAfter.Format(time.RFC3339))
	}

	// Load and validate CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate: invalid PEM format")
	}

	// Parse CA certificate to check expiration using PEM decoding
	block, _ := pem.Decode(caCert)
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate PEM block")
	}
	caX509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}
	if now.Before(caX509Cert.NotBefore) {
		return fmt.Errorf("CA certificate is not yet valid (valid from: %s)", caX509Cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(caX509Cert.NotAfter) {
		return fmt.Errorf("CA certificate has expired (expired on: %s)", caX509Cert.NotAfter.Format(time.RFC3339))
	}

	return nil
}
