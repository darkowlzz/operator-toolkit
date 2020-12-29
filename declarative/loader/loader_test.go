package loader

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/api/filesys"
)

func TestLoadPackage(t *testing.T) {
	// Create an in-memory filesystem and load the packages in it.
	fs := ManifestFileSystem{filesys.MakeFsInMemory()}
	err := LoadPackages(fs, "../testdata/channels", "")
	assert.Nil(t, err)

	wantSA, err := ioutil.ReadFile("../testdata/channels/packages/guestbook/0.1.0/service_account.yaml")
	assert.Nil(t, err)
	wantDB, err := ioutil.ReadFile("../testdata/channels/packages/registry/0.3.0/db.yaml")
	assert.Nil(t, err)

	b, err := fs.ReadFile("guestbook/service_account.yaml")
	assert.Nil(t, err)
	assert.Equal(t, string(wantSA), string(b))

	b, err = fs.ReadFile("registry/db.yaml")
	assert.Nil(t, err)
	assert.Equal(t, string(wantDB), string(b))
}
