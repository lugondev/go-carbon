package decoder

import (
	"context"
	"fmt"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/view"
)

type BatchOptions struct {
	CollectErrors bool
	MaxErrors     int
}

type BatchResult struct {
	Events []*Event
	Errors []error
}

type DecodeError struct {
	Index   int
	Message string
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode error at index %d: %s", e.Index, e.Message)
}

func newDecodeError(index int, message string) *DecodeError {
	return &DecodeError{Index: index, Message: message}
}

type BatchDecoder struct {
	registry *Registry
}

func NewBatchDecoder(registry *Registry) *BatchDecoder {
	return &BatchDecoder{
		registry: registry,
	}
}

func (b *BatchDecoder) DecodeAllFast(dataList [][]byte, programID *solana.PublicKey) ([]*Event, error) {
	result := b.DecodeAllFastWithOptions(dataList, programID, nil)
	return result.Events, nil
}

func (b *BatchDecoder) DecodeAllFastWithOptions(dataList [][]byte, programID *solana.PublicKey, opts *BatchOptions) *BatchResult {
	result := &BatchResult{
		Events: make([]*Event, 0, len(dataList)),
		Errors: make([]error, 0),
	}

	if len(dataList) == 0 {
		return result
	}

	collectErrors := opts != nil && opts.CollectErrors
	maxErrors := 0
	if opts != nil && opts.MaxErrors > 0 {
		maxErrors = opts.MaxErrors
	}

	b.registry.mu.RLock()
	decoders := b.getDecodersForProgram(programID)
	b.registry.mu.RUnlock()

	if len(decoders) == 0 {
		return result
	}

	for i, data := range dataList {
		if len(data) < 8 {
			if collectErrors {
				result.Errors = append(result.Errors, newDecodeError(i, "data too short: need at least 8 bytes"))
				if maxErrors > 0 && len(result.Errors) >= maxErrors {
					break
				}
			}
			continue
		}

		eventView, err := view.NewEventView(data)
		if err != nil {
			if collectErrors {
				result.Errors = append(result.Errors, newDecodeError(i, fmt.Sprintf("failed to create event view: %v", err)))
				if maxErrors > 0 && len(result.Errors) >= maxErrors {
					break
				}
			}
			continue
		}

		decoded := false
		for _, decoder := range decoders {
			anchorDecoder, ok := decoder.(*AnchorDecoderBase)
			if !ok {
				if decoder.CanDecode(data) {
					event, err := decoder.Decode(data)
					if err == nil && event != nil {
						result.Events = append(result.Events, event)
						decoded = true
						break
					}
					if collectErrors && err != nil {
						result.Errors = append(result.Errors, newDecodeError(i, fmt.Sprintf("decode failed: %v", err)))
						if maxErrors > 0 && len(result.Errors) >= maxErrors {
							return result
						}
					}
				}
				continue
			}

			if anchorDecoder.FastCanDecodeWithView(eventView) {
				event, err := anchorDecoder.DecodeFromView(eventView)
				if err == nil && event != nil {
					result.Events = append(result.Events, event)
					decoded = true
					break
				}
				if collectErrors && err != nil {
					result.Errors = append(result.Errors, newDecodeError(i, fmt.Sprintf("anchor decode failed: %v", err)))
					if maxErrors > 0 && len(result.Errors) >= maxErrors {
						return result
					}
				}
			}
		}

		if !decoded && collectErrors {
			result.Errors = append(result.Errors, newDecodeError(i, "no decoder matched"))
			if maxErrors > 0 && len(result.Errors) >= maxErrors {
				break
			}
		}
	}

	return result
}

func (b *BatchDecoder) decodeSingleEvent(data []byte, decoders []Decoder) (*Event, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short: need at least 8 bytes")
	}

	eventView, err := view.NewEventView(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create event view: %w", err)
	}

	for _, decoder := range decoders {
		anchorDecoder, ok := decoder.(*AnchorDecoderBase)
		if !ok {
			if decoder.CanDecode(data) {
				event, err := decoder.Decode(data)
				if err == nil && event != nil {
					return event, nil
				}
			}
			continue
		}

		if anchorDecoder.FastCanDecodeWithView(eventView) {
			event, err := anchorDecoder.DecodeFromView(eventView)
			if err == nil && event != nil {
				return event, nil
			}
		}
	}

	return nil, fmt.Errorf("no decoder matched")
}

func (b *BatchDecoder) DecodeAllParallel(dataList [][]byte, programID *solana.PublicKey, workers int) ([]*Event, error) {
	return b.DecodeAllParallelWithContext(context.Background(), dataList, programID, workers)
}

func (b *BatchDecoder) DecodeAllParallelWithContext(ctx context.Context, dataList [][]byte, programID *solana.PublicKey, workers int) ([]*Event, error) {
	if len(dataList) == 0 {
		return nil, nil
	}

	if workers <= 0 {
		workers = 4
	}

	type result struct {
		index int
		event *Event
		err   error
	}

	resultsChan := make(chan result, len(dataList))
	var wg sync.WaitGroup

	chunkSize := (len(dataList) + workers - 1) / workers

	for i := 0; i < workers; i++ {
		start := i * chunkSize
		end := min(start+chunkSize, len(dataList))
		if start >= len(dataList) {
			break
		}

		wg.Add(1)
		go func(chunk [][]byte, startIdx int) {
			defer wg.Done()

			b.registry.mu.RLock()
			decoders := b.getDecodersForProgram(programID)
			b.registry.mu.RUnlock()

			for i, data := range chunk {
				select {
				case <-ctx.Done():
					return
				default:
				}

				event, err := b.decodeSingleEvent(data, decoders)
				if err == nil && event != nil {
					resultsChan <- result{index: startIdx + i, event: event}
				}
			}
		}(dataList[start:end], start)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	resultsMap := make(map[int]*Event)
	for res := range resultsChan {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			resultsMap[res.index] = res.event
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	events := make([]*Event, 0, len(resultsMap))
	for i := range len(dataList) {
		if event, exists := resultsMap[i]; exists {
			events = append(events, event)
		}
	}

	return events, nil
}

func (b *BatchDecoder) getDecodersForProgram(programID *solana.PublicKey) []Decoder {
	var decoders []Decoder

	if programID != nil && !programID.IsZero() {
		if programDecoders, exists := b.registry.decodersByPubkey[*programID]; exists {
			decoders = make([]Decoder, len(programDecoders))
			copy(decoders, programDecoders)
			return decoders
		}
	}

	decoders = make([]Decoder, 0, len(b.registry.decoders))
	for _, decoder := range b.registry.decoders {
		decoders = append(decoders, decoder)
	}

	return decoders
}

type DiscriminatorMatcher struct {
	discriminators map[[8]byte]*AnchorDecoderBase
}

func NewDiscriminatorMatcher(decoders []*AnchorDecoderBase) *DiscriminatorMatcher {
	matcher := &DiscriminatorMatcher{
		discriminators: make(map[[8]byte]*AnchorDecoderBase, len(decoders)),
	}

	for _, decoder := range decoders {
		matcher.discriminators[decoder.discriminator] = decoder
	}

	return matcher
}

func (m *DiscriminatorMatcher) Match(discriminator [8]byte) (*AnchorDecoderBase, bool) {
	decoder, exists := m.discriminators[discriminator]
	return decoder, exists
}

func (m *DiscriminatorMatcher) MatchBatch(discriminators [][8]byte) []*AnchorDecoderBase {
	results := make([]*AnchorDecoderBase, len(discriminators))

	for i, disc := range discriminators {
		if decoder, exists := m.discriminators[disc]; exists {
			results[i] = decoder
		}
	}

	return results
}
