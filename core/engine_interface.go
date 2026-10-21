package core

import "fmt"

type EngineType string

const (
	EngineUSN        EngineType = "usn"
	EngineRawMFT     EngineType = "mft"
	EngineEverything EngineType = "everything"
	EngineCrawler    EngineType = "crawler"
)

// IndexerEngine provides a unified interface for all disk scanning strategies.
type IndexerEngine interface {
	Type() EngineType
	Name() string
	Scan(target string) (*VolumeIndex, uint64, error)
}

// USNEngine implements IndexerEngine using Windows USN Change Journal.
type USNEngine struct{}

func (e *USNEngine) Type() EngineType { return EngineUSN }
func (e *USNEngine) Name() string     { return "Windows USN Journal (FSCTL_ENUM_USN_DATA)" }

func (e *USNEngine) Scan(target string) (*VolumeIndex, uint64, error) {
	idx, err := ScanVolumeUSN(target)
	if err != nil {
		return nil, 0, err
	}
	rootID, _ := idx.FindFolderID(target)
	return idx, rootID, nil
}

// CrawlerEngine implements non-elevated directory crawling as an automatic fallback.
type CrawlerEngine struct{}

func (e *CrawlerEngine) Type() EngineType { return EngineCrawler }
func (e *CrawlerEngine) Name() string     { return "Direct Filesystem Crawler (Non-elevated)" }

func (e *CrawlerEngine) Scan(target string) (*VolumeIndex, uint64, error) {
	return ScanFolderFallback(target)
}

// EngineManager manages active scanning engines with graceful fallback.
type EngineManager struct {
	PreferredEngine EngineType
	Engines         map[EngineType]IndexerEngine
}

func NewEngineManager(preferred EngineType) *EngineManager {
	em := &EngineManager{
		PreferredEngine: preferred,
		Engines:         make(map[EngineType]IndexerEngine),
	}
	em.Engines[EngineUSN] = &USNEngine{}
	em.Engines[EngineCrawler] = &CrawlerEngine{}
	return em
}

// Scan attempts to scan using the preferred engine, falling back to crawler if permissions fail.
func (em *EngineManager) Scan(target string) (*VolumeIndex, uint64, string, error) {
	eng, exists := em.Engines[em.PreferredEngine]
	if exists {
		idx, rootID, err := eng.Scan(target)
		if err == nil {
			return idx, rootID, eng.Name(), nil
		}
	}

	// Fallback to non-elevated crawler
	crawler := em.Engines[EngineCrawler]
	idx, rootID, err := crawler.Scan(target)
	if err != nil {
		return nil, 0, "", fmt.Errorf("all indexing engines failed for %s: %w", target, err)
	}
	return idx, rootID, crawler.Name(), nil
}
