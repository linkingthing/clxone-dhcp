package resource

const ResourceIDCacheHitRatio = "cachehitratio"

type CacheHitRatio struct {
	Ratios     []RatioWithTimestamp `json:"ratios"`
	ViewRatios []ViewCacheHitRatio  `json:"viewRatios"`
}

type ViewCacheHitRatio struct {
	View   string               `json:"view"`
	Ratios []RatioWithTimestamp `json:"ratios"`
}
