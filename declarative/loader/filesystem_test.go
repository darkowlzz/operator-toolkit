package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/api/filesys"
)

func TestDeepCopy(t *testing.T) {
	testFilePath := "guestbook/service_account.yaml"

	// Load a filesystem.
	fs1, err := NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	wantContent, err := fs1.ReadFile(testFilePath)
	assert.Nil(t, err)

	// Create an empty filesystem.
	fs2 := &ManifestFileSystem{FileSystem: filesys.MakeFsInMemory()}

	// Ensure it's empty.
	_, err = fs2.ReadFile(testFilePath)
	assert.NotNil(t, err)

	// Copy.
	err = DeepCopy(fs1, fs2)
	assert.Nil(t, err)

	// Read the file again and check content.
	gotContent, err := fs2.ReadFile(testFilePath)
	assert.Nil(t, err)
	assert.Equal(t, string(wantContent), string(gotContent))

	assert.True(t, fs2.Exists("/guestbook/role.yaml"))
	assert.True(t, fs2.Exists("/guestbook/kustomization.yaml"))
	assert.True(t, fs2.Exists("/registry/db.yaml"))
	assert.True(t, fs2.Exists("/registry/frontend.yaml"))
	assert.True(t, fs2.Exists("/registry/kustomization.yaml"))
}
