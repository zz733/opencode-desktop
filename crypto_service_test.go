package main

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
)

func TestNewCryptoService(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
	}{
		{"Normal key", "test-master-key"},
		{"Empty key", ""},
		{"Long key", strings.Repeat("a", 1000)},
		{"Unicode key", "测试密钥🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewCryptoService(tt.masterKey)
			if cs == nil {
				t.Error("NewCryptoService returned nil")
			}
			if len(cs.key) != 32 {
				t.Errorf("Expected key length 32, got %d", len(cs.key))
			}
		})
	}
}

func TestCryptoService_Encrypt_Decrypt(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name string
		data []byte
	}{
		{"Simple text", []byte("Hello, World!")},
		{"Empty data", []byte("")},
		{"Binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
		{"Large data", make([]byte, 10000)},
		{"Unicode text", []byte("测试数据 🔐 encryption")},
	}

	// Initialize large data test
	rand.Read(tests[3].data)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) == 0 && tt.name != "Empty data" {
				return // Skip empty data for non-empty test cases
			}

			// Test empty data separately
			if tt.name == "Empty data" {
				_, err := cs.Encrypt(tt.data)
				if err == nil {
					t.Error("Expected error for empty data, got nil")
				}
				return
			}

			// Encrypt
			encrypted, err := cs.Encrypt(tt.data)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Verify encrypted data is different from original
			if bytes.Equal(encrypted, tt.data) {
				t.Error("Encrypted data should be different from original")
			}

			// Decrypt
			decrypted, err := cs.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			// Verify decrypted data matches original
			if !bytes.Equal(decrypted, tt.data) {
				t.Error("Decrypted data doesn't match original")
			}
		})
	}
}

func TestCryptoService_EncryptString_DecryptString(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name string
		text string
	}{
		{"Simple text", "Hello, World!"},
		{"Empty string", ""},
		{"Unicode text", "测试字符串 🔐"},
		{"JSON data", `{"key": "value", "number": 123}`},
		{"Long text", strings.Repeat("Lorem ipsum ", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.text == "" && tt.name != "Empty string" {
				return
			}

			// Test empty string separately
			if tt.name == "Empty string" {
				_, err := cs.EncryptString(tt.text)
				if err == nil {
					t.Error("Expected error for empty string, got nil")
				}
				return
			}

			// Encrypt
			encrypted, err := cs.EncryptString(tt.text)
			if err != nil {
				t.Fatalf("EncryptString failed: %v", err)
			}

			// Verify encrypted string is different from original
			if encrypted == tt.text {
				t.Error("Encrypted string should be different from original")
			}

			// Decrypt
			decrypted, err := cs.DecryptString(encrypted)
			if err != nil {
				t.Fatalf("DecryptString failed: %v", err)
			}

			// Verify decrypted string matches original
			if decrypted != tt.text {
				t.Errorf("Decrypted string doesn't match original. Expected: %s, Got: %s", tt.text, decrypted)
			}
		})
	}
}

func TestCryptoService_EncryptWithPassword_DecryptWithPassword(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name     string
		data     []byte
		password string
	}{
		{"Normal case", []byte("sensitive data"), "strong-password"},
		{"Weak password", []byte("test data"), "123"},
		{"Unicode password", []byte("test data"), "密码🔐"},
		{"Long password", []byte("test data"), strings.Repeat("p", 1000)},
		{"Binary data", []byte{0x00, 0x01, 0xFF, 0xFE}, "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := cs.EncryptWithPassword(tt.data, tt.password)
			if err != nil {
				t.Fatalf("EncryptWithPassword failed: %v", err)
			}

			// Verify encrypted data is different from original
			if bytes.Equal(encrypted, tt.data) {
				t.Error("Encrypted data should be different from original")
			}

			// Decrypt with correct password
			decrypted, err := cs.DecryptWithPassword(encrypted, tt.password)
			if err != nil {
				t.Fatalf("DecryptWithPassword failed: %v", err)
			}

			// Verify decrypted data matches original
			if !bytes.Equal(decrypted, tt.data) {
				t.Error("Decrypted data doesn't match original")
			}

			// Try to decrypt with wrong password
			_, err = cs.DecryptWithPassword(encrypted, "wrong-password")
			if err == nil {
				t.Error("Expected error when decrypting with wrong password")
			}
		})
	}
}

func TestCryptoService_PasswordEncryption_ErrorCases(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	// Test empty data
	_, err := cs.EncryptWithPassword([]byte{}, "password")
	if err == nil {
		t.Error("Expected error for empty data")
	}

	// Test empty password
	_, err = cs.EncryptWithPassword([]byte("data"), "")
	if err == nil {
		t.Error("Expected error for empty password")
	}

	// Test decryption with empty data
	_, err = cs.DecryptWithPassword([]byte{}, "password")
	if err == nil {
		t.Error("Expected error for empty encrypted data")
	}

	// Test decryption with empty password
	_, err = cs.DecryptWithPassword([]byte("data"), "")
	if err == nil {
		t.Error("Expected error for empty password in decryption")
	}

	// Test decryption with too short data
	_, err = cs.DecryptWithPassword([]byte("short"), "password")
	if err == nil {
		t.Error("Expected error for too short encrypted data")
	}
}

func TestCryptoService_GenerateRandomKey(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name   string
		length int
		hasErr bool
	}{
		{"16 bytes", 16, false},
		{"32 bytes", 32, false},
		{"64 bytes", 64, false},
		{"Zero length", 0, true},
		{"Negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := cs.GenerateRandomKey(tt.length)

			if tt.hasErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateRandomKey failed: %v", err)
			}

			if len(key) != tt.length {
				t.Errorf("Expected key length %d, got %d", tt.length, len(key))
			}

			// Generate another key and verify they're different
			key2, err := cs.GenerateRandomKey(tt.length)
			if err != nil {
				t.Fatalf("Second GenerateRandomKey failed: %v", err)
			}

			if bytes.Equal(key, key2) {
				t.Error("Two random keys should be different")
			}
		})
	}
}

func TestCryptoService_GenerateRandomString(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name   string
		length int
		hasErr bool
	}{
		{"Short string", 8, false},
		{"Medium string", 32, false},
		{"Long string", 128, false},
		{"Zero length", 0, true},
		{"Negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, err := cs.GenerateRandomString(tt.length)

			if tt.hasErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateRandomString failed: %v", err)
			}

			if len(str) != tt.length {
				t.Errorf("Expected string length %d, got %d", tt.length, len(str))
			}

			// Generate another string and verify they're different
			str2, err := cs.GenerateRandomString(tt.length)
			if err != nil {
				t.Fatalf("Second GenerateRandomString failed: %v", err)
			}

			if str == str2 {
				t.Error("Two random strings should be different")
			}
		})
	}
}

func TestCryptoService_HashPassword_VerifyPassword(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	tests := []struct {
		name     string
		password string
	}{
		{"Simple password", "password123"},
		{"Complex password", "P@ssw0rd!@#$%^&*()"},
		{"Unicode password", "密码测试🔐"},
		{"Long password", strings.Repeat("a", 1000)},
		{"Empty password", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Hash password
			hash := cs.HashPassword(tt.password)
			if hash == "" {
				t.Error("Hash should not be empty")
			}

			// Verify correct password
			if !cs.VerifyPassword(tt.password, hash) {
				t.Error("Password verification failed for correct password")
			}

			// Verify wrong password
			if cs.VerifyPassword(tt.password+"wrong", hash) {
				t.Error("Password verification should fail for wrong password")
			}

			// Verify same password produces same hash
			hash2 := cs.HashPassword(tt.password)
			if hash != hash2 {
				t.Error("Same password should produce same hash")
			}
		})
	}
}

func TestCryptoService_SecureWipe(t *testing.T) {
	cs := NewCryptoService("test-master-key")

	// Create test data
	data := []byte("sensitive data that should be wiped")
	original := make([]byte, len(data))
	copy(original, data)

	// Verify data is initially correct
	if !bytes.Equal(data, original) {
		t.Error("Initial data copy failed")
	}

	// Wipe the data
	cs.SecureWipe(data)

	// Verify data is zeroed
	for i, b := range data {
		if b != 0 {
			t.Errorf("Byte at index %d not wiped: got %d, expected 0", i, b)
		}
	}

	// Verify original is unchanged
	if bytes.Equal(data, original) {
		t.Error("Data should be different from original after wiping")
	}
}

func TestCryptoService_EncryptionConsistency(t *testing.T) {
	// Test that different CryptoService instances with same key produce compatible results
	key := "consistent-test-key"
	cs1 := NewCryptoService(key)
	cs2 := NewCryptoService(key)

	testData := []byte("consistency test data")

	// Encrypt with first instance
	encrypted, err := cs1.Encrypt(testData)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Decrypt with second instance
	decrypted, err := cs2.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !bytes.Equal(testData, decrypted) {
		t.Error("Cross-instance encryption/decryption failed")
	}
}

func TestCryptoService_EncryptionUniqueness(t *testing.T) {
	cs := NewCryptoService("test-master-key")
	testData := []byte("test data for uniqueness")

	// Encrypt same data multiple times
	encrypted1, err := cs.Encrypt(testData)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	encrypted2, err := cs.Encrypt(testData)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	// Encrypted results should be different (due to random nonce)
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Multiple encryptions of same data should produce different results")
	}

	// But both should decrypt to same original data
	decrypted1, err := cs.Decrypt(encrypted1)
	if err != nil {
		t.Fatalf("First decryption failed: %v", err)
	}

	decrypted2, err := cs.Decrypt(encrypted2)
	if err != nil {
		t.Fatalf("Second decryption failed: %v", err)
	}

	if !bytes.Equal(testData, decrypted1) || !bytes.Equal(testData, decrypted2) {
		t.Error("Decrypted data doesn't match original")
	}
}

// Benchmark tests
func BenchmarkCryptoService_Encrypt(b *testing.B) {
	cs := NewCryptoService("benchmark-key")
	data := make([]byte, 1024) // 1KB data
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cs.Encrypt(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCryptoService_Decrypt(b *testing.B) {
	cs := NewCryptoService("benchmark-key")
	data := make([]byte, 1024) // 1KB data
	rand.Read(data)

	encrypted, err := cs.Encrypt(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cs.Decrypt(encrypted)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCryptoService_EncryptWithPassword(b *testing.B) {
	cs := NewCryptoService("benchmark-key")
	data := make([]byte, 1024) // 1KB data
	rand.Read(data)
	password := "benchmark-password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cs.EncryptWithPassword(data, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}