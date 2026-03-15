package ok

import (
	"crypto/tls"

	"github.com/twmb/tlscfg"
)

// TLSConfig holds paths for TLS certificate configuration.
type TLSConfig struct {
	// CertFile is the path to the client certificate file (PEM).
	CertFile string `cfg:"cert_file" json:"cert_file,omitempty"`
	// KeyFile is the path to the client private key file (PEM).
	KeyFile string `cfg:"key_file" json:"key_file,omitempty"`
	// CAFile is the path to the CA certificate file (PEM) for server verification.
	CAFile string `cfg:"ca_file" json:"ca_file,omitempty"`
}

// Generate creates a *tls.Config from the TLSConfig paths.
// It uses the system certificate pool and adds the CA file if specified.
// Client certificates are loaded if both CertFile and KeyFile are set.
func (t TLSConfig) Generate() (*tls.Config, error) {
	var opts []tlscfg.Opt

	if t.CAFile != "" {
		opts = append(opts, tlscfg.MaybeWithDiskCA(t.CAFile, tlscfg.ForClient))
	}

	if t.CertFile != "" && t.KeyFile != "" {
		opts = append(opts, tlscfg.MaybeWithDiskKeyPair(t.CertFile, t.KeyFile))
	}

	return tlscfg.New(opts...)
}
