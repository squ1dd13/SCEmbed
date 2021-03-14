package save

import (
	"io"
	"os"
)

type scriptAttachType int8

const (
	attachScriptToPed     scriptAttachType = 0
	attachScriptToObject  scriptAttachType = 1
	attachBrainForCodeUse scriptAttachType = 3
	attachBrokenCodeUse   scriptAttachType = 4
	attachAttractorScript scriptAttachType = 5
	attachNotInUse        scriptAttachType = -1
)

type brain struct {
	General struct {
		Index      uint16
		AttachType scriptAttachType
		GroupId    uint8
		Status     uint32
		Radius     float32
	}

	ScriptName string

	pedOrObject struct {
		ModelId          uint16
		ActivationChance uint16
		Gap              [4]padding
	}
}

func readBrain(file *os.File) brain {
	theBrain := brain{}

	mustRead(file, &theBrain.General)

	attachType := theBrain.General.AttachType

	if attachType == attachBrainForCodeUse || attachType == attachAttractorScript {
		nameBytes := make([]uint8, 8)
		mustRead(file, &nameBytes)

		theBrain.ScriptName = string(nameBytes)

		return theBrain
	}

	mustRead(file, &theBrain.pedOrObject)
	return theBrain
}

type script struct {
	Index uint16

	// Mobile only.
	StreamedScriptIndex uint32

	Mission struct {
		// 69000 bytes. We don't use [69000]uint8 though, because this may not be
		//  a mission script.
		MissionCode []uint8

		// [1024]uint32
		Locals []uint32
	}

	Link struct {
		PointerToNext     uint32
		PointerToPrevious uint32
	}

	Name string

	Execution struct {
		BaseInstructionPointer    uint32
		CurrentInstructionPointer uint32
		ReturnStack               [8]uint32
		ReturnStackIndex          uint16

		Gap [2]padding
	}

	// Local storage size depends on game version.
	Locals []uint32
	Timers [2]uint32

	Info struct {
		IsActive           bool
		ConditionResult    bool
		UsesMissionCleanup bool
		IsExternal         bool
		OverridesTextbox   bool
		AttachType         scriptAttachType

		Unknown [2]uint8

		ActivationTime      uint32
		ConditionCount      uint16
		InvertReturn        bool
		GameOverCheckActive bool
		WantedOrBusted      bool

		Unknown0 [3]uint8

		SkipScenePosition uint32
		IsMission         bool

		Gap boolPadding

		RelativeInstructionPointer uint32
		RelativeReturnStack        [8]uint32
	}
}

func readScript(platform *GamePlatform, file *os.File) script {
	theScript := script{}
	mustRead(file, &theScript.Index)

	if platform.IsMobile {
		mustRead(file, &theScript.StreamedScriptIndex)
	}

	if theScript.Index&0x8000 != 0 {
		theScript.Mission.MissionCode = make([]uint8, 69000)
		mustRead(file, &theScript.Mission.MissionCode)

		theScript.Mission.Locals = make([]uint32, 1024)
		mustRead(file, &theScript.Mission.Locals)
	}

	mustRead(file, &theScript.Link)

	nameBytes := make([]byte, 8)
	file.Read(nameBytes)

	theScript.Name = string(nameBytes)
	nullTerminate(&theScript.Name)

	mustRead(file, &theScript.Execution)

	theScript.Locals = make([]uint32, platform.MaxLocals())

	mustRead(file, &theScript.Locals)
	mustRead(file, &theScript.Timers)
	mustRead(file, &theScript.Info)

	return theScript
}

type scriptBlock struct {
	blockIdentifier [5]uint8

	GlobalStorage struct {
		GlobalSpaceSize uint32
		Globals         []uint32
	}

	Brains [70]brain

	MissionInfo struct {
		OnMissionFlagOffset uint32
		LastMissionTime     uint32
	}

	Arrays struct {
		StaticReplacements [25]struct {
			Type       uint32
			Handle     uint32
			NewModelId int32
			OldModelId int32
		}

		InvisibleObjects [20]struct {
			Type   uint32
			Handle uint32
		}

		SuppressedVehicleModels [20]uint32

		LodAssignments [10]struct {
			ObjectHandle uint32
			LodHandle    uint32
		}

		ScriptAssignments [8]struct {
			ActorModelId uint32
			ScriptName   [8]uint8
			Unknown      [2]uint32
		}
	}

	Values struct {
		Unknown [2]uint8

		MainScmSize        uint32
		LargestMissionSize uint32
		MissionCount       uint32
		HighestLocal       uint32
		RunningScriptCount uint32
	}

	// Mobile
	SaveGameStateType uint32

	Running struct {
		RunningScripts []script
	}

	// There is more to the block, but we don't need any of it.
}

func (block *scriptBlock) ScriptAt(index int) *script {
	return &block.Running.RunningScripts[index]
}

// Extends the global storage to a size big enough to store `variableCount` variables.
func (block *scriptBlock) ExpandGlobalSpace(variableCount int) {
	// Update the global storage size.
	block.GlobalStorage.GlobalSpaceSize = uint32(variableCount) * 4

	// Extend the global variable slice.
	extendCount := variableCount - len(block.GlobalStorage.Globals)
	block.GlobalStorage.Globals = append(block.GlobalStorage.Globals, make([]uint32, extendCount)...)

	// Add the size into the first two globals.
	// [0] stores the lowest order byte in its highest order byte, and the other three bytes are in the lowest three of [1].
	// There's probably a shorter way of writing these lines, but I CBA to think about it.
	block.GlobalStorage.Globals[0] = (block.GlobalStorage.Globals[0] & 0x00ffffff) | (block.GlobalStorage.GlobalSpaceSize << 24)
	block.GlobalStorage.Globals[1] = (block.GlobalStorage.Globals[1] & 0xff000000) | (block.GlobalStorage.GlobalSpaceSize >> 8)
}

func WriteScriptBlock(platform *GamePlatform, file io.Writer, block *scriptBlock) {
	mustWrite(file, block.blockIdentifier)
	mustWrite(file, block.GlobalStorage.GlobalSpaceSize)
	mustWrite(file, block.GlobalStorage.Globals)

	for _, theBrain := range block.Brains {
		mustWrite(file, theBrain.General)

		if theBrain.General.AttachType == attachBrainForCodeUse || theBrain.General.AttachType == attachAttractorScript {
			mustWrite(file, ([]byte)(theBrain.ScriptName))
			mustWrite(file, make([]byte, 8-len(theBrain.ScriptName)))
		} else {
			mustWrite(file, theBrain.pedOrObject)
		}
	}

	mustWrite(file, block.MissionInfo)
	mustWrite(file, block.Arrays)
	mustWrite(file, block.Values)

	if platform.IsMobile {
		mustWrite(file, block.SaveGameStateType)
	}

	for _, theScript := range block.Running.RunningScripts {
		mustWrite(file, theScript.Index)

		if platform.IsMobile {
			mustWrite(file, theScript.StreamedScriptIndex)
		}

		if theScript.Index&0x8000 != 0 {
			mustWrite(file, theScript.Mission.MissionCode)
			mustWrite(file, theScript.Mission.Locals)
		}

		mustWrite(file, theScript.Link)
		mustWrite(file, ([]byte)(theScript.Name))
		mustWrite(file, make([]byte, 8-len(theScript.Name)))
		mustWrite(file, theScript.Execution)
		mustWrite(file, theScript.Locals)
		mustWrite(file, theScript.Timers)
		mustWrite(file, theScript.Info)
	}
}

func ReadScriptBlock(platform *GamePlatform, file *os.File) scriptBlock {
	block := scriptBlock{}

	mustRead(file, &block.blockIdentifier)

	mustRead(file, &block.GlobalStorage.GlobalSpaceSize)

	// Size is in bytes, so divide by 4 to find the number of uint32s.
	block.GlobalStorage.Globals = make([]uint32, block.GlobalStorage.GlobalSpaceSize/4)
	mustRead(file, &block.GlobalStorage.Globals)

	for i := range block.Brains {
		block.Brains[i] = readBrain(file)
	}

	mustRead(file, &block.MissionInfo)
	mustRead(file, &block.Arrays.StaticReplacements)
	mustRead(file, &block.Arrays.InvisibleObjects)
	mustRead(file, &block.Arrays.SuppressedVehicleModels)
	mustRead(file, &block.Arrays.LodAssignments)
	mustRead(file, &block.Arrays.ScriptAssignments)
	mustRead(file, &block.Values)

	if platform.IsMobile {
		mustRead(file, &block.SaveGameStateType)
	}

	block.Running.RunningScripts = make([]script, block.Values.RunningScriptCount)
	for i := range block.Running.RunningScripts {
		block.Running.RunningScripts[i] = readScript(platform, file)
	}

	return block
}
