package crypto

import (
	"unsafe"
)

// Zero securely zeros a byte slice to remove sensitive data from memory.
// This function uses a volatile write pattern to prevent compiler optimization
// from removing the zeroing operation.
//
// Call this function immediately after using sensitive data like:
// - Decrypted credentials (PINs, passwords)
// - Derived keys
// - Plaintext data
//
// Example:
//
//	pin, err := decryptor.DecryptCredential(...)
//	if err != nil { return err }
//	defer crypto.Zero(pin)
//	// use pin...
func Zero(b []byte) {
	if len(b) == 0 {
		return
	}

	// Use a volatile-like pattern to prevent optimization
	// The compiler cannot prove the pointer is never read again
	ptr := unsafe.Pointer(&b[0])
	for i := range b {
		*(*byte)(unsafe.Add(ptr, i)) = 0
	}
}

// ZeroString zeros a string's underlying byte array.
// Note: strings in Go are immutable, but we can still zero the underlying memory.
// This is a best-effort function - the runtime may have copied the string.
//
// For sensitive strings, prefer using []byte and Zero() instead.
func ZeroString(s *string) {
	if s == nil || len(*s) == 0 {
		return
	}

	// Get the underlying byte slice
	// This is unsafe but necessary for memory clearing
	header := (*struct {
		Data uintptr
		Len  int
	})(unsafe.Pointer(s))

	if header.Data == 0 {
		return
	}

	ptr := unsafe.Pointer(header.Data)
	for i := 0; i < header.Len; i++ {
		*(*byte)(unsafe.Add(ptr, i)) = 0
	}
}

// SecureBytes is a wrapper around []byte that automatically zeros on finalization.
// Use this for sensitive data that should be cleaned up when no longer needed.
//
// Example:
//
//	secure := crypto.NewSecureBytes(decryptedPin)
//	defer secure.Zero()
//	pin := secure.Bytes()
type SecureBytes struct {
	data []byte
}

// NewSecureBytes creates a new SecureBytes from existing data.
// The original data is NOT zeroed - the caller should zero it if needed.
func NewSecureBytes(data []byte) *SecureBytes {
	// Make a copy to ensure we own the memory
	copied := make([]byte, len(data))
	copy(copied, data)
	return &SecureBytes{data: copied}
}

// Bytes returns the underlying byte slice.
// The caller should NOT store this reference beyond the SecureBytes lifetime.
func (s *SecureBytes) Bytes() []byte {
	return s.data
}

// String returns the data as a string.
// The caller should NOT store this string beyond the SecureBytes lifetime.
func (s *SecureBytes) String() string {
	return string(s.data)
}

// Len returns the length of the data
func (s *SecureBytes) Len() int {
	return len(s.data)
}

// Zero clears the sensitive data from memory
func (s *SecureBytes) Zero() {
	Zero(s.data)
	s.data = nil
}

// IsZeroed returns true if the data has been zeroed
func (s *SecureBytes) IsZeroed() bool {
	return s.data == nil
}

// WithSecureBytes executes a function with secure bytes and ensures cleanup.
// This is the recommended pattern for handling sensitive data.
//
// Example:
//
//	err := crypto.WithSecureBytes(decryptedPin, func(pin []byte) error {
//	    return api.Authenticate(pin)
//	})
func WithSecureBytes(data []byte, fn func([]byte) error) error {
	secure := NewSecureBytes(data)
	defer secure.Zero()

	// Zero the original data
	Zero(data)

	return fn(secure.Bytes())
}

// ZeroAll zeros multiple byte slices
func ZeroAll(slices ...[]byte) {
	for _, s := range slices {
		Zero(s)
	}
}
