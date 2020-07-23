package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/remotes"
	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	emptyDigest  digest.Digest
	emptyDesc    ocispec.Descriptor
	zstdLayerTyp = "application/vnd.oci.image.layer.v1.tar+zstd"
)

var imageZstdConvertCommand = cli.Command{
	Name:        "zstdconv",
	Usage:       "Convert image layer into zstd type",
	ArgsUsage:   "<src-image> <dst-image>",
	Description: `Export images to an OCI tar[.gz] into zstd format`,
	Action: func(context *cli.Context) error {
		var (
			srcImage    = context.Args().First()
			targetImage = context.Args().Get(1)
		)
		if srcImage == "" || targetImage == "" {
			return errors.New("please provide both an output filename and an image reference")
		}

		cli, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()

		ctx, done, err := cli.WithLease(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to create lease")
		}
		defer done(ctx)

		img, err := ensureImageExist(ctx, cli, srcImage)
		if err != nil {
			return err
		}

		cs := cli.ContentStore()
		manifest, err := currentPlatformManifest(ctx, cs, img)
		if err != nil {
			return errors.Wrap(err, "failed to read manifest")
		}

		newMfDesc, err := zstdLayersConv(ctx, cs, manifest)
		if err != nil {
			return errors.Wrap(err, "failed to handle zstd converter")
		}

		newImage := images.Image{
			Name:   targetImage,
			Target: newMfDesc,
		}
		return createImage(ctx, cli.ImageService(), newImage)
	},
}

func ensureImageExist(ctx context.Context, cli *containerd.Client, imageName string) (containerd.Image, error) {
	img, err := cli.GetImage(ctx, imageName)
	if err != nil {
		if !errdefs.IsNotFound(err) {
			return nil, errors.Wrapf(err, "failed to get image %v", imageName)
		}
		return cli.Pull(ctx, imageName, func(_ *containerd.Client, rc *containerd.RemoteContext) error {
			rc.Unpack = false
			return nil
		})
	}
	return img, nil
}

func currentPlatformManifest(ctx context.Context, cs content.Provider, img containerd.Image) (ocispec.Manifest, error) {
	return images.Manifest(ctx, cs, img.Target(), platforms.Default())
}

func zstdLayersConv(ctx context.Context, cs content.Store, manifest ocispec.Manifest) (_ ocispec.Descriptor, err0 error) {
	var copyManifest = struct {
		ocispec.Manifest `json:",omitempty"`
		// MediaType is the media type of the object this schema refers to.
		MediaType string `json:"mediaType,omitempty"`
	}{
		Manifest:  manifest,
		MediaType: images.MediaTypeDockerSchema2Manifest,
	}

	copyManifest.Layers = nil
	for _, layer := range manifest.Layers {
		d, size, err := convertTarGZIntoZstd(ctx, cs, layer)
		if err != nil {
			return emptyDesc, err
		}

		copyManifest.Layers = append(copyManifest.Layers, ocispec.Descriptor{
			MediaType: zstdLayerTyp,
			Digest:    d,
			Size:      size,
		})
	}

	mb, err := json.MarshalIndent(copyManifest, "", "   ")
	if err != nil {
		return emptyDesc, err
	}
	fmt.Println(string(mb))

	desc := ocispec.Descriptor{
		MediaType: copyManifest.MediaType,
		Digest:    digest.Canonical.FromBytes(mb),
		Size:      int64(len(mb)),
	}

	labels := map[string]string{}
	labels["containerd.io/gc.ref.content.0"] = copyManifest.Config.Digest.String()
	for i, ch := range copyManifest.Layers {
		labels[fmt.Sprintf("containerd.io/gc.ref.content.%d", i+1)] = ch.Digest.String()
	}

	ref := remotes.MakeRefKey(ctx, desc)
	if err := content.WriteBlob(ctx, cs, ref, bytes.NewReader(mb), desc, content.WithLabels(labels)); err != nil {
		return emptyDesc, errors.Wrap(err, "failed to write image manifest")
	}
	return desc, nil
}

type readCounter struct {
	r io.Reader
	c int64
}

func (rc *readCounter) Read(p []byte) (n int, err error) {
	n, err = rc.r.Read(p)
	rc.c += int64(n)
	return
}

func convertTarGZIntoZstd(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (digest.Digest, int64, error) {
	labels := map[string]string{
		"kubeConEU2020/from-blob": string(desc.Digest),
	}

	cw, err := content.OpenWriter(ctx, cs, content.WithRef(fmt.Sprintf("zstd-layer-%v", desc.Digest)))
	if err != nil {
		return emptyDigest, 0, errors.Wrapf(err, "failed to open writer")
	}
	defer cw.Close()

	// read layer blob
	ra, err := cs.ReaderAt(ctx, desc)
	if err != nil {
		return emptyDigest, 0, err
	}
	defer ra.Close()

	dr, err := compression.DecompressStream(content.NewReader(ra))
	if err != nil {
		return emptyDigest, 0, err
	}
	defer dr.Close()

	pipeR, pipeW := io.Pipe()

	cmd := exec.CommandContext(ctx, "zstd", "-cf")
	cmd.Stdin = dr
	cmd.Stdout = pipeW
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	go func() {
		pipeW.CloseWithError(cmd.Run())
	}()

	digester := digest.Canonical.Digester()

	rc := &readCounter{
		r: io.TeeReader(pipeR, digester.Hash()),
	}

	if _, err := io.Copy(cw, rc); err != nil {
		return emptyDigest, 0, errors.Wrap(err, "failed to copy")
	}

	size := rc.c
	dig := digester.Digest()
	if err := cw.Commit(ctx, size, dig, content.WithLabels(labels)); err != nil {
		if !errdefs.IsAlreadyExists(err) {
			return emptyDigest, 0, errors.Wrapf(err, "failed commit")
		}
	}
	return dig, size, nil
}

func createImage(ctx context.Context, is images.Store, img images.Image) error {
	for {
		if created, err := is.Create(ctx, img); err != nil {
			if !errdefs.IsAlreadyExists(err) {
				return err
			}

			updated, err := is.Update(ctx, img)
			if err != nil {
				if errdefs.IsNotFound(err) {
					continue
				}
				return err
			}

			img = updated
		} else {
			img = created
		}
		return nil
	}
}
