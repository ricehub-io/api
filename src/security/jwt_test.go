package security

import (
	"os"
	"testing"
)

const accessPrivPEM = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgGSpGIiazFUSDONm0
XupELbMQbBnSmgocwcx+o0uTIWihRANCAATMogo6VBDQanJ+X2ZZjbn1V1+UN3re
WRYdG2kLyfjxERaKQJhiuPBUCN+itdyjXbrZNDC+Jf4SQa8fpdxy2X3P
-----END PRIVATE KEY-----

`
const accessPubPEM = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEzKIKOlQQ0Gpyfl9mWY259VdflDd6
3lkWHRtpC8n48REWikCYYrjwVAjforXco1262TQwviX+EkGvH6Xcctl9zw==
-----END PUBLIC KEY-----
`

const refreshPrivPEM = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgIeVCwhYdKkOuPW5M
HubmLnEL+HX90x/TkPUvyV2vvMqhRANCAASzMGtrh1E7zxxGsrbUWDjKAPIfUSBy
/U1Gjm1CaFza4QBuy8qR3h08njRT/IFSB4PH6SP0qbAWJhZxvv4sWK0s
-----END PRIVATE KEY-----
`

const refreshPubPEM = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEszBra4dRO88cRrK21Fg4ygDyH1Eg
cv1NRo5tQmhc2uEAbsvKkd4dPJ40U/yBUgeDx+kj9KmwFiYWcb7+LFitLA==
-----END PUBLIC KEY-----
`

// writePEM writes a provided content string to a temporary '.pem' file and returns its path.
func writePEM(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "*.pem")
	if err != nil {
		t.Fatalf("failed to create a temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	f.Close()
	return f.Name()
}

// #################################################
// ############### loadECPrivateKey ################
// #################################################
func TestLoadECPrivateKey_ValidPEM(t *testing.T) {
	path := writePEM(t, accessPrivPEM)

	key, err := loadECPrivateKey(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if key == nil {
		t.Fatal("expected a non-nil key")
	}
}
