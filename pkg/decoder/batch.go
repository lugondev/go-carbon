package decoder

import (
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/view"
)

type BatchDecoder struct {
	registry *Registry
}

func NewBatchDecoder(registry *Registry) *BatchDecoder {
	return &BatchDecoder{
		registry: registry,
	}
}

func (b *BatchDecoder) DecodeAllFast(dataList [][]byte, programID *solana.PublicKey) ([]*Event, error) {
	if len(dataList) == 0 {
		return nil, nil
	}

	events := make([]*Event, 0, len(dataList))

	b.registry.mu.RLock()
	decoders := b.getDecodersForProgram(programID)
	b.registry.mu.RUnlock()

	if len(decoders) == 0 {
		return events, nil
	}

	for _, data := range dataList {
		if len(data) < 8 {
			continue
		}

		eventView, err := view.NewEventView(data)
		if err != nil {
			continue
		}

		for _, decoder := range decoders {
			anchorDecoder, ok := decoder.(*AnchorDecoderBase)
			if !ok {
				if decoder.CanDecode(data) {
					event, err := decoder.Decode(data)
					if err == nil && event != nil {
						events = append(events, event)
						break
					}
				}
				continue
			}

			if anchorDecoder.FastCanDecodeWithView(eventView) {
				event, err := anchorDecoder.DecodeFromView(eventView)
				if err == nil && event != nil {
					events = append(events, event)
					break
				}
			}
		}
	}

	return events, nil
}

func (b *BatchDecoder) DecodeAllParallel(dataList [][]byte, programID *solana.PublicKey, workers int) ([]*Event, error) {
	if len(dataList) == 0 {
		return nil, nil
	}

	if workers <= 0 {
		workers = 4
	}

	type result struct {
		index int
		event *Event
	}

	resultsChan := make(chan result, len(dataList))
	var wg sync.WaitGroup

	chunkSize := (len(dataList) + workers - 1) / workers

	for i := 0; i < workers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(dataList) {
			end = len(dataList)
		}
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
				if len(data) < 8 {
					continue
				}

				eventView, err := view.NewEventView(data)
				if err != nil {
					continue
				}

				for _, decoder := range decoders {
					anchorDecoder, ok := decoder.(*AnchorDecoderBase)
					if !ok {
						if decoder.CanDecode(data) {
							event, err := decoder.Decode(data)
							if err == nil && event != nil {
								resultsChan <- result{index: startIdx + i, event: event}
								break
							}
						}
						continue
					}

					if anchorDecoder.FastCanDecodeWithView(eventView) {
						event, err := anchorDecoder.DecodeFromView(eventView)
						if err == nil && event != nil {
							resultsChan <- result{index: startIdx + i, event: event}
							break
						}
					}
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
		resultsMap[res.index] = res.event
	}

	events := make([]*Event, 0, len(resultsMap))
	for i := 0; i < len(dataList); i++ {
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
