package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestArchivesAndNoOverwrite(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "stage")
	os.Mkdir(source, 0700)
	os.WriteFile(filepath.Join(source, "tunnelctl"), []byte("synthetic executable"), 0700)
	for _, isZip := range []bool{false, true} {
		path := filepath.Join(root, "asset.tar.gz")
		if isZip {
			path = filepath.Join(root, "asset.zip")
		}
		if e := archive(source, path, isZip); e != nil {
			t.Fatal(e)
		}
		if e := archive(source, path, isZip); e == nil {
			t.Fatal("existing release asset overwritten")
		}
		if isZip {
			z, e := zip.OpenReader(path)
			if e != nil {
				t.Fatal(e)
			}
			if len(z.File) != 1 || z.File[0].Name != "tunnelctl" {
				t.Fatal("ZIP content mismatch")
			}
			z.Close()
		} else {
			f, e := os.Open(path)
			if e != nil {
				t.Fatal(e)
			}
			gz, e := gzip.NewReader(f)
			if e != nil {
				t.Fatal(e)
			}
			r := tar.NewReader(gz)
			h, e := r.Next()
			if e != nil || h.Name != "tunnelctl" || h.Mode != 0755 {
				t.Fatal("Unix executable mode missing")
			}
			b, e := io.ReadAll(r)
			if e != nil || string(b) != "synthetic executable" {
				t.Fatal("archive content mismatch")
			}
			gz.Close()
			f.Close()
		}
	}
	if e := run("v00.2.0", filepath.Join(root, "bad"), true); e == nil {
		t.Fatal("invalid release tag accepted")
	}
}
