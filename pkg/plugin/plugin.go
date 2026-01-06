// Package plugin provides a modular plugin system for go-carbon.
//
// The plugin system allows developers to:
//   - Create custom event processors as plugins
//   - Register decoders dynamically
//   - Extend pipeline functionality without modifying core code
//   - Load plugins at runtime
//
// Example plugin implementation:
//
//	type MyPlugin struct{}
//
//	func (p *MyPlugin) Name() string { return "my-plugin" }
//	func (p *MyPlugin) Version() string { return "1.0.0" }
//	func (p *MyPlugin) Initialize(ctx context.Context) error { return nil }
//	func (p *MyPlugin) Shutdown(ctx context.Context) error { return nil }
//
//	// Register the plugin
//	registry := plugin.NewRegistry()
//	registry.Register(myPlugin)
package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/log"
)

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	// Name returns the unique name of this plugin.
	Name() string

	// Version returns the version of this plugin.
	Version() string

	// Description returns a description of what this plugin does.
	Description() string

	// Initialize initializes the plugin.
	// Called once when the plugin is registered.
	Initialize(ctx context.Context) error

	// Shutdown gracefully shuts down the plugin.
	// Called when the application is shutting down.
	Shutdown(ctx context.Context) error
}

// EventProcessorPlugin extends Plugin with event processing capabilities.
type EventProcessorPlugin interface {
	Plugin

	// ProcessEvent processes a decoded event.
	// Returns true if the event was handled.
	ProcessEvent(ctx context.Context, event *decoder.Event) (bool, error)

	// GetEventTypes returns the event types this plugin handles.
	// Empty slice means it handles all events.
	GetEventTypes() []string
}

// DecoderPlugin extends Plugin with decoder registration.
type DecoderPlugin interface {
	Plugin

	// GetDecoders returns the decoders provided by this plugin.
	GetDecoders() []decoder.Decoder

	// GetLogProcessors returns custom log processors provided by this plugin.
	GetLogProcessors() []log.LogProcessor
}

// FullPlugin combines both event processing and decoder capabilities.
type FullPlugin interface {
	Plugin
	EventProcessorPlugin
	DecoderPlugin
}

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	Tags        []string
}

// Registry manages plugin registration and lifecycle.
type Registry struct {
	mu                  sync.RWMutex
	plugins             map[string]Plugin
	eventProcessors     []EventProcessorPlugin
	decoderPlugins      []DecoderPlugin
	decoderRegistry     *decoder.Registry
	logParserProcessors []log.LogProcessor
	initialized         bool
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins:             make(map[string]Plugin),
		eventProcessors:     make([]EventProcessorPlugin, 0),
		decoderPlugins:      make([]DecoderPlugin, 0),
		decoderRegistry:     decoder.NewRegistry(),
		logParserProcessors: make([]log.LogProcessor, 0),
	}
}

// Register registers a plugin.
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.plugins[name] = plugin

	// Register event processor if applicable
	if ep, ok := plugin.(EventProcessorPlugin); ok {
		r.eventProcessors = append(r.eventProcessors, ep)
	}

	// Register decoder plugin if applicable
	if dp, ok := plugin.(DecoderPlugin); ok {
		r.decoderPlugins = append(r.decoderPlugins, dp)

		// Register decoders from plugin
		for _, dec := range dp.GetDecoders() {
			r.decoderRegistry.Register(name+":"+dec.GetName(), dec)
		}

		// Register log processors from plugin
		for _, proc := range dp.GetLogProcessors() {
			r.logParserProcessors = append(r.logParserProcessors, proc)
		}
	}

	return nil
}

// MustRegister registers a plugin and panics on error.
func (r *Registry) MustRegister(plugin Plugin) {
	if err := r.Register(plugin); err != nil {
		panic(fmt.Sprintf("failed to register plugin: %v", err))
	}
}

// Initialize initializes all registered plugins.
func (r *Registry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return fmt.Errorf("registry already initialized")
	}

	for name, plugin := range r.plugins {
		if err := plugin.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
		}
	}

	r.initialized = true
	return nil
}

// Shutdown shuts down all registered plugins.
func (r *Registry) Shutdown(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, plugin := range r.plugins {
		if err := plugin.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("plugin %s shutdown error: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// Get retrieves a plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return plugin, exists
}

// GetEventProcessors returns all event processor plugins.
func (r *Registry) GetEventProcessors() []EventProcessorPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.eventProcessors
}

// GetDecoderRegistry returns the decoder registry populated with plugin decoders.
func (r *Registry) GetDecoderRegistry() *decoder.Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.decoderRegistry
}

// GetLogProcessors returns all log processors from plugins.
func (r *Registry) GetLogProcessors() []log.LogProcessor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.logParserProcessors
}

// ProcessEvent processes an event through all registered event processor plugins.
func (r *Registry) ProcessEvent(ctx context.Context, event *decoder.Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, processor := range r.eventProcessors {
		// Check if processor handles this event type
		eventTypes := processor.GetEventTypes()
		if len(eventTypes) > 0 {
			handled := false
			for _, eventType := range eventTypes {
				if eventType == event.Name {
					handled = true
					break
				}
			}
			if !handled {
				continue
			}
		}

		// Process the event
		if _, err := processor.ProcessEvent(ctx, event); err != nil {
			return fmt.Errorf("plugin %s failed to process event: %w", processor.Name(), err)
		}
	}

	return nil
}

// ListPlugins returns information about all registered plugins.
func (r *Registry) ListPlugins() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		infos = append(infos, PluginInfo{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Description: plugin.Description(),
		})
	}

	return infos
}

// BasePlugin provides a basic implementation of the Plugin interface.
// Plugins can embed this to get default implementations.
type BasePlugin struct {
	name        string
	version     string
	description string
}

// NewBasePlugin creates a new BasePlugin.
func NewBasePlugin(name, version, description string) *BasePlugin {
	return &BasePlugin{
		name:        name,
		version:     version,
		description: description,
	}
}

// Name implements Plugin interface.
func (p *BasePlugin) Name() string {
	return p.name
}

// Version implements Plugin interface.
func (p *BasePlugin) Version() string {
	return p.version
}

// Description implements Plugin interface.
func (p *BasePlugin) Description() string {
	return p.description
}

// Initialize implements Plugin interface with no-op.
func (p *BasePlugin) Initialize(ctx context.Context) error {
	return nil
}

// Shutdown implements Plugin interface with no-op.
func (p *BasePlugin) Shutdown(ctx context.Context) error {
	return nil
}

// EventProcessorBase provides base implementation for event processors.
type EventProcessorBase struct {
	*BasePlugin
	eventTypes  []string
	processFunc func(context.Context, *decoder.Event) (bool, error)
}

// NewEventProcessorPlugin creates a new event processor plugin.
func NewEventProcessorPlugin(
	name, version, description string,
	eventTypes []string,
	processFunc func(context.Context, *decoder.Event) (bool, error),
) *EventProcessorBase {
	return &EventProcessorBase{
		BasePlugin:  NewBasePlugin(name, version, description),
		eventTypes:  eventTypes,
		processFunc: processFunc,
	}
}

// ProcessEvent implements EventProcessorPlugin interface.
func (p *EventProcessorBase) ProcessEvent(ctx context.Context, event *decoder.Event) (bool, error) {
	return p.processFunc(ctx, event)
}

// GetEventTypes implements EventProcessorPlugin interface.
func (p *EventProcessorBase) GetEventTypes() []string {
	return p.eventTypes
}

// DecoderPluginBase provides base implementation for decoder plugins.
type DecoderPluginBase struct {
	*BasePlugin
	decoders      []decoder.Decoder
	logProcessors []log.LogProcessor
}

// NewDecoderPlugin creates a new decoder plugin.
func NewDecoderPlugin(
	name, version, description string,
	decoders []decoder.Decoder,
	logProcessors []log.LogProcessor,
) *DecoderPluginBase {
	return &DecoderPluginBase{
		BasePlugin:    NewBasePlugin(name, version, description),
		decoders:      decoders,
		logProcessors: logProcessors,
	}
}

// GetDecoders implements DecoderPlugin interface.
func (p *DecoderPluginBase) GetDecoders() []decoder.Decoder {
	return p.decoders
}

// GetLogProcessors implements DecoderPlugin interface.
func (p *DecoderPluginBase) GetLogProcessors() []log.LogProcessor {
	return p.logProcessors
}

// GlobalRegistry is the default global plugin registry.
var GlobalRegistry = NewRegistry()

// Register registers a plugin in the global registry.
func Register(plugin Plugin) error {
	return GlobalRegistry.Register(plugin)
}

// MustRegister registers a plugin in the global registry and panics on error.
func MustRegister(plugin Plugin) {
	GlobalRegistry.MustRegister(plugin)
}
