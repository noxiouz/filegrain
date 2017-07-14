package lazyfs

import (
	"github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/opencontainers/go-digest"
	continuitypb "github.com/stevvooe/continuity/proto"
)

type file struct {
	opts Options
	res  *continuitypb.Resource
	nodefs.File
}

func newFile(opts Options, res *continuitypb.Resource) nodefs.File {
	f := new(file)
	f.opts = opts
	f.res = res
	f.File = nodefs.NewDefaultFile()
	return f
}

func (f *file) GetAttr(out *fuse.Attr) fuse.Status {
	*out = *continuityResourceToFuseAttr(f.res)
	return fuse.OK
}

func (f *file) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	if len(f.res.Digest) == 0 {
		logrus.Errorf("no digest for %#v", f.res)
		return nil, fuse.EIO
	}
	dgst := digest.Digest(f.res.Digest[0])
	br, err := f.opts.Puller.PullBlob(f.opts.Image, dgst)
	if err != nil {
		logrus.Errorf("error while pulling %s: %v", dgst, err)
		return nil, fuse.EIO
	}
	if _, err := br.Seek(off, 0); err != nil {
		logrus.Errorf("error while seeking %s to %d: %v", dgst, off, err)
		return nil, fuse.EIO
	}
	if _, err := br.Read(buf); err != nil {
		logrus.Errorf("error while reading %d bytes at %d for %s: %v",
			len(buf), off, dgst, err)
		return nil, fuse.EIO
	}
	defer br.Close()
	return fuse.ReadResultData(buf), fuse.OK
}