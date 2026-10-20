// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentSizePolicy(t *testing.T) {
	const gib uint64 = 1 << 30
	for _, tc := range []struct {
		name string
		size uint64
		allow bool
	}{
		{"report.zip", gib, true},
		{"report.zip", gib + 1, false},
		{"video.MP4", 5 * gib, true},
		{"video.MP4", 5*gib + 1, false},
		{"video.webm", 2 * gib, true},
		{"archive.mp4.zip", 2 * gib, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.allow, attachmentSizeAllowed(tc.name, tc.size))
		})
	}
}

func TestTaskAttachment_ReadOne(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("Normal File", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		files.InitTestFileFixtures(t)
		ta := &TaskAttachment{
			ID: 1,
		}
		err := ta.ReadOne(s, u)
		require.NoError(t, err)
		assert.NotNil(t, ta.File)
		assert.True(t, ta.File.ID == ta.FileID && ta.FileID != 0)

		// Load the actual attachment file and check its content
		err = ta.File.LoadFileByID()
		require.NoError(t, err)

		// Validate the file exists at the expected storage path
		stat, err := files.FileStat(ta.File)
		require.NoError(t, err)
		assert.NotNil(t, stat)

		content := make([]byte, 9)
		read, err := ta.File.File.Read(content)
		require.NoError(t, err)
		assert.Equal(t, 9, read)
		assert.Equal(t, []byte("testfile1"), content)
	})
	t.Run("Nonexisting Attachment", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		files.InitTestFileFixtures(t)
		ta := &TaskAttachment{
			ID: 9999,
		}
		err := ta.ReadOne(s, u)
		require.Error(t, err)
		assert.True(t, IsErrTaskAttachmentDoesNotExist(err))
	})
	t.Run("Existing Attachment, Nonexisting File", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		files.InitTestFileFixtures(t)
		ta := &TaskAttachment{
			ID: 2,
		}
		err := ta.ReadOne(s, u)
		require.Error(t, err)
		require.EqualError(t, err, "file 9999 does not exist")
	})
}

func TestTaskAttachment_NewAttachment(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	files.InitTestFileFixtures(t)
	// Assert the file is being stored correctly
	ta := TaskAttachment{
		TaskID: 1,
	}
	testuser := &user.User{ID: 1}

	content := []byte("testingstuff")
	err := ta.NewAttachment(s, bytes.NewReader(content), "testfile", uint64(len(content)), testuser)
	require.NoError(t, err)
	assert.NotEqual(t, 0, ta.FileID)
	_, err = files.FileStat(ta.File)
	require.NoError(t, err)
	assert.False(t, os.IsNotExist(err))
	assert.Equal(t, testuser.ID, ta.CreatedByID)

	err = s.Commit()
	require.NoError(t, err)

	s2 := db.NewSession()
	defer s2.Close()

	// Check the file was inserted correctly
	ta.File = &files.File{ID: ta.FileID}
	err = ta.File.LoadFileMetaByID(s2)
	require.NoError(t, err)
	assert.Equal(t, testuser.ID, ta.File.CreatedByID)
	assert.Equal(t, "testfile", ta.File.Name)
	assert.Equal(t, uint64(len(content)), ta.File.Size)
	assert.NotEmpty(t, ta.File.Mime, "mime type should be detected and stored")

	// Extra test for max size test
}

// Upload and delete events are recorded in the immutable task history.
// The delete event must retain metadata after the underlying file is removed.
func TestTaskAttachment_ImmutableHistory(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	files.InitTestFileFixtures(t)
	actor := &user.User{ID: 1}
	content := []byte("audit attachment bytes")

	s := db.NewSession()
	attachment := &TaskAttachment{TaskID: 1, Source: "task"}
	require.NoError(t, attachment.NewAttachment(s, bytes.NewReader(content), "audit-archive.zip", uint64(len(content)), actor))
	require.NoError(t, s.Commit())
	require.NoError(t, s.Close())

	reader := db.NewSession()
	rows, total, err := ListHistory(reader, "task", 1, 1, 50)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "upload", rows[0].Action)
	assert.Equal(t, "attachment", rows[0].Field)
	assert.Equal(t, actor.ID, rows[0].ActorID)
	assert.Contains(t, rows[0].NewValue, "audit-archive.zip")
	assert.Contains(t, rows[0].NewValue, "来源：task")
	assert.NotEmpty(t, rows[0].Created)
	require.NoError(t, reader.Close())

	remove := db.NewSession()
	require.NoError(t, (&TaskAttachment{ID: attachment.ID, TaskID: 1}).Delete(remove, actor))
	require.NoError(t, remove.Commit())
	require.NoError(t, remove.Close())

	reader = db.NewSession()
	defer reader.Close()
	rows, total, err = ListHistory(reader, "task", 1, 1, 50)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
	assert.Equal(t, "delete", rows[0].Action)
	assert.Equal(t, "attachment", rows[0].Field)
	assert.Contains(t, rows[0].OldValue, "audit-archive.zip")
	assert.Empty(t, rows[0].NewValue)
	assert.Equal(t, "upload", rows[1].Action)
}

func TestTaskAttachment_ReadAll(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	files.InitTestFileFixtures(t)
	ta := &TaskAttachment{TaskID: 1}
	as, _, _, err := ta.ReadAll(s, &user.User{ID: 1}, "", 0, 50)
	attachments, _ := as.([]*TaskAttachment)
	require.NoError(t, err)
	assert.Len(t, attachments, 3)
	assert.Equal(t, "test", attachments[0].File.Name)
	for _, a := range attachments {
		assert.NotNil(t, a.CreatedBy)
	}
	assert.Equal(t, int64(-2), attachments[2].CreatedByID)
	assert.Equal(t, int64(-2), attachments[2].CreatedBy.ID)
}

func TestTaskAttachment_Delete(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("Normal", func(t *testing.T) {
		files.InitTestFileFixtures(t)
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ta := &TaskAttachment{ID: 1}
		err := ta.Delete(s, u)
		require.NoError(t, err)
		// Check if the file itself was deleted
		_, err = files.FileStat(ta.File) // The new file has the id 2 since it's the second attachment
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
	t.Run("Nonexisting", func(t *testing.T) {
		files.InitTestFileFixtures(t)
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ta := &TaskAttachment{ID: 9999}
		err := ta.Delete(s, u)
		require.Error(t, err)
		assert.True(t, IsErrTaskAttachmentDoesNotExist(err))
	})
	t.Run("Existing attachment, nonexisting file", func(t *testing.T) {
		files.InitTestFileFixtures(t)
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ta := &TaskAttachment{ID: 2}
		err := ta.Delete(s, u)
		require.NoError(t, err)
	})
}

func TestAttachmentPreviewRejectsLargeImages(t *testing.T) {
	// Create a 10000x10000 pixel image (100M pixels, well above the 50M limit)
	// As a PNG this is small on disk but huge when decoded into memory
	img := image.NewNRGBA(image.Rect(0, 0, 10000, 10000))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	attachment := &TaskAttachment{
		ID: 999999,
		File: &files.File{
			File: io.NopCloser(bytes.NewReader(buf.Bytes())),
		},
	}

	result := attachment.GetPreview(PreviewMedium)
	assert.Nil(t, result, "Preview should be nil for images exceeding max pixel count")
}

func TestTaskAttachment_Permissions(t *testing.T) {
	u := &user.User{ID: 1}
	t.Run("Can Read", func(t *testing.T) {
		t.Run("Allowed", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 1}
			can, _, err := ta.CanRead(s, u)
			require.NoError(t, err)
			assert.True(t, can)
		})
		t.Run("Forbidden", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 14}
			can, _, err := ta.CanRead(s, u)
			require.NoError(t, err)
			assert.False(t, can)
		})
	})
	t.Run("Can Delete", func(t *testing.T) {
		t.Run("Allowed", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 1}
			can, err := ta.CanDelete(s, u)
			require.NoError(t, err)
			assert.True(t, can)
		})
		t.Run("Forbidden, no access", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 14}
			can, err := ta.CanDelete(s, u)
			require.NoError(t, err)
			assert.False(t, can)
		})
		t.Run("Forbidden, shared read only", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 15}
			can, err := ta.CanDelete(s, u)
			require.NoError(t, err)
			assert.False(t, can)
		})
	})
	t.Run("Can Create", func(t *testing.T) {
		t.Run("Allowed", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 1}
			can, err := ta.CanCreate(s, u)
			require.NoError(t, err)
			assert.True(t, can)
		})
		t.Run("Forbidden, no access", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 14}
			can, err := ta.CanCreate(s, u)
			require.NoError(t, err)
			assert.False(t, can)
		})
		t.Run("Forbidden, shared read only", func(t *testing.T) {
			db.LoadAndAssertFixtures(t)
			s := db.NewSession()
			defer s.Close()

			ta := &TaskAttachment{TaskID: 15}
			can, err := ta.CanCreate(s, u)
			require.NoError(t, err)
			assert.False(t, can)
		})
	})
}
