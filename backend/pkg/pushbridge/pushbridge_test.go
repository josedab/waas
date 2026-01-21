package pushbridge

import (
	"testing"
)

func TestCredentialData_Validate_FCM(t *testing.T) {
	tests := []struct {
		name    string
		creds   CredentialData
		wantErr bool
	}{
		{
			name: "valid FCM with service account",
			creds: CredentialData{
				FCMProjectID:      "my-project",
				FCMServiceAccount: `{"type":"service_account"}`,
			},
			wantErr: false,
		},
		{
			name: "valid FCM with server key",
			creds: CredentialData{
				FCMProjectID:  "my-project",
				FCMServerKey:  "server-key-value",
			},
			wantErr: false,
		},
		{
			name:    "missing project ID",
			creds:   CredentialData{FCMServiceAccount: `{}`},
			wantErr: true,
		},
		{
			name:    "missing credentials",
			creds:   CredentialData{FCMProjectID: "my-project"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate(PlatformAndroid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCredentialData_Validate_APNs(t *testing.T) {
	tests := []struct {
		name    string
		creds   CredentialData
		wantErr bool
	}{
		{
			name: "valid APNs with private key",
			creds: CredentialData{
				APNsBundleID:   "com.example.app",
				APNsKeyID:      "ABCD1234",
				APNsTeamID:     "TEAM123",
				APNsPrivateKey: "-----BEGIN PRIVATE KEY-----...",
			},
			wantErr: false,
		},
		{
			name: "valid APNs with certificate",
			creds: CredentialData{
				APNsBundleID:    "com.example.app",
				APNsCertificate: "-----BEGIN CERTIFICATE-----...",
			},
			wantErr: false,
		},
		{
			name:    "missing bundle ID",
			creds:   CredentialData{APNsPrivateKey: "key"},
			wantErr: true,
		},
		{
			name: "private key without key ID",
			creds: CredentialData{
				APNsBundleID:   "com.example.app",
				APNsPrivateKey: "key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate(PlatformIOS)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCredentialData_Validate_WebPush(t *testing.T) {
	tests := []struct {
		name    string
		creds   CredentialData
		wantErr bool
	}{
		{
			name: "valid VAPID keys",
			creds: CredentialData{
				VAPIDPublicKey:  "BPublicKey...",
				VAPIDPrivateKey: "PrivateKey...",
			},
			wantErr: false,
		},
		{
			name:    "missing public key",
			creds:   CredentialData{VAPIDPrivateKey: "key"},
			wantErr: true,
		},
		{
			name:    "missing private key",
			creds:   CredentialData{VAPIDPublicKey: "key"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate(PlatformWeb)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCredentialData_Validate_Huawei(t *testing.T) {
	tests := []struct {
		name    string
		creds   CredentialData
		wantErr bool
	}{
		{
			name: "valid Huawei credentials",
			creds: CredentialData{
				HuaweiAppID:     "12345",
				HuaweiAppSecret: "secret123",
			},
			wantErr: false,
		},
		{
			name:    "missing app ID",
			creds:   CredentialData{HuaweiAppSecret: "secret"},
			wantErr: true,
		},
		{
			name:    "missing app secret",
			creds:   CredentialData{HuaweiAppID: "12345"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.Validate(PlatformHuawei)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlatformConstants(t *testing.T) {
	platforms := []Platform{PlatformIOS, PlatformAndroid, PlatformWeb, PlatformHuawei}
	
	for _, p := range platforms {
		if p == "" {
			t.Error("Platform constant should not be empty")
		}
	}
}

func TestDeviceStatus_Constants(t *testing.T) {
	statuses := []DeviceStatus{DeviceActive, DeviceInactive, DeviceUnregistered, DeviceSuspended}
	
	for _, s := range statuses {
		if s == "" {
			t.Error("DeviceStatus constant should not be empty")
		}
	}
}

func TestCreateCredentialsRequest_Validation(t *testing.T) {
	req := &CreateCredentialsRequest{
		Provider: PlatformAndroid,
		Name:     "Production FCM",
		Credentials: CredentialData{
			FCMProjectID:     "my-project",
			FCMServiceAccount: `{"type":"service_account"}`,
		},
		Environment: "production",
		IsDefault:   true,
	}

	if req.Provider != PlatformAndroid {
		t.Error("Provider should be android")
	}
	if req.Name == "" {
		t.Error("Name should not be empty")
	}
	if err := req.Credentials.Validate(req.Provider); err != nil {
		t.Errorf("Credentials should be valid: %v", err)
	}
}
