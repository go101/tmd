package render

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"go101.org/nstd"
)

//go:embed tmd.wasm
var tmdWasm []byte

type Renderer struct {
	context context.Context

	runtime wazero.Runtime

	module api.Module
	memory api.Memory

	funcBufferOffset api.Function
	funcTmdToHtml    api.Function
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

// NewRenderer creates a Renderer. If it succeeds, call Renderer.Destroy method
// to release the resouce and Renderer.Render method to render a TMD document.
func NewRenderer() (*Renderer, error) {
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
	tmd_to_html := mod.ExportedFunction("tmd_to_html")
	memory := mod.Memory()

	return &Renderer{
		context: ctx,

		runtime: r,

		module: mod,
		memory: memory,

		funcBufferOffset: buffer_offset,
		funcTmdToHtml:    tmd_to_html,
	}, nil
}

// Destroy releases the resource allocated for a Renderer.
func (r *Renderer) Destroy() {
	r.runtime.Close(r.context)
}

// Render converts a TMD document into HTML.
func (r *Renderer) Render(tmdData []byte, fullHtml bool, supportCustomBlocks bool) (htmlData []byte, err error) {
	rets, err := r.funcBufferOffset.Call(r.context)
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad input offset: %d", int32(rets[0]))
	}

	bufferOffset := uint32(rets[0])

	maxInputLength, ok := r.memory.ReadUint32Le(bufferOffset)
	if !ok {
		return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (max input length)", bufferOffset)
	}

	if !r.memory.WriteByte(bufferOffset, 0) {
		return nil, fmt.Errorf("Memory.WriteByte(%d, %d) not okay", bufferOffset+4, 0)
	}

	if uint32(len(tmdData)) > maxInputLength {
		return nil, fmt.Errorf("Input length too large (%d > %d)", len(tmdData), maxInputLength)
	}

	if !r.memory.WriteUint32Le(bufferOffset+1, uint32(len(tmdData))) {
		return nil, fmt.Errorf("Memory.WriteUint32Le(%d, %d) not okay", bufferOffset, len(tmdData))
	}

	if !r.memory.Write(bufferOffset+5, tmdData) {
		return nil, fmt.Errorf("Memory.WriteString(%d, %s) not okay", bufferOffset+4, tmdData)
	}

	rets, err = r.funcTmdToHtml.Call(r.context, uint64(nstd.Btoi(fullHtml)), uint64(nstd.Btoi(supportCustomBlocks)))
	if err != nil {
		return nil, err
	}
	if int32(rets[0]) < 0 {
		return nil, fmt.Errorf("Bad output offset: %d", int32(rets[0]))
	}

	outputOffset := uint32(rets[0])

	outputLength, ok := r.memory.ReadUint32Le(outputOffset)
	if !ok {
		return nil, fmt.Errorf("Memory.ReadUint32Le(%d) not okay (output length)", bufferOffset)
	}

	output, ok := r.memory.Read(outputOffset+4, outputLength)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) not okay (output length)", outputOffset+4, outputLength)
	}

	return output, nil
}
