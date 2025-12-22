package input

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// MaxMemoryUsage defines the max memory used for all loaded raw testcases
const MaxMemoryUsage = 2 * 1024 * 1024 * 1024 // 2 GB

// Source takes a byteslice, and arguments can be pulled from it.
type Source struct {
	s         []byte
	i         int64 // current reading index
	exhausted bool
}

func NewSource(data []byte) *Source {
	return &Source{data, 0, false}
}

// IsExhausted returns true if we tried to read more data than this source
// could deliver.
func (s *Source) IsExhausted() bool {
	return s.exhausted
}

// Len returns the number of bytes of the unread portion of the data.
func (s *Source) Len() int {
	if s.i >= int64(len(s.s)) {
		return 0
	}
	return int(int64(len(s.s)) - s.i)
}

// Used returns the number of bytes already consumed.
func (s *Source) Used() int {
	return int(s.i)
}

// Read implements the io.Reader interface.
func (s *Source) Read(b []byte) (n int, err error) {
	if s.i >= int64(len(s.s)) {
		n, err = 0, io.EOF
	} else {
		n = copy(b, s.s[s.i:])
		s.i += int64(n)
	}
	if n < len(b) {
		s.exhausted = true
	}
	return n, err
}

// getBytes returns a slice of size bytes, as a direct reference if possible.
func (s *Source) getBytes(size int) []byte {
	if end := int(s.i) + size; end < len(s.s) { // Fast-path, no-copy deliver
		pos := s.i
		s.i += int64(size)
		return s.s[pos:end]
	}
	// Slow path
	buf := make([]byte, size)
	s.Read(buf)
	return buf
}

// readInt reads a signed integer from the source
func (s *Source) readInt(num reflect.Kind) int64 {
	switch num {
	case reflect.Int8:
		return int64(int8(s.getBytes(1)[0]))
	case reflect.Int16:
		return int64(int16(binary.BigEndian.Uint16(s.getBytes(2))))
	case reflect.Int32:
		return int64(int32(binary.BigEndian.Uint32(s.getBytes(4))))
	case reflect.Int64, reflect.Int:
		return int64(binary.BigEndian.Uint64(s.getBytes(8)))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// readUint reads an unsigned integer from the source
func (s *Source) readUint(num reflect.Kind) uint64 {
	switch num {
	case reflect.Uint8:
		return uint64(uint8(s.getBytes(1)[0]))
	case reflect.Uint16:
		return uint64(binary.BigEndian.Uint16(s.getBytes(2)))
	case reflect.Uint32:
		return uint64(binary.BigEndian.Uint32(s.getBytes(4)))
	case reflect.Uint, reflect.Uint64:
		return binary.BigEndian.Uint64(s.getBytes(8))
	}
	panic(fmt.Sprintf("unsupported type: %v", num))
}

// FillAndCall fills the argument for the given ff (which is supposed to be a function),
// and then invokes the function.
// It returns 'true' if the function was invoked. A return-value of false means
// that the method was not invoked: probably because of insufficient input.
func (s *Source) FillAndCall(ff any, arg0 reflect.Value) (ok bool) {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	args := s.createArgs(ff, arg0)
	fn.Call(args)
	return true
}

func (s *Source) CreateGoTestcase(ff any, arg0 reflect.Value) string {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	
	// Use original createArgs for backward compatibility
	args := s.createArgs(ff, arg0)

	var sb strings.Builder
	sb.WriteString("go test fuzz v1\n")
	for i, arg := range args {
		switch arg.Kind() {
		case reflect.Ptr: // *testing.T
			// skip
			continue

		case reflect.String:
			// Properly escape as a Go string literal.
			sb.WriteString("string(")
			sb.WriteString(strconv.Quote(arg.String()))
			sb.WriteString(")")

		case reflect.Slice:
			// Only []byte is supported; escape contents as Go string literal.
			if arg.Type().Elem().Kind() == reflect.Uint8 {
				sb.WriteString("[]byte(")
				sb.WriteString(strconv.Quote(string(arg.Bytes())))
				sb.WriteString(")")
			} else {
				panic(fmt.Sprintf("unsupported slice elem type: %v", arg.Type().Elem()))
			}

		default:
			// Other primitives: emit kind(value).
			// Use the underlying value printed with %v, which is fine for ints/floats/bools.
			sb.WriteString(fmt.Sprintf("%s(%v)", arg.Kind(), arg.Interface()))
		}

		if i < method.NumIn()-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// CreateGoTestcaseWithBoundaries is like CreateGoTestcase but uses the improved
// createArgsWithBoundaries algorithm that preserves semantic boundaries better
// for multi-parameter fuzzers. Use this for fuzzers with many parameters where
// the model/DSL data should be preserved in the first parameter.
func (s *Source) CreateGoTestcaseWithBoundaries(ff any, arg0 reflect.Value) string {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	
	// Use the SAME algorithm as FillAndCall for consistency
	args := s.createArgs(ff, arg0)

	var sb strings.Builder
	sb.WriteString("go test fuzz v1\n")
	
	for i, arg := range args {
		switch arg.Kind() {
		case reflect.Ptr: // *testing.T
			// skip
			continue

		case reflect.String:
			// Properly escape as a Go string literal.
			sb.WriteString("string(")
			sb.WriteString(strconv.Quote(arg.String()))
			sb.WriteString(")")

		case reflect.Slice:
			// Only []byte is supported; escape contents as Go string literal.
			if arg.Type().Elem().Kind() == reflect.Uint8 {
				sb.WriteString("[]byte(")
				sb.WriteString(strconv.Quote(string(arg.Bytes())))
				sb.WriteString(")")
			} else {
				panic(fmt.Sprintf("unsupported slice elem type: %v", arg.Type().Elem()))
			}

		default:
			// Other primitives: emit kind(value).
			// Use the underlying value printed with %v, which is fine for ints/floats/bools.
			sb.WriteString(fmt.Sprintf("%s(%v)", arg.Kind(), arg.Interface()))
		}

		if i < method.NumIn()-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (s *Source) createArgs(ff any, arg0 reflect.Value) []reflect.Value {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	args := make([]reflect.Value, method.NumIn())
	args[0] = arg0
	var dynamic []int
	// Fill all fixed-size arguments first, then dynamic-sized fields.
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			args[i] = s.fillArg(v, 0)
		} else { // dynamic or panic later
			dynamic = append(dynamic, i)
		}
	}
	// Second loop to fill dynamic-sized stuff
	// For filling the dynamic fields.
	// If we have only one field, it should get all the remaining input.
	// If we have N, then,
	// 1. Read N bytes [b1, b2, b3 .. bn] .
	// 2. Let the relative weights of b determine how much of the
	//    remaining input that field n gets
	weights := s.getBytes(len(dynamic))
	sum := 0
	for _, v := range weights {
		sum += int(v)
	}
	bytesLeft := s.Len()
	for i, argNum := range dynamic {
		if i == len(dynamic)-1 { // last element, it get's all that if left
			args[argNum] = s.fillArg(method.In(argNum), s.Len())
			break
		}
		var argSize = bytesLeft / len(dynamic)
		if sum > 0 {
			argSize = (bytesLeft * int(weights[i])) / sum
		}
		args[argNum] = s.fillArg(method.In(argNum), argSize)
	}
	return args
}

// createArgsWithBoundaries is a FIXED version of createArgs that doesn't consume
// input bytes for weight calculation. Instead uses deterministic weights based on 
// parameter count, preserving all input data for actual parameters.
func (s *Source) createArgsWithBoundaries(ff any, arg0 reflect.Value) []reflect.Value {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	
	args := make([]reflect.Value, method.NumIn())
	args[0] = arg0
	
	// First pass: calculate total bytes needed for fixed params
	fixedBytesNeeded := 0
	var dynamicIndices []int
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			fixedBytesNeeded += sizeOfType(v.Kind())
		} else { // dynamic (string, slice, etc.)
			dynamicIndices = append(dynamicIndices, i)
		}
	}
	
	numDynamic := len(dynamicIndices)
	// FIXED: Calculate bytes available for dynamic params AFTER accounting for fixed params
	bytesForDynamic := s.Len() - fixedBytesNeeded
	if bytesForDynamic < 0 {
		bytesForDynamic = 0
	}
	
	// Calculate sizes for dynamic params (50% to first, rest split equally)
	dynamicSizes := make([]int, numDynamic)
	remaining := bytesForDynamic
	for i := 0; i < numDynamic; i++ {
		if i == numDynamic-1 {
			// Last param gets all remaining
			dynamicSizes[i] = remaining
		} else if i == 0 && numDynamic > 1 {
			// First param gets 50%
			dynamicSizes[i] = bytesForDynamic / 2
			remaining -= dynamicSizes[i]
		} else {
			// Other params split equally
			remainingParams := numDynamic - i
			dynamicSizes[i] = remaining / remainingParams
			remaining -= dynamicSizes[i]
		}
	}
	
	// Second pass: fill parameters IN ORDER (respecting their position in signature)
	dynamicIdx := 0
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			// Fixed params consume bytes in-order from the input stream
			args[i] = s.fillArg(v, 0)
		} else { // dynamic
			size := dynamicSizes[dynamicIdx]
			args[i] = s.fillArg(v, size)
			dynamicIdx++
		}
	}
	
	return args
}

// sizeOfType returns the number of bytes a fixed-size type consumes
func sizeOfType(k reflect.Kind) int {
	switch k {
	case reflect.Int8, reflect.Uint8, reflect.Bool:
		return 1
	case reflect.Int16, reflect.Uint16:
		return 2
	case reflect.Int32, reflect.Uint32, reflect.Float32:
		return 4
	case reflect.Int64, reflect.Uint64, reflect.Float64, reflect.Int, reflect.Uint:
		return 8
	default:
		return 0
	}
}

func (s *Source) fillArg(v reflect.Type, max int) reflect.Value {
	newElem := reflect.New(v).Elem()
	switch k := v.Kind(); k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		newElem.SetInt(s.readInt(k))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		newElem.SetUint(s.readUint(k))
	case reflect.Float32:
		newElem.Set(reflect.ValueOf(math.Float32frombits(uint32(s.readUint(reflect.Uint32)))))
	case reflect.Float64:
		newElem.Set(reflect.ValueOf(math.Float64frombits(s.readUint(reflect.Uint64))))
	case reflect.Bool:
		newElem.Set(reflect.ValueOf(s.readUint(reflect.Uint8)&0x1 != 0))
	case reflect.String:
		newElem.SetString(string(s.getBytes(max)))
	case reflect.Slice:
		if v.Elem().Kind() == reflect.Uint8 { // []byte
			newElem.SetBytes(s.getBytes(max))
		} else {
			panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
	}
	return newElem
}

func ParseGoTestcase(testcase string) ([]byte, error) {
	lines := strings.Split(testcase, "\n")
	if len(lines) == 0 || lines[0] != "go test fuzz v1" {
		return nil, fmt.Errorf("invalid test case header")
	}

	var buf bytes.Buffer

	// Parse to match the format expected by createArgs:
	// [fixed params][weight bytes][dynamic params]
	
	// First pass: separate fixed and dynamic params
	type param struct {
		kind  string
		value string
		index int
	}
	var allParams []param
	for i, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		kind, val, err := parseTypedValue(line)
		if err != nil {
			return nil, err
		}
		
		allParams = append(allParams, param{kind, val, i})
	}
	
	// Separate into fixed and dynamic
	var fixedParams []param
	var dynamicParams []param
	for _, p := range allParams {
		isDynamic := p.kind == "string" || p.kind == "[]byte"
		if isDynamic {
			dynamicParams = append(dynamicParams, p)
		} else {
			fixedParams = append(fixedParams, p)
		}
	}
	
	// Write fixed params in order
	for _, p := range fixedParams {
		err := writeStaticValue(&buf, p.kind, p.value)
		if err != nil {
			return nil, err
		}
	}

	// Write weight bytes for dynamic params
	numDynamic := len(dynamicParams)
	if numDynamic > 0 {
		// Calculate weights that will produce the exact lengths
		var lengths []int
		totalDataBytes := 0
		for _, p := range dynamicParams {
			length := len(p.value)
			lengths = append(lengths, length)
			totalDataBytes += length
		}
		
		weights := calculatePerfectWeights(lengths, totalDataBytes)
		for _, w := range weights {
			buf.WriteByte(w)
		}
		
		// Write dynamic param data
		for _, p := range dynamicParams {
			buf.Write([]byte(p.value))
		}
	}

	return buf.Bytes(), nil
}

func writeStaticValue(buf *bytes.Buffer, kind, val string) error {
	switch kind {
	case "int8":
		n, err := strconv.ParseInt(val, 10, 8)
		if err != nil {
			return err
		}
		buf.WriteByte(byte(int8(n)))
	case "int16":
		n, err := strconv.ParseInt(val, 10, 16)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, int16(n))
	case "int32":
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, int32(n))
	case "int64", "int":
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, uint64(n))
	case "uint8":
		n, err := strconv.ParseUint(val, 10, 8)
		if err != nil {
			return err
		}
		buf.WriteByte(byte(n))
	case "uint16":
		n, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, uint16(n))
	case "uint32":
		n, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, uint32(n))
	case "uint64", "uint":
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		binary.Write(buf, binary.BigEndian, n)
	case "float32":
		f, err := strconv.ParseFloat(val, 32)
		if err != nil {
			return err
		}
		bits := math.Float32bits(float32(f))
		binary.Write(buf, binary.BigEndian, bits)
	case "float64":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		bits := math.Float64bits(f)
		binary.Write(buf, binary.BigEndian, bits)
	case "bool":
		if val == "true" {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
	default:
		return fmt.Errorf("unsupported kind: %s", kind)
	}
	return nil
}

func parseTypedValue(line string) (kind string, value string, err error) {
	idx := strings.Index(line, "(")
	if idx < 0 || !strings.HasSuffix(line, ")") {
		return "", "", fmt.Errorf("malformed line: %q", line)
	}
	kind = line[:idx]
	value = line[idx+1 : len(line)-1]
	if kind == "string" || kind == "[]byte" {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", "", fmt.Errorf("failed to unquote string/[]byte: %w", err)
		}
		return kind, unquoted, nil
	}
	return kind, value, nil
}

// calculatePerfectWeights computes weight bytes that cause libFuzzer to allocate
// the specified lengths. We don't need the EXACT original weights - just any weights
// that produce the same output lengths.
//
// LibFuzzer's allocation algorithm (from createArgs):
//   for i = 0 to N-2:
//     size[i] = (bytesLeft * weight[i]) / sumWeights
//     bytesLeft -= size[i]  
//   size[N-1] = bytesLeft
//
// Simple approach: Make weights proportional to desired lengths.
// This naturally produces similar allocations due to proportional division.
func calculatePerfectWeights(lengths []int, totalDataBytes int) []byte {
	n := len(lengths)
	if n == 0 {
		return nil
	}
	
	if totalDataBytes == 0 {
		// All empty - weights don't matter
		weights := make([]byte, n)
		for i := range weights {
			weights[i] = 0x80
		}
		return weights
	}
	
	if n == 1 {
		// Single param gets everything
		return []byte{0xFF}
	}
	
	// Simple proportional approach: make weights match desired lengths
	// Start with lengths as weights, then scale to fit [0-255] byte range
	result := make([]byte, n)
	
	// Find max length for scaling
	maxLen := 0
	for _, l := range lengths {
		if l > maxLen {
			maxLen = l
		}
	}
	
	if maxLen == 0 {
		// All zero-length params - use equal weights
		for i := range result {
			result[i] = 128
		}
		return result
	}
	
	// Scale lengths proportionally to fit [0-255]
	if maxLen <= 255 {
		// Direct mapping
		for i, l := range lengths {
			result[i] = byte(l)
			if result[i] == 0 && l > 0 {
				result[i] = 1
			}
		}
	} else {
		// Scale down proportionally
		for i, l := range lengths {
			scaled := (l * 255) / maxLen
			if scaled == 0 && l > 0 {
				scaled = 1
			}
			result[i] = byte(scaled)
		}
	}
	
	// Iteratively refine weights to match exact lengths using simulation
	// Simulate libFuzzer's exact allocation algorithm and adjust ONE weight per iteration
	for iteration := 0; iteration < 500; iteration++ {
		sum := 0
		for _, w := range result {
			sum += int(w)
		}
		if sum == 0 {
			sum = 1
		}
		
		// Simulate sequential allocation to see what we'd get with current weights
		bytesLeft := totalDataBytes
		allocations := make([]int, n)
		
		for i := 0; i < n-1; i++ {
			allocations[i] = (bytesLeft * int(result[i])) / sum
			bytesLeft -= allocations[i]
		}
		allocations[n-1] = bytesLeft // Last param gets remainder
		
		// Check if perfect
		perfect := true
		firstMismatch := -1
		for i := 0; i < n; i++ {
			if allocations[i] != lengths[i] {
				perfect = false
				if firstMismatch == -1 {
					firstMismatch = i
				}
			}
		}
		
		if perfect {
			break // Success!
		}
		
		// Adjust the first mismatched weight
		if firstMismatch >= 0 && firstMismatch < n-1 {
			if allocations[firstMismatch] < lengths[firstMismatch] && result[firstMismatch] < 255 {
				result[firstMismatch]++
			} else if allocations[firstMismatch] > lengths[firstMismatch] && result[firstMismatch] > 0 {
				result[firstMismatch]--
			}
		} else if firstMismatch == n-1 {
			// Last param is wrong - need to adjust an earlier weight
			// Try adjusting the last adjustable weight
			if n >= 2 {
				if allocations[n-1] < lengths[n-1] && result[n-2] > 0 {
					result[n-2]--  // Give less to second-to-last
				} else if allocations[n-1] > lengths[n-1] && result[n-2] < 255 {
					result[n-2]++  // Give more to second-to-last
				}
			}
		}
	}
	
	// Final safety: ensure no zero weights for non-empty params
	for i := 0; i < n; i++ {
		if result[i] == 0 && lengths[i] > 0 {
			result[i] = 1
		}
	}
	
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ZipCorpusFromGoFuzzCases merges all fuzzing testcases from inputDir into a zip file.
// If outputName.zip exists in $OUT, it merges into it.
// If verbose is true, prints summary of added/skipped files.
// Memory usage is capped and streaming is used to reduce pressure.
func ZipCorpusFromGoFuzzCases(inputDir, outputName string, verbose bool) error {
	zipFileName := outputName + ".zip"

	outDir := os.Getenv("OUT")
	if outDir == "" {
		return fmt.Errorf("environment variable $OUT is not set")
	}
	existingZipPath := filepath.Join(outDir, zipFileName)
	tmpZipPath := filepath.Join(os.TempDir(), zipFileName)

	// Create temp zip file
	zipFile, err := os.Create(tmpZipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	// 1. Stream existing zip contents (if exists)
	if _, err := os.Stat(existingZipPath); err == nil {
		r, err := zip.OpenReader(existingZipPath)
		if err != nil {
			return fmt.Errorf("failed to open existing zip: %w", err)
		}
		defer r.Close()

		for _, f := range r.File {
			src, err := f.Open()
			if err != nil {
				continue
			}
			dst, err := zipWriter.Create(f.Name)
			if err != nil {
				src.Close()
				return fmt.Errorf("failed to copy existing file %s into zip: %w", f.Name, err)
			}
			_, err = io.Copy(dst, src)
			src.Close()
			if err != nil {
				return fmt.Errorf("failed to stream existing file %s: %w", f.Name, err)
			}
		}
	}

	// 2. Process inputDir files one at a time
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %w", err)
	}

	var memUsed int64
	var filesAdded, filesSkipped int
	var skippedFilenames []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		fullPath := filepath.Join(inputDir, name)

		content, err := os.ReadFile(fullPath)
		if err != nil {
			filesSkipped++
			skippedFilenames = append(skippedFilenames, name)
			continue
		}

		raw, err := ParseGoTestcase(string(content))
		if err != nil {
			filesSkipped++
			skippedFilenames = append(skippedFilenames, name)
			continue
		}

		memUsed += int64(len(raw))
		if memUsed > MaxMemoryUsage {
			filesSkipped++
			skippedFilenames = append(skippedFilenames, name)
			continue
		}

		dst, err := zipWriter.Create(name)
		if err != nil {
			return fmt.Errorf("failed to create entry in zip: %w", err)
		}
		_, err = dst.Write(raw)
		if err != nil {
			return fmt.Errorf("failed to write raw data for %s: %w", name, err)
		}
		filesAdded++
	}

	// Finalize zip
	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("failed to finalize zip: %w", err)
	}

	// Use copy + remove instead of rename to handle cross-device scenarios
	err = copyFile(tmpZipPath, existingZipPath)
	if err != nil {
		return fmt.Errorf("failed to copy zip file to $OUT: %w", err)
	}
	os.Remove(tmpZipPath) // Clean up temp file

	// Verbose output
	if verbose {
		fmt.Printf("[ZipCorpusFromGoFuzzCases] Added: %d files\n", filesAdded)
		fmt.Printf("[ZipCorpusFromGoFuzzCases] Skipped: %d files\n", filesSkipped)
		if filesSkipped > 0 {
			fmt.Println("[ZipCorpusFromGoFuzzCases] Skipped files:")
			for _, name := range skippedFilenames {
				fmt.Printf("  - %s\n", name)
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}