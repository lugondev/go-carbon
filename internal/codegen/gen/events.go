package gen

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/lugondev/go-carbon/internal/codegen"
)

// EventsGenerator generates event type definitions and decoders.
type EventsGenerator struct {
	*Generator
}

// NewEventsGenerator creates a new events generator.
func NewEventsGenerator(gen *Generator) *EventsGenerator {
	return &EventsGenerator{Generator: gen}
}

// Generate generates all event types from the IDL.
func (g *EventsGenerator) Generate() error {
	if len(g.IDL.Events) == 0 {
		return nil
	}

	// Generate each event definition
	for _, event := range g.IDL.Events {
		if err := g.generateEvent(event); err != nil {
			return fmt.Errorf("failed to generate event %s: %w", event.Name, err)
		}
	}

	// Generate event parser
	if err := g.generateEventParser(); err != nil {
		return fmt.Errorf("failed to generate event parser: %w", err)
	}

	return nil
}

// generateEvent generates a single event type.
func (g *EventsGenerator) generateEvent(event codegen.IDLEvent) error {
	eventName := FormatTypeName(event.Name)

	// Add documentation
	if len(event.Docs) > 0 {
		for _, doc := range FormatDocs(event.Docs) {
			g.File.Comment(doc)
		}
	}

	// Generate event struct
	if err := g.generateEventStruct(eventName, event); err != nil {
		return err
	}

	// Generate discriminator constant
	g.generateEventDiscriminator(eventName, event.Discriminator)

	// Generate Decode method
	if err := g.generateEventDecode(eventName, event); err != nil {
		return err
	}

	g.File.Line()
	return nil
}

// generateEventStruct generates the event struct type.
func (g *EventsGenerator) generateEventStruct(eventName string, event codegen.IDLEvent) error {
	if len(event.Fields) == 0 {
		// Empty event
		g.File.Type().Id(eventName + "Event").Struct()
		g.File.Line()
		return nil
	}

	// Generate struct fields
	fields := make([]jen.Code, 0, len(event.Fields))
	for _, field := range event.Fields {
		fieldStmt := GenerateStructField(field, g.Generator)
		fields = append(fields, fieldStmt)
	}

	g.File.Type().Id(eventName + "Event").Struct(fields...)
	g.File.Line()

	return nil
}

// generateEventDiscriminator generates the event discriminator constant.
func (g *EventsGenerator) generateEventDiscriminator(eventName string, disc []byte) {
	constName := eventName + "EventDiscriminator"
	g.File.Var().Id(constName).Op("=").Add(DiscriminatorToBytes(disc))
	g.File.Line()
}

// generateEventDecode generates the Decode method for events.
func (g *EventsGenerator) generateEventDecode(eventName string, event codegen.IDLEvent) error {
	eventTypeName := eventName + "Event"
	discConst := eventName + "EventDiscriminator"

	g.File.Comment(fmt.Sprintf("Decode%sEvent decodes event data into %s.", eventName, eventTypeName))
	g.File.Func().Id("Decode"+eventName+"Event").Params(
		jen.Id("data").Index().Byte(),
	).Params(
		jen.Op("*").Id(eventTypeName),
		jen.Error(),
	).Block(
		// Check minimum length (8 bytes discriminator)
		jen.If(jen.Len(jen.Id("data")).Op("<").Lit(8)).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("event data too short"))),
		),
		jen.Line(),

		// Verify discriminator
		jen.Id("discriminator").Op(":=").Id("data").Index(jen.Lit(0).Op(":").Lit(8)),
		jen.If(
			jen.Op("!").Qual("bytes", "Equal").Call(
				jen.Id("discriminator"),
				jen.Id(discConst),
			),
		).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit(fmt.Sprintf("invalid discriminator for %s event", eventName)),
			)),
		),
		jen.Line(),

		// Decode event data
		jen.Var().Id("event").Id(eventTypeName),
		jen.If(
			jen.Err().Op(":=").Qual("github.com/gagliardetto/binary", "UnmarshalBorsh").Call(
				jen.Op("&").Id("event"),
				jen.Id("data").Index(jen.Lit(8).Op(":")),
			),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("failed to decode event: %w"),
				jen.Err(),
			)),
		),
		jen.Line(),

		jen.Return(jen.Op("&").Id("event"), jen.Nil()),
	)
	g.File.Line()

	return nil
}

// generateEventParser generates a unified event parser.
func (g *EventsGenerator) generateEventParser() error {
	g.File.Comment("ParseEvent parses an event from raw data.")
	g.File.Comment("It uses the discriminator (first 8 bytes) to identify the event type.")
	g.File.Func().Id("ParseEvent").Params(
		jen.Id("data").Index().Byte(),
	).Params(
		jen.Interface(),
		jen.Error(),
	).Block(
		// Check minimum length
		jen.If(jen.Len(jen.Id("data")).Op("<").Lit(8)).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("event data too short"))),
		),
		jen.Line(),

		// Extract discriminator
		jen.Id("discriminator").Op(":=").Id("data").Index(jen.Lit(0).Op(":").Lit(8)),
		jen.Line(),

		// Switch on discriminator
		jen.Switch(jen.Qual("encoding/hex", "EncodeToString").Call(jen.Id("discriminator"))).Block(
			g.generateEventParserCases()...,
		),
	)
	g.File.Line()

	return nil
}

// generateEventParserCases generates switch cases for event parser.
func (g *EventsGenerator) generateEventParserCases() []jen.Code {
	cases := []jen.Code{}

	for _, event := range g.IDL.Events {
		eventName := FormatTypeName(event.Name)
		discHex := DiscriminatorToHex(event.Discriminator)

		cases = append(cases,
			jen.Case(jen.Lit(discHex)).Block(
				jen.Return(jen.Id("Decode"+eventName+"Event").Call(jen.Id("data"))),
			),
		)
	}

	// Default case
	cases = append(cases,
		jen.Default().Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("unknown event discriminator: %s"),
				jen.Qual("encoding/hex", "EncodeToString").Call(jen.Id("discriminator")),
			)),
		),
	)

	return cases
}

// GenerateEventsFile generates the events.go file.
func GenerateEventsFile(idl *codegen.IDL, packageName, outputDir string) error {
	gen := NewGenerator(idl, packageName)
	eventsGen := NewEventsGenerator(gen)

	// Add file header comment
	gen.File.Comment("Code generated by go-carbon. DO NOT EDIT.")
	gen.File.Line()

	// Generate all events
	if err := eventsGen.Generate(); err != nil {
		return err
	}

	// Save to file
	filename := fmt.Sprintf("%s/events.go", outputDir)
	return gen.WriteToFile(filename)
}
