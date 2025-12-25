package catalog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSigstoreVerifierConfig(t *testing.T) {
	t.Parallel()

	config := DefaultSigstoreVerifierConfig()

	assert.False(t, config.AllowExpired)
	assert.True(t, config.VerifyTimestamp)
	assert.Len(t, config.TrustedIdentities, 4) // GitHub, GitLab, Google, Microsoft
}

func TestSigstoreVerifier_SupportsType(t *testing.T) {
	t.Parallel()

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())

	assert.True(t, verifier.SupportsType(SignatureTypeSigstore))
	assert.False(t, verifier.SupportsType(SignatureTypeGPG))
	assert.False(t, verifier.SupportsType(SignatureTypeSSH))
}

func TestSigstoreVerifier_AddTrustedIdentity(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{}
	verifier := NewSigstoreVerifier(config)

	assert.Empty(t, verifier.config.TrustedIdentities)

	verifier.AddTrustedIdentity(SigstoreIdentity{
		IssuerRegexp:  `^https://custom-issuer\.com$`,
		SubjectRegexp: `.+@custom\.com`,
	})

	assert.Len(t, verifier.config.TrustedIdentities, 1)
}

func TestSigstoreVerifier_Verify_WrongSignatureType(t *testing.T) {
	t.Parallel()

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())
	sig := NewSignature(SignatureTypeGPG, "key-id", []byte("test"), Publisher{})

	err := verifier.Verify([]byte("content"), sig)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestSigstoreVerifier_Verify_NoCertificate(t *testing.T) {
	t.Parallel()

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())

	sigData := SigstoreSignature{
		Signature:   base64.StdEncoding.EncodeToString([]byte("sig")),
		Certificate: "",
	}
	sigBytes, _ := json.Marshal(sigData)
	sig := NewSignature(SignatureTypeSigstore, "", sigBytes, Publisher{})

	err := verifier.Verify([]byte("content"), sig)

	assert.ErrorIs(t, err, ErrSigstoreNoCertificate)
}

func TestSigstoreVerifier_Verify_InvalidCertificate(t *testing.T) {
	t.Parallel()

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())

	sigData := SigstoreSignature{
		Signature:   base64.StdEncoding.EncodeToString([]byte("sig")),
		Certificate: base64.StdEncoding.EncodeToString([]byte("not a certificate")),
	}
	sigBytes, _ := json.Marshal(sigData)
	sig := NewSignature(SignatureTypeSigstore, "", sigBytes, Publisher{})

	err := verifier.Verify([]byte("content"), sig)

	assert.ErrorIs(t, err, ErrSigstoreInvalidCertificate)
}

func TestSigstoreVerifier_Verify_InvalidJSON(t *testing.T) {
	t.Parallel()

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())
	sig := NewSignature(SignatureTypeSigstore, "", []byte("not json"), Publisher{})

	err := verifier.Verify([]byte("content"), sig)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse sigstore signature")
}

func TestSigstoreVerifier_verifyIdentity_Match(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			{
				IssuerRegexp:  `^https://token\.actions\.githubusercontent\.com$`,
				SubjectRegexp: `^https://github\.com/myorg/.+$`,
			},
		},
	}
	verifier := NewSigstoreVerifier(config)

	err := verifier.verifyIdentity(
		"https://token.actions.githubusercontent.com",
		"https://github.com/myorg/myrepo/.github/workflows/release.yml@refs/tags/v1.0.0",
	)

	assert.NoError(t, err)
}

func TestSigstoreVerifier_verifyIdentity_IssuerMismatch(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			{
				IssuerRegexp:  `^https://token\.actions\.githubusercontent\.com$`,
				SubjectRegexp: `.+`,
			},
		},
	}
	verifier := NewSigstoreVerifier(config)

	err := verifier.verifyIdentity(
		"https://evil-issuer.com",
		"https://github.com/myorg/myrepo",
	)

	assert.ErrorIs(t, err, ErrSigstoreIdentityMismatch)
}

func TestSigstoreVerifier_verifyIdentity_SubjectMismatch(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			{
				IssuerRegexp:  `^https://token\.actions\.githubusercontent\.com$`,
				SubjectRegexp: `^https://github\.com/trustedorg/.+$`,
			},
		},
	}
	verifier := NewSigstoreVerifier(config)

	err := verifier.verifyIdentity(
		"https://token.actions.githubusercontent.com",
		"https://github.com/untrustedorg/repo",
	)

	assert.ErrorIs(t, err, ErrSigstoreIdentityMismatch)
}

func TestSigstoreVerifier_verifyIdentity_InvalidRegex(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			{
				IssuerRegexp:  `[invalid`,
				SubjectRegexp: `.+`,
			},
		},
	}
	verifier := NewSigstoreVerifier(config)

	err := verifier.verifyIdentity("issuer", "subject")

	assert.ErrorIs(t, err, ErrSigstoreIdentityMismatch)
}

func TestSigstoreVerifier_verifyIdentity_MultipleIdentities(t *testing.T) {
	t.Parallel()

	config := SigstoreVerifierConfig{
		TrustedIdentities: []SigstoreIdentity{
			{
				IssuerRegexp:  `^https://gitlab\.com$`,
				SubjectRegexp: `.+`,
			},
			{
				IssuerRegexp:  `^https://accounts\.google\.com$`,
				SubjectRegexp: `.+@company\.com`,
			},
		},
	}
	verifier := NewSigstoreVerifier(config)

	// First identity doesn't match, but second does
	err := verifier.verifyIdentity(
		"https://accounts.google.com",
		"user@company.com",
	)

	assert.NoError(t, err)
}

func TestEqualOID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        []int
		b        []int
		expected bool
	}{
		{"equal", []int{1, 2, 3}, []int{1, 2, 3}, true},
		{"different length", []int{1, 2}, []int{1, 2, 3}, false},
		{"different values", []int{1, 2, 3}, []int{1, 2, 4}, false},
		{"empty", []int{}, []int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, equalOID(tt.a, tt.b))
		})
	}
}

func TestParseCertificateFromPEM(t *testing.T) {
	t.Parallel()

	// Generate a test certificate
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	t.Run("raw DER", func(t *testing.T) {
		t.Parallel()
		cert, err := parseCertificateFromPEM(certDER)
		require.NoError(t, err)
		assert.Equal(t, "test", cert.Subject.CommonName)
	})

	t.Run("PEM format", func(t *testing.T) {
		t.Parallel()
		pemData := "-----BEGIN CERTIFICATE-----\n" +
			base64.StdEncoding.EncodeToString(certDER) +
			"\n-----END CERTIFICATE-----"

		cert, err := parseCertificateFromPEM([]byte(pemData))
		require.NoError(t, err)
		assert.Equal(t, "test", cert.Subject.CommonName)
	})

	t.Run("base64 encoded DER", func(t *testing.T) {
		t.Parallel()
		b64 := base64.StdEncoding.EncodeToString(certDER)

		cert, err := parseCertificateFromPEM([]byte(b64))
		require.NoError(t, err)
		assert.Equal(t, "test", cert.Subject.CommonName)
	})
}

func TestVerifyECDSASignature(t *testing.T) {
	t.Parallel()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	content := []byte("test content")
	hash := sha256.Sum256(content)

	signature, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	require.NoError(t, err)

	t.Run("valid signature", func(t *testing.T) {
		t.Parallel()
		err := verifyECDSASignature(&privKey.PublicKey, hash[:], signature)
		assert.NoError(t, err)
	})

	t.Run("invalid signature", func(t *testing.T) {
		t.Parallel()
		badSig := []byte("invalid signature")
		err := verifyECDSASignature(&privKey.PublicKey, hash[:], badSig)
		assert.ErrorIs(t, err, ErrInvalidSignature)
	})

	t.Run("wrong hash", func(t *testing.T) {
		t.Parallel()
		wrongHash := sha256.Sum256([]byte("different content"))
		err := verifyECDSASignature(&privKey.PublicKey, wrongHash[:], signature)
		assert.ErrorIs(t, err, ErrInvalidSignature)
	})

	t.Run("non-ECDSA key", func(t *testing.T) {
		t.Parallel()
		err := verifyECDSASignature("not a key", hash[:], signature)
		assert.ErrorIs(t, err, ErrInvalidSignature)
	})
}

func TestSigstorePublisher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		issuer          string
		subject         string
		expectedName    string
		expectedEmail   string
		expectedKeyID   string
		expectedKeyType SignatureType
	}{
		{
			name:            "email subject",
			issuer:          "https://accounts.google.com",
			subject:         "user@example.com",
			expectedName:    "user",
			expectedEmail:   "user@example.com",
			expectedKeyID:   "https://accounts.google.com",
			expectedKeyType: SignatureTypeSigstore,
		},
		{
			name:            "github actions subject",
			issuer:          "https://token.actions.githubusercontent.com",
			subject:         "https://github.com/myorg/myrepo/.github/workflows/release.yml@refs/tags/v1.0.0",
			expectedName:    "myorg",
			expectedEmail:   "",
			expectedKeyID:   "https://token.actions.githubusercontent.com",
			expectedKeyType: SignatureTypeSigstore,
		},
		{
			name:            "non-email non-github",
			issuer:          "https://gitlab.com",
			subject:         "project:12345",
			expectedName:    "project:12345",
			expectedEmail:   "",
			expectedKeyID:   "https://gitlab.com",
			expectedKeyType: SignatureTypeSigstore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pub := SigstorePublisher(tt.issuer, tt.subject)
			assert.Equal(t, tt.expectedName, pub.Name())
			assert.Equal(t, tt.expectedEmail, pub.Email())
			assert.Equal(t, tt.expectedKeyID, pub.KeyID())
			assert.Equal(t, tt.expectedKeyType, pub.KeyType())
		})
	}
}

func TestExtractOIDCClaims(t *testing.T) {
	t.Parallel()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	t.Run("with email SAN", func(t *testing.T) {
		t.Parallel()

		// Create cert with email SAN and OIDC issuer extension
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test",
			},
			EmailAddresses: []string{"user@example.com"},
			NotBefore:      time.Now(),
			NotAfter:       time.Now().Add(time.Hour),
			ExtraExtensions: []pkix.Extension{
				{
					Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1},
					Value: []byte("https://accounts.google.com"),
				},
			},
		}

		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(certDER)
		require.NoError(t, err)

		issuer, subject, err := extractOIDCClaims(cert)
		require.NoError(t, err)
		assert.Equal(t, "https://accounts.google.com", issuer)
		assert.Equal(t, "user@example.com", subject)
	})

	t.Run("with CommonName subject", func(t *testing.T) {
		t.Parallel()

		// For self-signed certs, the Issuer is set from the Subject of the parent (template)
		// So we need the Subject.CommonName to serve as both the issuer (fallback) and subject
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName: "test-identity",
			},
			NotBefore: time.Now(),
			NotAfter:  time.Now().Add(time.Hour),
		}

		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(certDER)
		require.NoError(t, err)

		issuer, subject, err := extractOIDCClaims(cert)
		require.NoError(t, err)
		// Self-signed cert: Issuer.CommonName = Subject.CommonName
		assert.Equal(t, "test-identity", issuer)
		assert.Equal(t, "test-identity", subject)
	})

	t.Run("no issuer fallback", func(t *testing.T) {
		t.Parallel()

		// Certificate with empty CommonName - can't extract issuer
		template := &x509.Certificate{
			SerialNumber:   big.NewInt(1),
			EmailAddresses: []string{"user@example.com"},
			NotBefore:      time.Now(),
			NotAfter:       time.Now().Add(time.Hour),
		}

		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
		require.NoError(t, err)

		cert, err := x509.ParseCertificate(certDER)
		require.NoError(t, err)

		_, _, err = extractOIDCClaims(cert)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not extract OIDC issuer")
	})
}

func TestSigstoreVerifier_Verify_ExpiredCertificate(t *testing.T) {
	t.Parallel()

	// Create an expired certificate
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		EmailAddresses: []string{"user@example.com"},
		NotBefore:      time.Now().Add(-2 * time.Hour),
		NotAfter:       time.Now().Add(-1 * time.Hour), // Expired
		ExtraExtensions: []pkix.Extension{
			{
				Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1},
				Value: []byte("https://accounts.google.com"),
			},
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	certPEM := "-----BEGIN CERTIFICATE-----\n" +
		base64.StdEncoding.EncodeToString(certDER) +
		"\n-----END CERTIFICATE-----"

	content := []byte("test content")
	hash := sha256.Sum256(content)
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	require.NoError(t, err)

	sigData := SigstoreSignature{
		Signature:   base64.StdEncoding.EncodeToString(signature),
		Certificate: base64.StdEncoding.EncodeToString([]byte(certPEM)),
	}
	sigBytes, _ := json.Marshal(sigData)
	sig := NewSignature(SignatureTypeSigstore, "", sigBytes, Publisher{})

	t.Run("reject expired by default", func(t *testing.T) {
		t.Parallel()
		verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())
		err := verifier.Verify(content, sig)
		assert.ErrorIs(t, err, ErrSigstoreCertExpired)
	})

	t.Run("allow expired with config", func(t *testing.T) {
		t.Parallel()
		config := DefaultSigstoreVerifierConfig()
		config.AllowExpired = true
		verifier := NewSigstoreVerifier(config)
		// Will fail at identity verification since google.com isn't in trusted list
		err := verifier.Verify(content, sig)
		// Should pass the expiry check but may fail on identity
		assert.NotErrorIs(t, err, ErrSigstoreCertExpired)
	})
}

func TestSigstoreVerifier_Verify_NotYetValid(t *testing.T) {
	t.Parallel()

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		EmailAddresses: []string{"user@example.com"},
		NotBefore:      time.Now().Add(1 * time.Hour), // Not yet valid
		NotAfter:       time.Now().Add(2 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	certPEM := "-----BEGIN CERTIFICATE-----\n" +
		base64.StdEncoding.EncodeToString(certDER) +
		"\n-----END CERTIFICATE-----"

	sigData := SigstoreSignature{
		Signature:   base64.StdEncoding.EncodeToString([]byte("sig")),
		Certificate: base64.StdEncoding.EncodeToString([]byte(certPEM)),
	}
	sigBytes, _ := json.Marshal(sigData)
	sig := NewSignature(SignatureTypeSigstore, "", sigBytes, Publisher{})

	verifier := NewSigstoreVerifier(DefaultSigstoreVerifierConfig())
	err = verifier.Verify([]byte("content"), sig)

	assert.ErrorIs(t, err, ErrSigstoreNotYetValid)
}

func TestSigstoreBundle(t *testing.T) {
	t.Parallel()

	bundle := SigstoreBundle{
		UUID:           "test-uuid",
		LogIndex:       12345,
		IntegratedTime: time.Now().Unix(),
	}

	assert.Equal(t, "test-uuid", bundle.UUID)
	assert.Equal(t, int64(12345), bundle.LogIndex)
	assert.NotZero(t, bundle.IntegratedTime)
}

func TestSigstoreSignature_JSON(t *testing.T) {
	t.Parallel()

	sig := SigstoreSignature{
		Signature:   "c2lnbmF0dXJl",
		Certificate: "Y2VydGlmaWNhdGU=",
		Bundle: &SigstoreBundle{
			UUID:           "uuid",
			LogIndex:       100,
			IntegratedTime: 1234567890,
		},
	}

	data, err := json.Marshal(sig)
	require.NoError(t, err)

	var parsed SigstoreSignature
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, sig.Signature, parsed.Signature)
	assert.Equal(t, sig.Certificate, parsed.Certificate)
	assert.NotNil(t, parsed.Bundle)
	assert.Equal(t, sig.Bundle.UUID, parsed.Bundle.UUID)
}
