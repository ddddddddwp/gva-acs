package adapter

import (
	"context"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newConnectionProfileRepositoryTest(t *testing.T) (*ConnectionProfileRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(model.Device), new(model.ConnectionProfile)); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	cipher, err := NewCredentialCipher(testCredentialConfig(0x51))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	return NewConnectionProfileRepository(db, cipher), db
}

func createConnectionProfileDevice(t *testing.T, db *gorm.DB, serial string) model.Device {
	t.Helper()
	device := model.Device{SerialNumber: serial, OUI: "001122"}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create device: %v", err)
	}
	return device
}

func loadConnectionProfile(t *testing.T, db *gorm.DB, deviceID uint) model.ConnectionProfile {
	t.Helper()
	var profile model.ConnectionProfile
	if err := db.First(&profile, "device_id = ?", deviceID).Error; err != nil {
		t.Fatalf("load profile: %v", err)
	}
	return profile
}

func TestConnectionProfileCollectDiscoversReadyCredentials(t *testing.T) {
	repo, db := newConnectionProfileRepositoryTest(t)
	device := createConnectionProfileDevice(t, db, "PROFILE-COLLECT")

	result, err := repo.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName:      " http://127.0.0.1:8400 ",
		connectionRequestUsernameName: "existing-user",
		connectionRequestPasswordName: "existing-password",
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if result.NeedsProvisioning {
		t.Fatal("known credentials should not require provisioning")
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.DiscoveredURL != "http://127.0.0.1:8400" || profile.Username != "existing-user" {
		t.Fatalf("profile identity=%#v", profile)
	}
	if profile.ProvisionState != model.ConnectionProfileStateReady || len(profile.PasswordCiphertext) == 0 {
		t.Fatalf("profile readiness=%#v", profile)
	}
	resolved, err := repo.Resolve(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Password != "existing-password" || resolved.URL != "http://127.0.0.1:8400" {
		t.Fatalf("resolved=%#v", resolved)
	}
}

func TestConnectionProfileCollectDoesNotClearURLOrPassword(t *testing.T) {
	repo, db := newConnectionProfileRepositoryTest(t)
	device := createConnectionProfileDevice(t, db, "PROFILE-PRESERVE")
	if _, err := repo.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName:      "http://127.0.0.1:8400",
		connectionRequestUsernameName: "preserved-user",
		connectionRequestPasswordName: "preserved-password",
	}); err != nil {
		t.Fatalf("seed collect: %v", err)
	}
	before := loadConnectionProfile(t, db, device.ID)

	result, err := repo.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestPasswordName: "",
	})
	if err != nil {
		t.Fatalf("empty collect: %v", err)
	}
	after := loadConnectionProfile(t, db, device.ID)
	if result.NeedsProvisioning || after.DiscoveredURL != before.DiscoveredURL || after.Username != before.Username {
		t.Fatalf("profile identity changed: before=%#v after=%#v", before, after)
	}
	if string(after.PasswordCiphertext) != string(before.PasswordCiphertext) || after.CredentialKeyVersion != before.CredentialKeyVersion {
		t.Fatal("empty password readback replaced the stored credential")
	}
}

func TestConnectionProfileManualCredentialsWinOverAutomaticCollection(t *testing.T) {
	repo, db := newConnectionProfileRepositoryTest(t)
	device := createConnectionProfileDevice(t, db, "PROFILE-MANUAL")
	if err := db.Create(&model.ConnectionProfile{
		DeviceID: device.ID, DiscoveredURL: "http://127.0.0.1:8400",
		ProvisionState: model.ConnectionProfileStateDiscovered, AuthScheme: "digest",
	}).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := repo.StoreCredential(context.Background(), db, device.ID, "manual-user", "manual-password", model.ConnectionCredentialSourceManual); err != nil {
		t.Fatalf("store manual credential: %v", err)
	}

	if _, err := repo.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName:      "http://127.0.0.1:8500",
		connectionRequestUsernameName: "automatic-user",
		connectionRequestPasswordName: "automatic-password",
	}); err != nil {
		t.Fatalf("collect automatic values: %v", err)
	}
	profile := loadConnectionProfile(t, db, device.ID)
	if profile.DiscoveredURL != "http://127.0.0.1:8500" {
		t.Fatalf("discovered URL was not refreshed: %q", profile.DiscoveredURL)
	}
	if profile.Username != "manual-user" || profile.CredentialSource != model.ConnectionCredentialSourceManual {
		t.Fatalf("manual identity overwritten: %#v", profile)
	}
	resolved, err := repo.Resolve(context.Background(), device.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.Password != "manual-password" {
		t.Fatal("manual password was overwritten")
	}
}

func TestConnectionProfileCollectRequestsProvisioningForURLWithoutCredentials(t *testing.T) {
	repo, db := newConnectionProfileRepositoryTest(t)
	device := createConnectionProfileDevice(t, db, "PROFILE-NEEDS-PROVISION")

	result, err := repo.Collect(context.Background(), device.ID, map[string]string{
		connectionRequestURLName: "http://127.0.0.1:8400",
	})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if !result.NeedsProvisioning || result.Profile.ProvisionState != model.ConnectionProfileStateDiscovered {
		t.Fatalf("collect result=%#v", result)
	}
}
