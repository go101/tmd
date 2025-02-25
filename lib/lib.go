package lib

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"go101.org/nstd"
)

//go:embed tmd.wasm
var tmdWasm []byte

// All methods of TmdLib are not concurently safe.
type TmdLib struct {
	context context.Context

	runtime wazero.Runtime

	module api.Module
	memory api.Memory

	funcBufferOffset api.Function
	funcGetVersion   api.Function
	funcTmdToHtml    api.Function
	funcTmdFormat    api.Function

	tmdDataValid atomic.Bool
}

func printMessages(_ context.Context, m api.Module, offset, byteCount, offset2, byteCount2 uint32, extraInt32 int32) {
	buf, ok := m.Memory().Read(offset, byteCount)
	if !ok {
		log.Panicf("Memory.Read(%d, %d) out of range (1)", offset, byteCount)
	}
	buf2, ok := m.Memory().Read(offset2, byteCount2)
	if !ok {
		log.Panicf("Memory.Read(%d, %d) out of range (2)", offset2, byteCount2)
	}
	fmt.Printf("%s%s%d\n", buf, buf2, extraInt32)
}

// NewTmdLib creates a TmdLib. If it succeeds, call TmdLib.Destroy method
// to release the resouce and TmdLib.Render method to render a TMD document.
func NewTmdLib() (*TmdLib, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Instantiate a Go-defined module named "env" that exports a function to
	// log to the console.
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(printMessages).Export("print").
		Instantiate(ctx)
	if err != nil {
		r.Close(ctx)
		return nil, err
	}

	// Instantiate a WebAssembly module that imports the "log" function defined
	// in "env" and exports "memory" and functions we'll use in this example.
	mod, err := r.InstantiateWithConfig(ctx, tmdWasm,
		wazero.NewModuleConfig().WithStdout(os.Stdout).WithStderr(os.Stderr))
	if err != nil {
		r.Close(ctx)
		return nil, err
	}

	// Get references to WebAssembly functions we'll use in this example.
	buffer_offset := mod.ExportedFunction("buffer_offset")
	get_version := mod.ExportedFunction("get_version")
	tmd_to_html := mod.ExportedFunction("tmd_to_html")
	tmd_format := mod.ExportedFunction("tmd_format")
	memory := mod.Memory()

	return &TmdLib{
		context: ctx,

		runtime: r,

		module: mod,
		memory: memory,

		funcBufferOffset: buffer_offset,
		funcGetVersion:   get_version,
		funcTmdToHtml:    tmd_to_html,
		funcTmdFormat:    tmd_format,
	}, nil
}

// Destroy releases the resource allocated for a TmdLib.
func (lib *TmdLib) Destroy() {
	lib.runtime.Close(lib.context)
}

// Version returns the version of library.
func (lib *TmdLib) Version() (version []byte, err error) {
	rets, err := lib.funcBufferOffset.Call(lib.context)
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad input offset: %d", int32(rets[0]))
	}

	//bufferOffset := uint32(rets[0])

	//maxInputLength, ok := lib.memory.ReadUint32Le(bufferOffset)
	//if !ok {
	//	return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (max input length)", bufferOffset)
	//}

	lib.tmdDataValid.Store(false)

	rets, err = lib.funcGetVersion.Call(lib.context)
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad input offset: %d", int32(rets[0]))
	}

	versionOffset := uint32(rets[0])
	// bufferOffset == versionOffset

	versionLength, ok := lib.memory.ReadUint32Le(versionOffset)
	if !ok {
		return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (version length)", versionOffset)
	}

	version, ok = lib.memory.Read(versionOffset+4, versionLength)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) not okay (version length)", versionOffset+4, versionLength)
	}

	return version, nil
}

// WriteInputTmd prepares the input TMD data for later using.
func (lib *TmdLib) WriteInputTmd(tmdData []byte) error {
	rets, err := lib.funcBufferOffset.Call(lib.context)
	if err != nil {
		return err
	}
	if int32(rets[0]) < 0 {
		return fmt.Errorf("Bad input offset: %d", int32(rets[0]))
	}

	bufferOffset := uint32(rets[0])

	maxInputLength, ok := lib.memory.ReadUint32Le(bufferOffset)
	if !ok {
		return fmt.Errorf("Memory.ReadUint32Le(%d) not okay (max input length)", bufferOffset)
	}

	if !lib.memory.WriteByte(bufferOffset, 0) {
		return fmt.Errorf("Memory.WriteByte(%d, %d) not okay", bufferOffset+4, 0)
	}

	if uint32(len(tmdData)) > maxInputLength {
		return fmt.Errorf("Input length too large (%d > %d)", len(tmdData), maxInputLength)
	}

	if !lib.memory.WriteUint32Le(bufferOffset+1, uint32(len(tmdData))) {
		return fmt.Errorf("Memory.WriteUint32Le(%d, %d) not okay", bufferOffset, len(tmdData))
	}

	if !lib.memory.Write(bufferOffset+5, tmdData) {
		return fmt.Errorf("Memory.WriteString(%d, %s) not okay", bufferOffset+4, tmdData)
	}

	lib.tmdDataValid.Store(true)

	return nil
}

// GenerateHtml converts the written input TMD into HTML.
// Options:
//   - fullHtml: whether or not generate full HTML page.
//     To generate HTML pieces for embedding purpose, pass false.
//   - supportCustomBlocks: whether or not support custom blocks.
func (lib *TmdLib) GenerateHtml(fullHtml bool, supportCustomBlocks bool) (htmlData []byte, err error) {
	if !lib.tmdDataValid.Load() {
		return nil, errors.New("input TMD is invalid")
	}

	rets, err := lib.funcTmdToHtml.Call(lib.context, uint64(nstd.Btoi(fullHtml)), uint64(nstd.Btoi(supportCustomBlocks)))
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad output offset: %d", int32(rets[0]))
	}

	outputOffset := uint32(rets[0])

	outputLength, ok := lib.memory.ReadUint32Le(outputOffset)
	if !ok {
		return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (output length)", outputOffset)
	}

	output, ok := lib.memory.Read(outputOffset+4, outputLength)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) not okay (output length)", outputOffset+4, outputLength)
	}

	return output, nil
}

// GenerateHtmlFromTmd converts a TMD document into HTML.
// The options are the same as [GenerateHtml].
func (lib *TmdLib) GenerateHtmlFromTmd(tmdData []byte, fullHtml bool, supportCustomBlocks bool) (htmlData []byte, err error) {
	err = lib.WriteInputTmd(tmdData)
	if err != nil {
		return nil, fmt.Errorf("Write input TMD error (%w)", err)
	}

	return lib.GenerateHtml(fullHtml, supportCustomBlocks)
}

// Format formats the written input TMD.
func (lib *TmdLib) Format() (formattedData []byte, err error) {
	if !lib.tmdDataValid.Load() {
		return nil, errors.New("input TMD is invalid")
	}

	rets, err := lib.funcTmdFormat.Call(lib.context)
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad format offset: %d", int32(rets[0]))
	}

	formatOffset := uint32(rets[0])

	formatLength, ok := lib.memory.ReadUint32Le(formatOffset)
	if !ok {
		return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (format length)", formatOffset)
	}
	if formatLength == 0 {
		return nil, nil
	}

	formatted, ok := lib.memory.Read(formatOffset+4, formatLength)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) not okay (format length)", formatOffset+4, formatLength)
	}

	return formatted, nil
}

// FormatTmd formats a TMD document.
func (lib *TmdLib) FormatTmd(tmdData []byte) (formattedData []byte, err error) {
	err = lib.WriteInputTmd(tmdData)
	if err != nil {
		return nil, fmt.Errorf("Write input TMD error (%w)", err)
	}

	return lib.Format()
}
