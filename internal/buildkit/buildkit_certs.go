/*
 * Copyright 2024 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Immutos Community Edition License, Version 1.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *    http://immutos.com/licenses/LICENSE-1.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package buildkit

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	validFor    = 24 * time.Hour
	graceWindow = time.Hour
)

func refreshCertificates(certsDir string) (rotated bool, err error) {
	// Only generate certificates if they do not already exist or are expired.
	if _, err := os.Stat(filepath.Join(certsDir, "ca.pem")); err == nil {
		caCertPEM, err := os.ReadFile(filepath.Join(certsDir, "ca.pem"))
		if err != nil {
			return false, fmt.Errorf("failed to read BuildKit certificate: %w", err)
		}

		caCertBlock, _ := pem.Decode(caCertPEM)
		caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
		if err != nil {
			return false, fmt.Errorf("failed to parse BuildKit certificate: %w", err)
		}

		if time.Now().Add(graceWindow).Before(caCert.NotAfter) {
			return false, nil
		}
	}

	// Generate new certificates.
	if err := generateCA(certsDir); err != nil {
		return true, fmt.Errorf("failed to generate self-signed CA certificate: %w", err)
	}

	if err := generateCert(certsDir, "buildkitd", false); err != nil {
		return true, fmt.Errorf("failed to generate BuildKit server certificate: %w", err)
	}

	if err := generateCert(certsDir, "immutos", true); err != nil {
		return true, fmt.Errorf("failed to generate immutos client certificate: %w", err)
	}

	return true, nil
}

// generateCA createss a new self-signed CA certificate.
func generateCA(certsDir string) error {
	caPubKey, caPrivKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "BuildKit CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validFor),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, caPubKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes})
	if err := os.WriteFile(filepath.Join(certsDir, "ca.pem"), caCertPEM, 0o644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	marshalledCAKey, err := x509.MarshalPKCS8PrivateKey(caPrivKey)
	if err != nil {
		return fmt.Errorf("failed to marshal CA private key: %w", err)
	}

	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: marshalledCAKey})
	if err := os.WriteFile(filepath.Join(certsDir, "ca-key.pem"), caKeyPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	return nil
}

// generateCert generates a new certificate signed by the CA.
func generateCert(certsDir string, name string, client bool) error {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Pick a large random number to use as the serial number.
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(validFor),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		DNSNames:  []string{name},
	}

	if client {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	} else {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}

	caCertPEM, err := os.ReadFile(filepath.Join(certsDir, "ca.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caKeyPEM, err := os.ReadFile(filepath.Join(certsDir, "ca-key.pem"))
	if err != nil {
		return fmt.Errorf("failed to read CA key: %w", err)
	}

	caCertBlock, _ := pem.Decode(caCertPEM)
	caKeyBlock, _ := pem.Decode(caKeyPEM)

	ca, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	caPrivKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA key: %w", err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, pubKey, caPrivKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err := os.WriteFile(filepath.Join(certsDir, fmt.Sprintf("%s.pem", name)), certPEM, 0o644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	marshalledKey, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: marshalledKey})
	if err := os.WriteFile(filepath.Join(certsDir, fmt.Sprintf("%s-key.pem", name)), keyPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}
