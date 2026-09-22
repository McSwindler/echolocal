package bootimg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchTimeout bounds the download. Ten megabytes should not take this long on any connection worth
// installing over, and a stalled fetch should say so rather than hang between a device being plugged
// in and anything happening to it.
const fetchTimeout = 5 * time.Minute

// Resolve is the image's bytes, fetched from where it is published.
//
// Nothing the network hands back is trusted. The hash, the size and the kernel cmdline that makes
// this image the point of the exercise are all compiled into echoctl, and Verify checks every one of
// them before a caller sees a byte — so the worst a substituted download can do is fail. The trust
// root is this binary, which is the same argument the device's update channel makes for compiling
// its URLs in, and it holds harder here: this one is written to a boot partition.
//
// Nothing is kept afterwards. An install is rare and the image is a few megabytes, which is not worth
// leaving behind on a machine that only asked to install to a device.
//
// progress is called with a fraction as the download runs, and may be nil.
func (i Image) Resolve(ctx context.Context, progress func(float64)) ([]byte, string, error) {
	if i.URL == "" {
		return nil, "", fmt.Errorf("bootimg: no boot image is published for %s, so one has to be given with --boot-image", i.Device)
	}

	data, err := i.download(ctx, progress)
	if err != nil {
		return nil, "", err
	}
	if err := i.Verify(i.URL, data); err != nil {
		return nil, "", err
	}
	return data, i.URL, nil
}

func (i Image) download(ctx context.Context, progress func(float64)) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, i.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bootimg: fetching %s: %w", i.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bootimg: fetching %s: %s", i.URL, resp.Status)
	}

	// Sized from what the image is known to be rather than from what the server says it is, and the
	// read is capped there too: a response claiming to be far larger is refused before it is held.
	buf := make([]byte, 0, i.Size)
	sum := sha256.New()
	body := io.LimitReader(resp.Body, i.Size+1)

	chunk := make([]byte, 64*1024)
	for {
		n, err := body.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			sum.Write(chunk[:n])
			if progress != nil && i.Size > 0 {
				progress(min(float64(len(buf))/float64(i.Size), 1))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bootimg: fetching %s: %w", i.URL, err)
		}
	}

	if got := hex.EncodeToString(sum.Sum(nil)); got != i.SHA256 {
		return nil, fmt.Errorf("bootimg: %s hashes to %s, want %s", i.URL, got, i.SHA256)
	}
	return buf, nil
}
