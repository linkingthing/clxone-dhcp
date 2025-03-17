package resource

type DataSource string

const (
	DataSourceSystem DataSource = "system"
	DataSourceManual DataSource = "manual"
	DataSourceAuto   DataSource = "auto"
)

func (s DataSource) Validate() bool {
	return s == DataSourceSystem || s == DataSourceManual ||
		s == DataSourceAuto
}
