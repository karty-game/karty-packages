package controllercheck

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type scenario struct {
	Name        string   `json:"name"`
	PhysicsOnly bool     `json:"physicsOnly"`
	Start       string   `json:"start"`
	Variants    []string `json:"variants"`
}
type configuration struct {
	Enabled  bool            `json:"enabled"`
	Cases    []scenario      `json:"cases"`
	Reset    string          `json:"reset"`
	Commands map[string]byte `json:"commands"`
}
type caseResult struct {
	Mode           string            `json:"mode"`
	Name           string            `json:"name"`
	Updates        uint64            `json:"updates"`
	Proposals      uint64            `json:"proposals"`
	Allocations    uint64            `json:"allocations"`
	AllocatedBytes uint64            `json:"allocatedBytes"`
	Commands       map[string]uint64 `json:"commands"`
}
type guest struct {
	ctx      context.Context
	module   api.Module
	config   configuration
	commands map[string]uint64
	velocity [3]float32
	capture  byte
}

func execute(ctx context.Context, wasm []byte) ([]caseResult, error) {
	data, err := fixtures.ReadFile("testdata/cases.json")
	if err != nil {
		return nil, err
	}
	var config configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	if !config.Enabled || len(config.Cases) != 4 {
		return nil, fmt.Errorf("expected four enabled controller scenarios")
	}
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	defer runtime.Close(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return nil, err
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, err
	}
	var active *guest
	host := runtime.NewHostModuleBuilder("karty")
	seen := make(map[string]bool)
	for _, imported := range compiled.ImportedFunctions() {
		module, name, ok := imported.Import()
		if !ok || module != "karty" || seen[name] {
			continue
		}
		seen[name] = true
		if name != "submit_commands" && name != "log" && name != "action" && name != "play_sound" {
			return nil, fmt.Errorf("unexpected karty import %s", name)
		}
		function := api.GoModuleFunc(func(_ context.Context, _ api.Module, stack []uint64) {
			for index := range stack {
				stack[index] = 0
			}
		})
		if name == "submit_commands" {
			function = func(_ context.Context, module api.Module, stack []uint64) {
				data, ok := module.Memory().Read(uint32(stack[0]), uint32(stack[1]))
				if !ok {
					panic("controller command memory out of bounds")
				}
				if err := active.submit(data); err != nil {
					panic(err)
				}
			}
		}
		host.NewFunctionBuilder().WithGoModuleFunction(function, imported.ParamTypes(), imported.ResultTypes()).Export(name)
	}
	if _, err := host.Instantiate(ctx); err != nil {
		return nil, err
	}
	var results []caseResult
	for _, plain := range []bool{false, true} {
		for _, physics := range []bool{false, true} {
			for _, fixture := range config.Cases {
				if fixture.PhysicsOnly && !physics {
					continue
				}
				caseContext, cancel := context.WithTimeout(ctx, 9*time.Second)
				current := &guest{ctx: caseContext, config: config, commands: make(map[string]uint64)}
				active = current
				result, err := runCase(runtime, compiled, current, fixture, plain, physics)
				cancel()
				if err != nil {
					return nil, fmt.Errorf("%s/plain=%t/physics=%t: %w", fixture.Name, plain, physics, err)
				}
				results = append(results, result)
			}
		}
	}
	if len(results) != 14 {
		return nil, fmt.Errorf("expected 14 cases, got %d", len(results))
	}
	return results, nil
}

func runCase(runtime wazero.Runtime, compiled wazero.CompiledModule, current *guest, fixture scenario, plain, physics bool) (result caseResult, err error) {
	// Assertions fail with their contract message; Wazero also propagates traps here.
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()
	current.module, err = runtime.InstantiateModule(current.ctx, compiled, wazero.NewModuleConfig().WithName("").WithStartFunctions())
	if err != nil {
		return result, err
	}
	defer current.module.Close(context.Background())
	current.call("_initialize")
	var mode uint64
	if plain {
		mode |= 2
	}
	if physics {
		mode |= 1
	}
	current.call("karty_use_first_person", mode)
	current.call("karty_register")
	current.call("initialize")
	require(current.state(2) == 1, "fixture camera identity")
	require(len(fixture.Variants) > 0, "scenario must contain event variants")
	variants := make([][]byte, len(fixture.Variants))
	for index, encoded := range fixture.Variants {
		variants[index] = decode(encoded)
	}
	update := func(frame uint64, batch []byte) {
		if batch == nil {
			batch = variants[frame%uint64(len(variants))]
		}
		require(len(batch) >= 24, "complete event header")
		binary.LittleEndian.PutUint64(batch[8:], frame)
		pointer := uint32(current.call("event_buffer"))
		require(current.module.Memory().Write(pointer, batch), "event buffer capacity")
		current.call("update", frame, uint64(len(batch)))
	}
	update(0, decode(fixture.Start))
	for frame := uint64(1); frame <= 100; frame++ {
		update(frame, nil)
	}
	if fixture.Name == "dual-touch" {
		require(current.commands["shape"] == 4, "both sticks have visible bases and thumbs")
		require(current.state(3) == 1, "touch mode is enabled")
	}
	if fixture.Name == "captured-mouse" {
		require(current.capture == 1, "scene gesture requests capture")
	}
	beforeCommands := make(map[string]uint64)
	for key, value := range current.commands {
		beforeCommands[key] = value
	}
	beforeUpdates, beforeProposals := current.state(0), current.state(1)
	beforeY := current.scalar(5)
	beforeAllocations, beforeBytes := current.call("karty_allocations"), current.call("karty_allocated_bytes")
	for frame := uint64(101); frame <= 1100; frame++ {
		update(frame, nil)
	}
	allocations := current.call("karty_allocations") - beforeAllocations
	allocatedBytes := current.call("karty_allocated_bytes") - beforeBytes
	require(current.state(0)-beforeUpdates == 1000, "all 1000 game updates execute")
	require(current.state(1)-beforeProposals == 1000, "BeforeMove executes exactly once per update")
	var expectedVelocity uint64
	if physics {
		expectedVelocity = 1000
	}
	require(current.commands["velocity"]-beforeCommands["velocity"] == expectedVelocity, "physics applies every proposal")
	require(current.commands["shape"] == beforeCommands["shape"], "steady movement reuses touch entities")
	switch fixture.Name {
	case "keyboard":
		if physics {
			require(near(current.velocity[1], 3.3, 0.000001), "keyboard sends walking speed")
		} else {
			require(near(current.scalar(5)-beforeY, 55, 0.01), "free camera walks every tick")
		}
	case "accepted-physics-pose":
		require(current.scalar(4) == 2 && current.scalar(5) == 3, "accepted pose precedes proposal and game update")
		require(near(current.velocity[1], 3.3, 0.000001), "accepted pose preserves walking speed")
	}
	if fixture.Name == "captured-mouse" || fixture.Name == "dual-touch" {
		require(current.commands["camera"]-beforeCommands["camera"] == 1000, "look changes camera every tick")
		require(finite(current.scalar(6)) && finite(current.scalar(7)) && math.Abs(float64(current.scalar(7))) <= 1.350001, "finite yaw and bounded pitch")
	}
	if fixture.Name == "dual-touch" {
		require(current.commands["transform"]-beforeCommands["transform"] == 2000, "both moving thumbs render every tick")
		if physics {
			require(math.Hypot(float64(current.velocity[0]), float64(current.velocity[1])) > 1, "left finger walks while right finger looks")
		}
	}
	require(allocations == 0, fmt.Sprintf("guest allocations: %d", allocations))
	require(allocatedBytes == 0, fmt.Sprintf("guest allocated bytes: %d", allocatedBytes))
	commands := make(map[string]uint64)
	for key := range current.config.Commands {
		commands[key] = current.commands[key] - beforeCommands[key]
	}
	controlAllocations, controlBytes := current.call("karty_allocations"), current.call("karty_allocated_bytes")
	current.call("karty_negative_control")
	require(current.call("karty_allocations") > controlAllocations, "allocation counter detects heap growth")
	require(current.call("karty_allocated_bytes") > controlBytes, "byte counter detects heap growth")
	update(1101, decode(current.config.Reset))
	require(current.capture == 0, "focus reset releases capture")
	if physics {
		require(current.velocity == [3]float32{}, "focus reset releases movement")
	}
	current.call("shutdown")
	style, movement := "hooks", "free"
	if plain {
		style = "plain"
	}
	if physics {
		movement = "physics"
	}
	return caseResult{"fps-" + movement + "-" + style, fixture.Name, 1000, 1000, allocations, allocatedBytes, commands}, nil
}

func (g *guest) call(name string, arguments ...uint64) uint64 {
	function := g.module.ExportedFunction(name)
	require(function != nil, "missing guest export "+name)
	result, err := function.Call(g.ctx, arguments...)
	if err != nil {
		panic(fmt.Errorf("%s: %w", name, err))
	}
	if len(result) == 0 {
		return 0
	}
	return result[0]
}
func (g *guest) state(index uint64) uint64   { return g.call("karty_first_person_state", index) }
func (g *guest) scalar(index uint64) float32 { return math.Float32frombits(uint32(g.state(index))) }
func require(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
func finite(value float32) bool { return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0) }
func near(actual, expected, tolerance float32) bool {
	return math.Abs(float64(actual-expected)) < float64(tolerance)
}
func decode(value string) []byte {
	result, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return result
}

func (g *guest) submit(data []byte) error {
	if len(data) < 24 || string(data[:4]) != "KTYB" || data[6] != 2 || data[7] != 0 || int(binary.LittleEndian.Uint32(data[20:])) != len(data)-24 {
		return fmt.Errorf("invalid controller command envelope")
	}
	offset := 24
	for count := binary.LittleEndian.Uint32(data[16:]); count > 0; count-- {
		if offset+3 > len(data) {
			return fmt.Errorf("truncated command header")
		}
		tag, length := data[offset], int(binary.LittleEndian.Uint16(data[offset+1:]))
		offset += 3
		if length > len(data)-offset {
			return fmt.Errorf("truncated command payload")
		}
		payload := data[offset : offset+length]
		for key, expected := range g.config.Commands {
			if tag == expected {
				g.commands[key]++
			}
		}
		if tag == g.config.Commands["velocity"] {
			if length != 20 || binary.LittleEndian.Uint32(payload) != 1 || binary.LittleEndian.Uint32(payload[16:]) != 1 {
				return fmt.Errorf("walking velocity must target camera body and preserve vertical momentum")
			}
			for index := range g.velocity {
				g.velocity[index] = math.Float32frombits(binary.LittleEndian.Uint32(payload[4+index*4:]))
				if !finite(g.velocity[index]) {
					return fmt.Errorf("nonfinite walking velocity")
				}
			}
		}
		if tag == g.config.Commands["capture"] {
			if length != 1 || payload[0] > 1 {
				return fmt.Errorf("invalid pointer capture")
			}
			g.capture = payload[0]
		}
		offset += length
	}
	if offset != len(data) {
		return fmt.Errorf("incomplete controller command envelope")
	}
	return nil
}
