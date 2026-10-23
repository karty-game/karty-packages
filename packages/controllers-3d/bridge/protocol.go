package bridge

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"unicode/utf8"
)

const maximumBytes = 16 << 20
const maximumFiles = 256
const maximumEnvelope = maximumBytes + maximumFiles*262 + 8

// Process consumes the ordinary bounded KPR1 processor request and emits a
// deterministic response. Diagnostics belong on stderr, outside this stream.
func Process(input io.Reader, output io.Writer) error {
	encoded, err := io.ReadAll(io.LimitReader(input, maximumEnvelope+1))
	if err != nil {
		return err
	}
	files, err := decode(encoded)
	if err != nil {
		return err
	}
	goMod, ok := files["go.mod"]
	if !ok || len(files) != 1 {
		return fmt.Errorf("controllers-3d.sdk requires exactly the authored go.mod input")
	}
	generated, err := Generate(goMod)
	if err != nil {
		return err
	}
	response := encode(generated)
	if _, err := decode(response); err != nil {
		return err
	}
	_, err = io.Copy(output, bytes.NewReader(response))
	return err
}

func decode(data []byte) (map[string][]byte, error) {
	if len(data) < 8 || len(data) > maximumEnvelope || string(data[:4]) != "KPR1" {
		return nil, fmt.Errorf("invalid KPR1 envelope")
	}
	count := binary.LittleEndian.Uint32(data[4:])
	data = data[8:]
	if count > maximumFiles {
		return nil, fmt.Errorf("too many processor files")
	}
	files := make(map[string][]byte, int(count))
	folded := map[string]bool{}
	total := uint64(0)
	for range count {
		if len(data) < 6 {
			return nil, fmt.Errorf("truncated processor record")
		}
		nameSize := int(binary.LittleEndian.Uint16(data))
		size := uint64(binary.LittleEndian.Uint32(data[2:]))
		data = data[6:]
		if nameSize == 0 || nameSize > 256 || uint64(nameSize)+size > uint64(len(data)) {
			return nil, fmt.Errorf("invalid processor record bounds")
		}
		name := string(data[:nameSize])
		lower := strings.ToLower(name)
		if name == "." || !utf8.ValidString(name) || !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") || folded[lower] {
			return nil, fmt.Errorf("invalid or duplicate processor path")
		}
		total += size
		if total > maximumBytes {
			return nil, fmt.Errorf("processor input exceeds 16 MiB")
		}
		files[name] = data[nameSize : nameSize+int(size)]
		folded[lower] = true
		data = data[nameSize+int(size):]
	}
	if len(data) != 0 {
		return nil, fmt.Errorf("trailing processor bytes")
	}
	return files, nil
}

func encode(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	data := []byte{'K', 'P', 'R', '1', 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(data[4:], uint32(len(names)))
	for _, name := range names {
		contents := files[name]
		data = binary.LittleEndian.AppendUint16(data, uint16(len(name)))
		data = binary.LittleEndian.AppendUint32(data, uint32(len(contents)))
		data = append(data, name...)
		data = append(data, contents...)
	}
	return data
}
