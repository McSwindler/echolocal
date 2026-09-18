package bootimg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// fetchTimeout bounds the download. Ten megabytes should not take this long on any connection worth
// installing over, and a stalled fetch should say so rather than hang between a device being plugged
// in and anything happening to it.
const fetchTimeout = 5 * time.Minute

// Resolve is the image's bytes, from the cache or from where it is published.
//
// Nothing the network hands back is trusted. The hash, the size and the kernel cmdline that makes
// this image the point of the exercise are all compiled into echoctl, and Verify checks every one of
// them before a caller sees a byte — so the worst a substituted download can do is fail. The trust
// root is this binary, which is the same argument the device's update channel makes for compiling
// its URLs in, and it holds harder here: this one is written to a boot partition.
//
// progress is called with a fraction as the download runs, and may be nil.
func (i Image) Resolve(ctx context.Context, progress func(float64)) ([]byte, string, error) {
	cached, err := i.CachePath()
	if err == nil {
		if data, err := os.ReadFile(cached); err == nil {
			if err := i.Verify(cached, data); err == nil {
				return data, cached, nil
			}
			// A cache file is named by the hash it has to have, so one that does not verify is corrupt
			// rather than stale. Dropped and fetched again.
			_ = os.Remove(cached)
		}
	}

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

	if cached != "" {
		if err := writeCache(cached, data); err != nil {
			// Not fatal: the image is in hand and the install can go ahead. The only cost is fetching
			// it again next time.
			return data, i.URL, nil
		}
		return data, cached, nil
	}
	return data, i.URL, nil
}

// CachePath is where a fetched image is kept, named by the hash it has to have.
//
// Content-addressed, so the cache cannot go stale: a file either is the image this build wants or is
// not, and the name says which. Two boards, or two versions of echoctl, share the directory without
// either having to know about the other.
func (i Image) CachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "echolocal", "boot", i.SHA256+".img"), nil
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

// writeCache puts the image where the next run will find it, through a temporary file so a fetch
// interrupted half way cannot leave something that reads as the real thing.
func writeCache(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".incoming-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
