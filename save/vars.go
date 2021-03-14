package save

import (
	"io"
	"os"
	"strings"
	"unicode/utf16"
)

type gameTime struct {
	Month      uint8
	DayOfMonth uint8
	Hour       uint8
	Minute     uint8
}

// Only present in PC saves. Corresponds to some Windows type.
type systemTime struct {
	Year        uint16
	Month       uint16
	DayOfWeek   uint16
	Day         uint16
	Hour        uint16
	Minute      uint16
	Second      uint16
	Millisecond uint16
}

type varBlock struct {
	blockIdentifier [5]uint8

	// Save metadata.
	Metadata struct {
		VersionNumber     uint32
		LastMissionPassed string
		MissionPackGame   bool
		Gap               boolPadding
	}

	// Basic position information.
	Position struct {
		CurrentIsland  uint32
		CameraPosition vector3
	}

	// Information about the in-game clock.
	Clock struct {
		MillisecondsPerGameMinute uint32
		LastClockTick             uint32
		GameClock                 gameTime
		Weekday                   uint8
		StoredGameClock           gameTime
		ClockHasBeenStored        bool
	}

	// Basic gameplay setting information.
	Player struct {
		PadMode          uint16
		HasPlayerCheated bool

		// Pad to 4 bytes after the bool.
		Gap boolPadding
	}

	// Information for mapping between real time and game time.
	TimeMapping struct {
		TimeInMilliseconds uint32
		TimeScale          float32
		TimeStep           float32
		TimeStepNonClipped float32
		FrameCounter       uint32
	}

	Weather struct {
		OldWeatherType    uint16
		NewWeatherType    uint16
		ForcedWeatherType uint16

		// Pad to 8 bytes; we're at 6 currently.
		Gap [2]padding

		InterpolationValue float32
		WeatherTypeInList  uint32
		RainHeaviness      float32
	}

	Camera struct {
		Vehicle   uint32
		Character uint32
	}

	Surroundings struct {
		CurrentArea uint32
		InvertLook  bool
		Gap         boolPadding

		ExtraColor struct {
			Color              uint32
			Enabled            bool
			Gap                boolPadding
			InterpolationValue float32
			WeatherType        uint32
		}

		WaterConfiguration uint32
	}

	Riots struct {
		Active             bool
		PoliceCarsDisabled bool
		Gap                [2]padding
	}

	WantedLevel struct {
		Maximum      uint32
		MaximumChaos uint32
	}

	Audience struct {
		FrenchGame bool
		GermanGame bool
		Uncensored bool
		Gap        padding
	}

	UnknownBuffer [11]uint32

	CinematicCamera struct {
		// What?
		ShouldBeHere       uint8
		RemainingHelpShows uint8
	}

	TimeGroup struct {
		// On desktop:
		DesktopSystemTime systemTime
		DesktopUnknown    [2]uint8

		// On mobile:
		MobileUnknown [30]uint8

		// On PS2:
		PlaystationUnknown [34]uint8
	}

	Gui struct {
		TargetMarkerHandle uint32
		CarStealHelpShown  bool
	}

	Cheats struct {
		TaxisHaveNitro    bool
		ProstitutesPayYou bool
		Gap               padding
	}

	MobileUnknown [4]uint8
}

func WriteVarBlock(file io.Writer, block *varBlock) {
	mustWrite(file, block.blockIdentifier)
	mustWrite(file, block.Metadata.VersionNumber)

	if /* on mobile */ true {
		missionRunes := []rune(block.Metadata.LastMissionPassed)
		encoded := utf16.Encode(missionRunes)

		mustWrite(file, encoded)

		// Pad to 100 characters long.
		mustWrite(file, make([]uint16, 100-len(encoded)))
	} else {
		println("aaaa")
	}

	mustWrite(file, block.Metadata.MissionPackGame)
	mustWrite(file, block.Metadata.Gap)

	mustWrite(file, block.Position)
	mustWrite(file, block.Clock)
	mustWrite(file, block.Player)
	mustWrite(file, block.TimeMapping)
	mustWrite(file, block.Weather)
	mustWrite(file, block.Camera)
	mustWrite(file, block.Surroundings)
	mustWrite(file, block.Riots)
	mustWrite(file, block.WantedLevel)
	mustWrite(file, block.Audience)
	mustWrite(file, block.UnknownBuffer)
	mustWrite(file, block.CinematicCamera)

	if /* on desktop */ false {
		mustWrite(file, block.TimeGroup.DesktopSystemTime)
		mustWrite(file, block.TimeGroup.DesktopUnknown)
	} else if /* on mobile */ true {
		mustWrite(file, block.TimeGroup.MobileUnknown)
	} else if /* on PS2 */ false {
		mustWrite(file, block.TimeGroup.PlaystationUnknown)
	} else {
		// What?
	}

	mustWrite(file, block.Gui)
	mustWrite(file, block.Cheats)
	mustWrite(file, block.MobileUnknown)
}

func ReadVarBlock(file *os.File) varBlock {
	block := varBlock{}

	mustRead(file, &block.blockIdentifier)
	mustRead(file, &block.Metadata.VersionNumber)

	// For mobile, the characters are 2 bytes each.
	if /* on mobile */ true {
		characters := make([]uint16, 100)
		mustRead(file, &characters)

		missionRunes := utf16.Decode(characters)
		missionString := string(missionRunes)

		// Split after the null terminator.
		index := strings.IndexRune(missionString, '\x00')
		block.Metadata.LastMissionPassed = missionString[:index]
	}

	mustRead(file, &block.Metadata.MissionPackGame)
	mustRead(file, &block.Metadata.Gap)

	mustRead(file, &block.Position)
	mustRead(file, &block.Clock)
	mustRead(file, &block.Player)
	mustRead(file, &block.TimeMapping)
	mustRead(file, &block.Weather)
	mustRead(file, &block.Camera)
	mustRead(file, &block.Surroundings)
	mustRead(file, &block.Riots)
	mustRead(file, &block.WantedLevel)
	mustRead(file, &block.Audience)
	mustRead(file, &block.UnknownBuffer)
	mustRead(file, &block.CinematicCamera)

	if /* on desktop */ false {
		mustRead(file, &block.TimeGroup.DesktopSystemTime)
		mustRead(file, &block.TimeGroup.DesktopUnknown)
	} else if /* on mobile */ true {
		mustRead(file, &block.TimeGroup.MobileUnknown)
	} else if /* on PS2 */ false {
		mustRead(file, &block.TimeGroup.PlaystationUnknown)
	} else {
		// What?
	}

	mustRead(file, &block.Gui)
	mustRead(file, &block.Cheats)
	mustRead(file, &block.MobileUnknown)

	println("Read var block")

	return block
}
