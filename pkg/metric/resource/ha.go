package resource

type HaRequest struct {
	MasterIP string      `json:"masterIP"`
	Role     ServiceRole `json:"role"`
	Vip      string      `json:"vip"`
	SlaveIP  string      `json:"slaveIP"`
}

type HaResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ServiceRole string

const (
	ServiceRoleDHCP       ServiceRole = "dhcp"
	ServiceRoleDNS        ServiceRole = "dns"
	ServiceRoleController ServiceRole = "controller"
	ServiceRoleDataCenter ServiceRole = "dataCenter"
)

type HaCmd string

const (
	HaRequestFail   int   = 0
	HaRequestOK     int   = 1
	HaCmdStartHa    HaCmd = "start_ha"
	HaCmdMasterUp   HaCmd = "master_up"
	HaCmdMasterDown HaCmd = "master_down"
)

func (r *HaResponse) Error(msg string) *HaResponse {
	r.Code = HaRequestFail
	r.Message = msg
	return r
}

func (r *HaResponse) Success(msg string) *HaResponse {
	r.Code = HaRequestOK
	r.Message = msg
	return r
}
