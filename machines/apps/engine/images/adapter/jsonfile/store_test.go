package jsonfile_test

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/images"

	"github.com/jterrazz/studio-station/apps/engine/images/adapter/jsonfile"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/images/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func TestImageStore(t *testing.T) {
	contracttest.ImageStore(t, func(t *testing.T) contract.ImageStore {
		return jsonfile.New(local.New(nil, nil, nil, nil).FS, t.TempDir())
	})
}

func TestBuildMarks(t *testing.T) {
	contracttest.BuildMarks(t, func(t *testing.T) contract.BuildMarks {
		return jsonfile.NewMarks(local.New(nil, nil, nil, nil).FS, t.TempDir())
	})
}

type vanishing struct {
	jsonfile.Files
	gone string
}

func (v vanishing) ReadFile(name string) ([]byte, error) {
	if path.Base(name) == v.gone {
		return nil, fs.ErrNotExist
	}
	return v.Files.ReadFile(name)
}

func TestAMarkClearedWhileTheFolderIsReadIsNone(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	files := local.New(nil, nil, nil, nil).FS
	for _, image := range []string{"base-ubuntu", "studio-linux"} {
		if err := jsonfile.NewMarks(files, dir).Mark(ctx, images.Making{Image: image, Version: "1", Step: images.BuildSeal}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := jsonfile.NewMarks(vanishing{Files: files, gone: "base-ubuntu.json"}, dir).Marked(ctx)
	if err != nil || len(got) != 1 || got[0].Image != "studio-linux" {
		t.Fatalf("Marked = %+v, %v; want the mark still there", got, err)
	}
}

func TestTheLastRecordDeletedTakesItsImageFolderWithIt(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	files := local.New(nil, nil, nil, nil).FS
	store := jsonfile.New(files, dir)
	for _, version := range []string{"1", "2"} {
		if err := store.Put(ctx, images.Record{Name: "base-ubuntu", Version: version}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(ctx, "base-ubuntu", "1"); err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadDir(path.Join(dir, "base-ubuntu")); err != nil {
		t.Fatalf("the folder went with one of two records: %v", err)
	}
	if err := store.Delete(ctx, "base-ubuntu", "2"); err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadDir(path.Join(dir, "base-ubuntu")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the image folder stays after its last record: %v", err)
	}
}
