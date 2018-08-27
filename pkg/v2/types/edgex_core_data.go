package types

type EdgeX_Core_Data struct {
	// Clients is a map of services used by a DS.
	Clients map[string]clientInfo
	// Service contains service-specific settings.
	Service ServiceInfo
	// Registry contains registry-specific settings.
	Registry RegistryInfo
	// Logging contains logging-specific configuration settings.
	Logging LoggingInfo
	// Metadata contains metadata
	MetaData MetaDataInfo
	// Database
	Database DatabaseInfo
}
