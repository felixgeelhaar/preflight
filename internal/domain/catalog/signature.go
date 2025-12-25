package catalog

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Signature errors.
var (
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrSignatureExpired   = errors.New("signature expired")
	ErrUntrustedPublisher = errors.New("untrusted publisher")
	ErrNoSignature        = errors.New("no signature found")
)

// SignatureType represents the type of cryptographic signature.
type SignatureType string

// SignatureType constants.
const (
	SignatureTypeGPG      SignatureType = "gpg"
	SignatureTypeSSH      SignatureType = "ssh"
	SignatureTypeSigstore SignatureType = "sigstore"
)

// TrustLevel represents the trust level of a catalog.
type TrustLevel string

// TrustLevel constants.
const (
	TrustLevelBuiltin   TrustLevel = "builtin"
	TrustLevelVerified  TrustLevel = "verified"
	TrustLevelCommunity TrustLevel = "community"
	TrustLevelUntrusted TrustLevel = "untrusted"
)

// TrustLevelFromString parses a trust level from a string.
func TrustLevelFromString(s string) (TrustLevel, error) {
	switch strings.ToLower(s) {
	case "builtin":
		return TrustLevelBuiltin, nil
	case "verified":
		return TrustLevelVerified, nil
	case "community":
		return TrustLevelCommunity, nil
	case "untrusted":
		return TrustLevelUntrusted, nil
	default:
		return "", fmt.Errorf("unknown trust level: %s", s)
	}
}

// IsAtLeast returns true if this trust level is at least as trusted as other.
func (t TrustLevel) IsAtLeast(other TrustLevel) bool {
	levels := map[TrustLevel]int{
		TrustLevelBuiltin:   4,
		TrustLevelVerified:  3,
		TrustLevelCommunity: 2,
		TrustLevelUntrusted: 1,
	}
	return levels[t] >= levels[other]
}

// Signature represents a cryptographic signature on a manifest.
type Signature struct {
	signatureType SignatureType
	keyID         string
	value         []byte
	createdAt     time.Time
	expiresAt     time.Time
	publisher     Publisher
}

// NewSignature creates a new signature.
func NewSignature(sigType SignatureType, keyID string, value []byte, publisher Publisher) Signature {
	return Signature{
		signatureType: sigType,
		keyID:         keyID,
		value:         value,
		createdAt:     time.Now(),
		publisher:     publisher,
	}
}

// Type returns the signature type.
func (s Signature) Type() SignatureType {
	return s.signatureType
}

// KeyID returns the key identifier.
func (s Signature) KeyID() string {
	return s.keyID
}

// Value returns the raw signature bytes.
func (s Signature) Value() []byte {
	result := make([]byte, len(s.value))
	copy(result, s.value)
	return result
}

// CreatedAt returns when the signature was created.
func (s Signature) CreatedAt() time.Time {
	return s.createdAt
}

// ExpiresAt returns when the signature expires.
func (s Signature) ExpiresAt() time.Time {
	return s.expiresAt
}

// IsExpired returns true if the signature has expired.
func (s Signature) IsExpired() bool {
	if s.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(s.expiresAt)
}

// Publisher returns the publisher info.
func (s Signature) Publisher() Publisher {
	return s.publisher
}

// IsZero returns true if the signature is empty.
func (s Signature) IsZero() bool {
	return s.signatureType == "" && s.keyID == "" && len(s.value) == 0
}

// Publisher represents a catalog publisher.
type Publisher struct {
	name      string
	email     string
	keyID     string
	keyType   SignatureType
	createdAt time.Time
}

// NewPublisher creates a new publisher.
func NewPublisher(name, email, keyID string, keyType SignatureType) Publisher {
	return Publisher{
		name:      name,
		email:     email,
		keyID:     keyID,
		keyType:   keyType,
		createdAt: time.Now(),
	}
}

// Name returns the publisher name.
func (p Publisher) Name() string {
	return p.name
}

// Email returns the publisher email.
func (p Publisher) Email() string {
	return p.email
}

// KeyID returns the publisher's key ID.
func (p Publisher) KeyID() string {
	return p.keyID
}

// KeyType returns the type of key used.
func (p Publisher) KeyType() SignatureType {
	return p.keyType
}

// CreatedAt returns when the publisher was added.
func (p Publisher) CreatedAt() time.Time {
	return p.createdAt
}

// String returns a string representation.
func (p Publisher) String() string {
	if p.email != "" {
		return fmt.Sprintf("%s <%s>", p.name, p.email)
	}
	return p.name
}

// IsZero returns true if the publisher is empty.
func (p Publisher) IsZero() bool {
	return p.name == "" && p.email == "" && p.keyID == ""
}

// TrustedKey represents a trusted public key.
type TrustedKey struct {
	keyID       string
	keyType     SignatureType
	publicKey   crypto.PublicKey
	fingerprint string
	publisher   Publisher
	trustLevel  TrustLevel
	addedAt     time.Time
	expiresAt   time.Time
	comment     string
}

// NewTrustedKey creates a new trusted key.
func NewTrustedKey(keyID string, keyType SignatureType, publicKey crypto.PublicKey, publisher Publisher) *TrustedKey {
	return &TrustedKey{
		keyID:      keyID,
		keyType:    keyType,
		publicKey:  publicKey,
		publisher:  publisher,
		trustLevel: TrustLevelCommunity,
		addedAt:    time.Now(),
	}
}

// KeyID returns the key identifier.
func (k *TrustedKey) KeyID() string {
	return k.keyID
}

// KeyType returns the key type.
func (k *TrustedKey) KeyType() SignatureType {
	return k.keyType
}

// PublicKey returns the public key.
func (k *TrustedKey) PublicKey() crypto.PublicKey {
	return k.publicKey
}

// Fingerprint returns the key fingerprint.
func (k *TrustedKey) Fingerprint() string {
	return k.fingerprint
}

// Publisher returns the publisher info.
func (k *TrustedKey) Publisher() Publisher {
	return k.publisher
}

// TrustLevel returns the trust level.
func (k *TrustedKey) TrustLevel() TrustLevel {
	return k.trustLevel
}

// SetTrustLevel sets the trust level.
func (k *TrustedKey) SetTrustLevel(level TrustLevel) {
	k.trustLevel = level
}

// AddedAt returns when the key was added.
func (k *TrustedKey) AddedAt() time.Time {
	return k.addedAt
}

// ExpiresAt returns when the key expires.
func (k *TrustedKey) ExpiresAt() time.Time {
	return k.expiresAt
}

// IsExpired returns true if the key has expired.
func (k *TrustedKey) IsExpired() bool {
	if k.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(k.expiresAt)
}

// Comment returns the key comment.
func (k *TrustedKey) Comment() string {
	return k.comment
}

// SetComment sets the key comment.
func (k *TrustedKey) SetComment(comment string) {
	k.comment = comment
}

// SetFingerprint sets the key fingerprint.
func (k *TrustedKey) SetFingerprint(fingerprint string) {
	k.fingerprint = fingerprint
}

// SetExpiresAt sets the expiration time.
func (k *TrustedKey) SetExpiresAt(t time.Time) {
	k.expiresAt = t
}

// Verifier verifies signatures on catalog manifests.
type Verifier interface {
	// Verify checks that the signature is valid for the given content.
	Verify(content []byte, signature Signature) error

	// SupportsType returns true if this verifier supports the signature type.
	SupportsType(sigType SignatureType) bool
}

// ED25519Verifier verifies ED25519 signatures.
type ED25519Verifier struct {
	keys map[string]ed25519.PublicKey
}

// NewED25519Verifier creates a new ED25519 verifier.
func NewED25519Verifier() *ED25519Verifier {
	return &ED25519Verifier{
		keys: make(map[string]ed25519.PublicKey),
	}
}

// AddKey adds a public key.
func (v *ED25519Verifier) AddKey(keyID string, publicKey ed25519.PublicKey) {
	v.keys[keyID] = publicKey
}

// Verify verifies a signature.
func (v *ED25519Verifier) Verify(content []byte, signature Signature) error {
	if signature.IsExpired() {
		return ErrSignatureExpired
	}

	key, ok := v.keys[signature.KeyID()]
	if !ok {
		return fmt.Errorf("%w: key %s not found", ErrUntrustedPublisher, signature.KeyID())
	}

	// Hash the content
	hash := sha256.Sum256(content)

	// Verify the signature
	if !ed25519.Verify(key, hash[:], signature.Value()) {
		return ErrInvalidSignature
	}

	return nil
}

// SupportsType returns true for SSH signature type.
func (v *ED25519Verifier) SupportsType(sigType SignatureType) bool {
	return sigType == SignatureTypeSSH
}

// ComputeKeyFingerprint computes the fingerprint of a public key.
func ComputeKeyFingerprint(publicKey []byte) string {
	hash := sha256.Sum256(publicKey)
	return "SHA256:" + hex.EncodeToString(hash[:])
}

// SignedManifest represents a manifest with its signature.
type SignedManifest struct {
	manifest  Manifest
	signature Signature
}

// NewSignedManifest creates a new signed manifest.
func NewSignedManifest(manifest Manifest, signature Signature) SignedManifest {
	return SignedManifest{
		manifest:  manifest,
		signature: signature,
	}
}

// Manifest returns the manifest.
func (sm SignedManifest) Manifest() Manifest {
	return sm.manifest
}

// Signature returns the signature.
func (sm SignedManifest) Signature() Signature {
	return sm.signature
}

// IsSigned returns true if the manifest has a signature.
func (sm SignedManifest) IsSigned() bool {
	return !sm.signature.IsZero()
}

// VerificationResult contains the result of signature verification.
type VerificationResult struct {
	Verified   bool
	TrustLevel TrustLevel
	Publisher  Publisher
	KeyID      string
	Error      error
}

// NewVerificationResult creates a new verification result.
func NewVerificationResult(verified bool, trustLevel TrustLevel, publisher Publisher, keyID string) VerificationResult {
	return VerificationResult{
		Verified:   verified,
		TrustLevel: trustLevel,
		Publisher:  publisher,
		KeyID:      keyID,
	}
}

// WithError adds an error to the result.
func (vr VerificationResult) WithError(err error) VerificationResult {
	vr.Error = err
	return vr
}
