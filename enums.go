package seadex

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type Tag string

const (
	TagDolbyVision       Tag = "Dolby Vision"
	TagHDR               Tag = "HDR"
	TagDebandRequired    Tag = "Deband Required"
	TagDebandRecommended Tag = "Deband Recommended"
	TagYUV444P           Tag = "YUV444P"
	TagPatchRequired     Tag = "Patch Required"
	TagMisplacedSpecial  Tag = "Misplaced Special"
	TagVFR               Tag = "VFR"
	TagIncomplete        Tag = "Incomplete"
	TagBroken            Tag = "Broken"
)

func ParseTag(s string) (Tag, error) {
	tags := []Tag{
		TagDolbyVision, TagHDR, TagDebandRequired, TagDebandRecommended,
		TagYUV444P, TagPatchRequired, TagMisplacedSpecial, TagVFR,
		TagIncomplete, TagBroken,
	}
	lower := strings.ToLower(s)
	for _, t := range tags {
		if strings.ToLower(string(t)) == lower {
			return t, nil
		}
	}
	return "", fmt.Errorf("'%s' is not a valid Tag", s)
}

type Tracker string

const (
	TrackerNyaa            Tracker = "Nyaa"
	TrackerAnimeTosho      Tracker = "AnimeTosho"
	TrackerAniDex          Tracker = "AniDex"
	TrackerRuTracker       Tracker = "RuTracker"
	TrackerOther           Tracker = "Other"
	TrackerAnimeBytes      Tracker = "AB"
	TrackerBeyondHD        Tracker = "BeyondHD"
	TrackerPassThePopcorn  Tracker = "PassThePopcorn"
	TrackerBroadcastTheNet Tracker = "BroadcastTheNet"
	TrackerHDBits          Tracker = "HDBits"
	TrackerBlutopia        Tracker = "Blutopia"
	TrackerAither          Tracker = "Aither"
	TrackerOtherPrivate    Tracker = "OtherPrivate"
)

var trackerURLs = map[Tracker]string{
	TrackerNyaa:            "aHR0cHM6Ly9ueWFhLnNp",
	TrackerAnimeTosho:      "aHR0cHM6Ly9hbmltZXRvc2hvLm9yZw==",
	TrackerAniDex:          "aHR0cHM6Ly9hbmlkZXguaW5mbw==",
	TrackerRuTracker:       "aHR0cHM6Ly9ydXRyYWNrZXIub3Jn",
	TrackerAnimeBytes:      "aHR0cHM6Ly9hbmltZWJ5dGVzLnR2",
	TrackerBeyondHD:        "aHR0cHM6Ly9iZXlvbmQtaGQubWU=",
	TrackerPassThePopcorn:  "aHR0cHM6Ly9wYXNzdGhlcG9wY29ybi5tZQ==",
	TrackerBroadcastTheNet: "aHR0cHM6Ly9icm9hZGNhc3RoZS5uZXQ=",
	TrackerHDBits:          "aHR0cHM6Ly9oZGJpdHMub3Jn",
	TrackerBlutopia:        "aHR0cHM6Ly9ibHV0b3BpYS5jYw==",
	TrackerAither:          "aHR0cHM6Ly9haXRoZXIuY2M=",
	TrackerOther:           "",
	TrackerOtherPrivate:    "",
}

var publicTrackers = map[Tracker]bool{
	TrackerNyaa:       true,
	TrackerAnimeTosho: true,
	TrackerAniDex:     true,
	TrackerRuTracker:  true,
	TrackerOther:      true,
}

func ParseTracker(s string) (Tracker, error) {
	trackers := []Tracker{
		TrackerNyaa, TrackerAnimeTosho, TrackerAniDex, TrackerRuTracker, TrackerOther,
		TrackerAnimeBytes, TrackerBeyondHD, TrackerPassThePopcorn, TrackerBroadcastTheNet,
		TrackerHDBits, TrackerBlutopia, TrackerAither, TrackerOtherPrivate,
	}
	lower := strings.ToLower(s)
	for _, t := range trackers {
		if strings.ToLower(string(t)) == lower {
			return t, nil
		}
	}
	return "", fmt.Errorf("'%s' is not a valid Tracker", s)
}

func (t Tracker) IsPublic() bool {
	return publicTrackers[t]
}

func (t Tracker) IsPrivate() bool {
	return !t.IsPublic()
}

func (t Tracker) URL() string {
	encoded, ok := trackerURLs[t]
	if !ok || encoded == "" {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	return string(b)
}
