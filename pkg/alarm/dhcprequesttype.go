package alarm

type DHCPRequestType string

const (
	DHCPRequestTypeRequest    DHCPRequestType = "Request"
	DHCPRequestTypeRelease    DHCPRequestType = "Release"
	DHCPRequestTypeDecline    DHCPRequestType = "Decline"
	DHCPRequestTypeConflictIP DHCPRequestType = "ConflictIP"
	DHCPRequestTypeRenew      DHCPRequestType = "Renew"
	DHCPRequestTypeRebind     DHCPRequestType = "Rebind"
)
