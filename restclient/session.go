// SPDX-License-Identifier: LicenseRef-KSAL-1.0

package restclient

import (
	"crypto/md5"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// buildLoginResponse computes the challenge-response for login_sid.lua.
// PBKDF2 (FRITZ!OS 7.24+): challenge "2$<iter1>$<salt1>$<iter2>$<salt2>",
// response "<salt2>$<hash2>" where
//
//	static = PBKDF2-HMAC-SHA256(password, salt1, iter1, 32)
//	hash2  = PBKDF2-HMAC-SHA256(static, salt2, iter2, 32)
//
// Legacy: response "<challenge>-<md5(utf16le(challenge+'-'+password))>".
func buildLoginResponse(challenge, password string) (string, error) {
	if !strings.Contains(challenge, "$") {
		return challenge + "-" + md5U16LE(challenge+"-"+password), nil
	}
	parts := strings.Split(challenge, "$")
	if len(parts) != 5 {
		return "", fmt.Errorf("unexpected challenge format: %q", challenge)
	}
	iter1, err1 := strconv.Atoi(parts[1])
	iter2, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return "", fmt.Errorf("bad challenge iterations: %v %v", err1, err2)
	}
	salt1, err3 := hex.DecodeString(parts[2])
	salt2, err4 := hex.DecodeString(parts[4])
	if err3 != nil || err4 != nil {
		return "", fmt.Errorf("bad challenge salt: %v %v", err3, err4)
	}
	static, err := pbkdf2.Key(sha256.New, password, salt1, iter1, 32)
	if err != nil {
		return "", err
	}
	dynamic, err := pbkdf2.Key(sha256.New, string(static), salt2, iter2, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s$%s", parts[4], hex.EncodeToString(dynamic)), nil
}

// md5U16LE hashes UTF-16LE encoded input (legacy pre-7.24 login).
func md5U16LE(s string) string {
	enc := make([]byte, 0, len(s)*2)
	for _, r := range []rune(s) {
		if r > 0xFFFF {
			r = 0xFFFD
		}
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, uint16(r))
		enc = append(enc, b...)
	}
	sum := md5.Sum(enc)
	return hex.EncodeToString(sum[:])
}

type errUnauthorized struct{ op string }

func (e errUnauthorized) Error() string {
	return fmt.Sprintf("unauthorized (401/403) during %s", e.op)
}

// isUnauthorized reports whether err originated from a 401/403 REST response.
func isUnauthorized(err error) bool {
	var eu errUnauthorized
	return errors.As(err, &eu)
}

// sessionInfo is the XML result of login_sid.lua.
type sessionInfo struct {
	SID       string `xml:"SID"`
	Challenge string `xml:"Challenge"`
	BlockTime int    `xml:"BlockTime"`
}
