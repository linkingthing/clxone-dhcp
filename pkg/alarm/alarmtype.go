package alarm

type AlarmType string

const (
	AlarmTypeNone AlarmType = "None"

	AlarmTypeIllegalPacket                        AlarmType = "IllegalPacket"
	AlarmTypeIllegalPacketWithOpCode              AlarmType = "IllegalPacketWithOpCode"
	AlarmTypeIllegalInformPacketWithoutSourceAddr AlarmType = "IllegalInformPacketWithoutSourceAddr"

	AlarmTypeIllegalOptions                         AlarmType = "IllegalOptions"
	AlarmTypeIllegalOptionWithUnexpectedMessageType AlarmType = "IllegalOptionWithUnexpectedMessageType"
	AlarmTypeIllegalOptionWithUltraShortLeaseTime   AlarmType = "IllegalOptionWithUltraShortLeaseTime"
	AlarmTypeIllegalOptionWithUltraLongLeaseTime    AlarmType = "IllegalOptionWithUltraLongLeaseTime"
	AlarmTypeIllegalOptionWithUnexpectedServerId    AlarmType = "IllegalOptionWithUnexpectedServerId"
	AlarmTypeIllegalOptionWithForbiddenServerId     AlarmType = "IllegalOptionWithForbiddenServerId"
	AlarmTypeIllegalOptionWithMandatoryServerId     AlarmType = "IllegalOptionWithMandatoryServerId"
	AlarmTypeIllegalOptionWithInvalidServerId       AlarmType = "IllegalOptionWithInvalidServerId"
	AlarmTypeIllegalOptionWithMandatoryClientId     AlarmType = "IllegalOptionWithMandatoryClientId"
	AlarmTypeIllegalOptionWithInvalidClientId       AlarmType = "IllegalOptionWithInvalidClientId"

	AlarmTypeIllegalClient            AlarmType = "IllegalClient"
	AlarmTypeIllegalClientWithHighQPS AlarmType = "IllegalClientWithHighQps"
)
