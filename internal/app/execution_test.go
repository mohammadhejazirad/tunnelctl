package app

import "testing"

func TestTransientAppImageSessionsRefused(t *testing.T) {
	t.Setenv("APPIMAGE", "/synthetic/app.AppImage")
	t.Setenv("APPDIR", "/tmp/appimage_extracted_synthetic")
	if checkSessionExecution() == nil {
		t.Fatal("unsafe transient supervisor accepted")
	}
	t.Setenv("APPDIR", "/tmp/.mount_synthetic")
	t.Setenv("APPIMAGE_EXTRACT_AND_RUN", "")
	if e := checkSessionExecution(); e != nil {
		t.Fatal(e)
	}
	t.Setenv("APPIMAGE_EXTRACT_AND_RUN", "1")
	if checkSessionExecution() == nil {
		t.Fatal("explicit transient environment accepted")
	}
	t.Setenv("APPIMAGE", "")
	if e := checkSessionExecution(); e != nil {
		t.Fatal("ordinary extracted executable refused", e)
	}
}
