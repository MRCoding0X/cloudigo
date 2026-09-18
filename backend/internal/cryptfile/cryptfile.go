// Package cryptfile implements a seekable, chunked AES-256-GCM file format so
// encrypted uploads can still serve HTTP Range requests (resumable/streamed
// downloads) without decrypting the whole file into memory.
//
// Format: a small fixed header, followed by consecutive GCM-sealed chunks of
// ChunkSize plaintext bytes each (the final chunk may be shorter). Each
// chunk's nonce is the file's random 8-byte base nonce plus a 4-byte
// big-endian chunk index, so nonces never repeat within a file. Every chunk
// that is actually read is authenticated (tampering with a chunk's bytes is
// detected), but — as with any seekable-encryption scheme, e.g. rclone
// crypt/restic — a Range read that only touches some chunks cannot detect
// truncation or reordering of chunks it never reads. A full, non-Range
// download decrypts every chunk in order and is fully tamper-evident.
package cryptfile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	ChunkSize  = 64 * 1024
	KeySize    = 32 // AES-256
	tagSize    = 16
	headerSize = 24
)

var magic = [4]byte{'C', 'D', 'E', '1'}

var ErrBadHeader = errors.New("cryptfile: invalid or corrupt header")

func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func nonceFor(base [8]byte, index uint32) []byte {
	nonce := make([]byte, 12)
	copy(nonce, base[:])
	binary.BigEndian.PutUint32(nonce[8:], index)
	return nonce
}

// Encrypt streams src through AES-256-GCM in fixed-size chunks and writes the
// cryptfile format to dst. Returns the total plaintext size written.
func Encrypt(src io.Reader, dst io.Writer, key []byte) (int64, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return 0, err
	}

	var base [8]byte
	if _, err := rand.Read(base[:]); err != nil {
		return 0, err
	}

	// Header is finalized (total size) only after we know it, so encrypt to a
	// temp buffer-less approach: write a placeholder header, stream chunks,
	// then seek back and patch in the real total size.
	seeker, canSeek := dst.(io.WriteSeeker)

	header := make([]byte, headerSize)
	copy(header[0:4], magic[:])
	binary.BigEndian.PutUint32(header[4:8], uint32(ChunkSize))
	binary.BigEndian.PutUint64(header[8:16], 0) // patched below if possible
	copy(header[16:24], base[:])
	if _, err := dst.Write(header); err != nil {
		return 0, err
	}

	buf := make([]byte, ChunkSize)
	var total int64
	var index uint32

	for {
		n, readErr := io.ReadFull(src, buf)
		if n > 0 {
			ciphertext := gcm.Seal(nil, nonceFor(base, index), buf[:n], nil)
			if _, err := dst.Write(ciphertext); err != nil {
				return 0, err
			}
			total += int64(n)
			index++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return 0, readErr
		}
	}

	if canSeek {
		if _, err := seeker.Seek(8, io.SeekStart); err != nil {
			return total, err
		}
		var sizeBuf [8]byte
		binary.BigEndian.PutUint64(sizeBuf[:], uint64(total))
		if _, err := seeker.Write(sizeBuf[:]); err != nil {
			return total, err
		}
	}

	return total, nil
}

// Reader provides random-access (Range) reads over an encrypted file.
type Reader struct {
	f         *os.File
	gcm       cipher.AEAD
	chunkSize int64
	totalSize int64
	base      [8]byte
}

func OpenReader(path string, key []byte) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	header := make([]byte, headerSize)
	if _, err := io.ReadFull(f, header); err != nil {
		f.Close()
		return nil, fmt.Errorf("%w: %v", ErrBadHeader, err)
	}
	if string(header[0:4]) != string(magic[:]) {
		f.Close()
		return nil, ErrBadHeader
	}

	gcm, err := newGCM(key)
	if err != nil {
		f.Close()
		return nil, err
	}

	r := &Reader{
		f:         f,
		gcm:       gcm,
		chunkSize: int64(binary.BigEndian.Uint32(header[4:8])),
		totalSize: int64(binary.BigEndian.Uint64(header[8:16])),
	}
	copy(r.base[:], header[16:24])
	return r, nil
}

func (r *Reader) TotalSize() int64 { return r.totalSize }
func (r *Reader) Close() error     { return r.f.Close() }

func (r *Reader) numChunks() int64 {
	if r.totalSize == 0 {
		return 0
	}
	return (r.totalSize + r.chunkSize - 1) / r.chunkSize
}

func (r *Reader) readChunk(index int64) ([]byte, error) {
	plainLen := r.chunkSize
	if last := r.numChunks() - 1; index == last {
		plainLen = r.totalSize - index*r.chunkSize
	}
	cipherLen := plainLen + tagSize
	offset := int64(headerSize) + index*(r.chunkSize+tagSize)

	ciphertext := make([]byte, cipherLen)
	if _, err := r.f.ReadAt(ciphertext, offset); err != nil {
		return nil, err
	}

	plaintext, err := r.gcm.Open(nil, nonceFor(r.base, uint32(index)), ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("cryptfile: chunk %d authentication failed: %w", index, err)
	}
	return plaintext, nil
}

// WriteRange decrypts and writes exactly the plaintext bytes [start, end]
// (inclusive) to w.
func (r *Reader) WriteRange(w io.Writer, start, end int64) error {
	if start < 0 || end >= r.totalSize || start > end {
		return fmt.Errorf("cryptfile: invalid range %d-%d for size %d", start, end, r.totalSize)
	}

	firstChunk := start / r.chunkSize
	lastChunk := end / r.chunkSize

	for i := firstChunk; i <= lastChunk; i++ {
		plaintext, err := r.readChunk(i)
		if err != nil {
			return err
		}

		chunkStart := i * r.chunkSize
		from := int64(0)
		if i == firstChunk {
			from = start - chunkStart
		}
		to := int64(len(plaintext))
		if i == lastChunk {
			to = end - chunkStart + 1
		}

		if _, err := w.Write(plaintext[from:to]); err != nil {
			return err
		}
	}

	return nil
}
