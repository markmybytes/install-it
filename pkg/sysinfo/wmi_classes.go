package sysinfo

/*
The Win32_Processor WMI class represents a device that can interpret a sequence of instructions on a computer running on a Windows operating system.

See: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-processor
*/
type Win32_Processor struct {
	AddressWidth                            uint16
	Architecture                            uint16
	AssetTag                                string
	Availability                            uint16
	Caption                                 string
	Characteristics                         uint32
	ConfigManagerErrorCode                  uint32
	ConfigManagerUserConfig                 bool
	CpuStatus                               uint16
	CreationClassName                       string
	CurrentClockSpeed                       uint32
	CurrentVoltage                          uint16
	DataWidth                               uint16
	Description                             string
	DeviceID                                string
	ErrorCleared                            bool
	ErrorDescription                        string
	ExtClock                                uint32
	Family                                  uint16
	InstallDate                             string
	L2CacheSize                             uint32
	L2CacheSpeed                            uint32
	L3CacheSize                             uint32
	L3CacheSpeed                            uint32
	LastErrorCode                           uint32
	Level                                   uint16
	LoadPercentage                          uint16
	Manufacturer                            string
	MaxClockSpeed                           uint32
	Name                                    string
	NumberOfCores                           uint32
	NumberOfEnabledCore                     uint32
	NumberOfLogicalProcessors               uint32
	OtherFamilyDescription                  string
	PartNumber                              string
	PNPDeviceID                             string
	PowerManagementCapabilities             []uint16
	PowerManagementSupported                bool
	ProcessorId                             string
	ProcessorType                           uint16
	Revision                                uint16
	Role                                    string
	SecondLevelAddressTranslationExtensions bool
	SerialNumber                            string
	SocketDesignation                       string
	Status                                  string
	StatusInfo                              uint16
	Stepping                                string
	SystemCreationClassName                 string
	SystemName                              string
	ThreadCount                             uint32
	UniqueId                                string
	UpgradeMethod                           uint16
	Version                                 string
	VirtualizationFirmwareEnabled           bool
	VMMonitorModeExtensions                 bool
	VoltageCaps                             uint32
}

/*
The Win32_BaseBoard WMI class represents a baseboard, which is also known as a motherboard or system board.

See: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-baseboard
*/
type Win32_BaseBoard struct {
	Caption                 string
	ConfigOptions           []string
	CreationClassName       string
	Depth                   float32
	Description             string
	Height                  float32
	HostingBoard            bool
	HotSwappable            bool
	InstallDate             string
	Manufacturer            string
	Model                   string
	Name                    string
	OtherIdentifyingInfo    string
	PartNumber              string
	PoweredOn               bool
	Product                 string
	Removable               bool
	Replaceable             bool
	RequirementsDescription string
	RequiresDaughterBoard   bool
	SerialNumber            string
	SKU                     string
	SlotLayout              string
	SpecialRequirements     bool
	Status                  string
	Tag                     string
	Version                 string
	Weight                  float32
	Width                   float32
}

/*
The Win32_PhysicalMemory WMI class represents a physical memory device located on a computer system and available to the operating system.

See: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-physicalmemory
*/
type Win32_PhysicalMemory struct {
	Attributes           uint32
	BankLabel            string
	Capacity             uint64
	Caption              string
	ConfiguredClockSpeed uint32
	ConfiguredVoltage    uint32
	CreationClassName    string
	DataWidth            uint16
	Description          string
	DeviceLocator        string
	FormFactor           uint16
	HotSwappable         bool
	InstallDate          string
	InterleaveDataDepth  uint16
	InterleavePosition   uint32
	Manufacturer         string
	MaxVoltage           uint32
	MemoryType           uint16
	MinVoltage           uint32
	Model                string
	Name                 string
	OtherIdentifyingInfo string
	PartNumber           string
	PositionInRow        uint32
	PoweredOn            bool
	Removable            bool
	Replaceable          bool
	SerialNumber         string
	SKU                  string
	SMBIOSMemoryType     uint32
	Speed                uint32
	Status               string
	Tag                  string
	TotalWidth           uint16
	TypeDetail           uint16
	Version              string
}

/*
The Win32_DiskDrive WMI class represents a physical disk drive as seen by a computer running the Windows operating system.

See: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-diskdrive
*/
type Win32_DiskDrive struct {
	Availability                uint16
	BytesPerSector              uint32
	Capabilities                []int32 // []uint16 causes panic (call of reflect.Value on int32 Value)
	CapabilityDescriptions      []string
	Caption                     string
	CompressionMethod           string
	ConfigManagerErrorCode      uint32
	ConfigManagerUserConfig     bool
	CreationClassName           string
	DefaultBlockSize            uint64
	Description                 string
	DeviceID                    string
	ErrorCleared                bool
	ErrorDescription            string
	ErrorMethodology            string
	FirmwareRevision            string
	Index                       uint32
	InstallDate                 string
	InterfaceType               string
	LastErrorCode               uint32
	Manufacturer                string
	MaxBlockSize                uint64
	MaxMediaSize                uint64
	MediaLoaded                 bool
	MediaType                   string
	MinBlockSize                uint64
	Model                       string
	Name                        string
	NeedsCleaning               bool
	NumberOfMediaSupported      uint32
	Partitions                  uint32
	PNPDeviceID                 string
	PowerManagementCapabilities []uint16
	PowerManagementSupported    bool
	SCSIBus                     uint32
	SCSILogicalUnit             uint16
	SCSIPort                    uint16
	SCSITargetId                uint16
	SectorsPerTrack             uint32
	SerialNumber                string
	Signature                   uint32
	Size                        uint64
	Status                      string
	StatusInfo                  uint16
	SystemCreationClassName     string
	SystemName                  string
	TotalCylinders              uint64
	TotalHeads                  uint32
	TotalSectors                uint64
	TotalTracks                 uint64
	TracksPerCylinder           uint32
}

// Device setup class GUIDs for filtering Win32_PnPEntity by device type.
const (
	GUID_DEVCLASS_DISPLAY   = "{4d36e968-e325-11ce-bfc1-08002be10318}"
	GUID_DEVCLASS_NET       = "{4d36e972-e325-11ce-bfc1-08002be10318}"
	GUID_DEVCLASS_BLUETOOTH = "{e0cbf06c-cd8b-4647-bb8a-263b43f0f974}"
)

/*
The Win32_PnPEntity WMI class represents a Plug and Play device on a Windows system.
HardwareID and CompatibleID are populated by the bus enumerator and are available
even when no driver is installed for the device.

See: https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-pnpentity
*/
type Win32_PnPEntity struct {
	DeviceID               string
	PNPDeviceID            string
	ClassGuid              string
	HardwareID             []string
	CompatibleID           []string
	ConfigManagerErrorCode uint32
	Manufacturer           string
	Name                   string
	Description            string
	Service                string
	Status                 string
	Present                bool
}
