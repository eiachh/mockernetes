package auth

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
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
