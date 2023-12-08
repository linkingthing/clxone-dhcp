package errorno

import (
	"errors"
	"fmt"
	"strings"

	goresterr "github.com/linkingthing/gorest/error"
)

var (
	ErrSharedNetSubnetIds = func(target string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("shared network %s subnet ids length should excceed 1", target),
			fmt.Sprintf("共享网络 %s 的子网ID数量应该超过1", target))
	}
	ErrNoIntersectionNodes = func(subnet1, subnet2 string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("subnet %s has no intersection nodes with subnet %s", subnet1, subnet2),
			fmt.Sprintf("子网 %s 与 %s 之间无交集节点", subnet1, subnet2))
	}
	ErrExistIntersection = func(target1, target2 interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%v has intersection with %s", target1, target2),
			fmt.Sprintf("%v 与 %v 之间存在交集", target1, target2))
	}
	ErrNoNode = func(target ErrName, name string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s no nodes info", target, name),
			fmt.Sprintf("%s %s 无节点信息", localizeErrName(target), name))
	}
	ErrNotContainNode = func(target ErrName, name string, nodes []string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s should contains nodes %q", target, name, nodes),
			fmt.Sprintf("%s %s 应该包含节点 %q", localizeErrName(target), name, nodes))
	}
	ErrConflict = func(src, dct ErrName, srcValue, dctValue string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s is conflict with %s %s", src, srcValue, dct, dctValue),
			fmt.Sprintf("%s %s 与 %s %s 冲突", localizeErrName(src), srcValue, localizeErrName(dct), dctValue))
	}
	ErrOperateResource = func(op ErrName, name, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s failed: %s", op, name, errMsg),
			fmt.Sprintf("%s %s 失败：%s", localizeErrName(op),
				localizeErrName(ErrName(name)), errMsg))
	}
	ErrNotFound = func(errName ErrName, value string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s not found", errName, value),
			fmt.Sprintf("%s %s 不存在", localizeErrName(errName), value))
	}
	ErrIpv6Preferred = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("ipv6-only preferred must not be less than 300"),
			fmt.Sprintf("提供的IPv6偏好值不能小于300"))
	}
	ErrMinLifetime = func(min uint32) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("min-lifetime must not less than %d, and less than max-lifetime", min),
			fmt.Sprintf("最短租约时长不能小于 %d，且不超过最长租约", min))
	}
	ErrDefaultLifetime = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("default-lifetime should between min-lifttime and max-lifetime"),
			fmt.Sprintf("租约时长应该位于最短租约和最长租约之间"))
	}
	ErrEui64Conflict = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("subnet use eui64 conflict with use address code"),
			fmt.Sprintf("不能同时开启EUI64和地址编码"))
	}
	ErrAddressCodeMask = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("the mask size of subnet use address code must not less than 64"),
			fmt.Sprintf("开启了地址编码的子网的前缀长度不能小于64"))
	}
	ErrHasPool = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("subnet6 has pools, can not enabled eui64 or address code"),
			fmt.Sprintf("子网已配置地址池，不能开启EUI64或者地址编码"))
	}
	ErrHaMode = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("ha model can`t update subnet nodes"),
			fmt.Sprintf("HA模式下不能更新节点"))
	}
	ErrHaModeVip = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("only node with virtual ip could be selected for ha model"),
			fmt.Sprintf("HA模式下只能选择有虚拟IP的节点"))
	}
	ErrHandleCmd = func(cmd, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("handle command %s failed: %s", cmd, errMsg),
			fmt.Sprintf("处理命令 %s 失败: %s", cmd, errMsg))
	}
	ErrContain = func(target ErrName, obj, part interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s %v contains "%v"`, target, obj, part),
			fmt.Sprintf(`%s %v 中包含了"%v"`, localizeErrName(target), obj, part))
	}
	ErrInvalidRange = func(name string, begin, end interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s has invalid scope %v-%v`, name, begin, end),
			fmt.Sprintf(`%s 有无效的范围 %v-%v`, name, begin, end))
	}
	ErrNotInRange = func(target ErrName, begin, end interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s should in scope %v-%v`, target, begin, end),
			fmt.Sprintf(`%s 应该位于范围 %v-%v 中`, localizeErrName(target), begin, end))
	}
	ErrNotInScope = func(target ErrName, values ...string) *goresterr.ErrorMessage {
		localizeValues := make([]string, len(values))
		for i, val := range values {
			localizeValues[i] = localizeErrName(ErrName(val))
		}
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s should in %v`, target, values),
			fmt.Sprintf(`%s 应该位于范围 %v 中`, localizeErrName(target), localizeValues))
	}
	ErrOnlyOne = func(values ...string) *goresterr.ErrorMessage {
		localizeValues := make([]string, len(values))
		for i, val := range values {
			localizeValues[i] = localizeErrName(ErrName(val))
		}
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%q must have only one`, values),
			fmt.Sprintf(`%q 必须有且只能有一个`, localizeValues))
	}
	ErrEmpty = func(values ...string) *goresterr.ErrorMessage {
		localizeValues := make([]string, len(values))
		for i, val := range values {
			localizeValues[i] = localizeErrName(ErrName(val))
		}
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%q is empty", values),
			fmt.Sprintf("%q 不能为空", localizeValues))
	}
	ErrInvalidAddressCode = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`begin address code should in [65, 128], and end in [68 72 76 80 84 88 92 96 100 104 108 112 116 120 124 128]`),
			fmt.Sprintf(`开始地址码应位于范围[65, 128]中，结束地址码为[68 72 76 80 84 88 92 96 100 104 108 112 116 120 124 128]之一`))
	}
	ErrMismatchAddressCode = func(code string, begin, end uint32) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`code %s length mismatch with begin %d and end %d`, code, begin, end),
			fmt.Sprintf(`地址码 %s 的长度不匹配开始 %d 与结束 %d`, code, begin, end))
	}
	ErrUsedReservation = func(ip string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`reservation %s exists with same subnet, mac and hostname or ip`, ip),
			fmt.Sprintf(`%s 已存在拥有相同子网、MAC和主机或IP的 %s`, ip, localizeErrName(ErrNameDhcpReservation)))
	}
	ErrGetNodeInfoFromPrometheus = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("get nodes from prometheus failed"),
			fmt.Sprintf("从Prometheus获取节点信息失败"),
		)
	}
	ErrSubnetWithEui64OrCode = func(name string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`subnet6 %s has opened EUI64 or address code`, name),
			fmt.Sprintf(`%s 已经设置了EUI64 或者地址编码`, name))
	}
	ErrAddressWithEui64OrCode = func(address string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("subnet of %s has opened EUI64 or address code", address),
			fmt.Sprintf("%s所属DHCP子网已设置EUI64或者地址编码", address))
	}
	ErrBiggerThan = func(target ErrName, obj1, obj2 interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s %v is bigger than %v`, target, obj1, obj2),
			fmt.Sprintf(`%s %v 大于了 %v`, localizeErrName(target), obj1, obj2))
	}
	ErrLessThan = func(target ErrName, obj1, obj2 interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf(`%s %v is less than %v`, target, obj1, obj2),
			fmt.Sprintf(`%s %v 小于了 %v`, localizeErrName(target), obj1, obj2))
	}
	ErrChanged = func(target ErrName, obj, before, now interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s of %v changed from %v to %v", target, obj, before, now),
			fmt.Sprintf("%v的%s已从%v变更为%v", obj, localizeErrName(target), before, now))
	}
	ErrNoResourceWith = func(objKind, condKind ErrName, obj, cond interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("cannot find %s %s with %s %s", objKind, obj, condKind, cond),
			fmt.Sprintf("找不到拥有%s%v的%s%v", localizeErrName(objKind), obj, localizeErrName(condKind), cond))
	}

	ErrDuplicate = func(errName ErrName, value string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s is duplicate", errName, value),
			fmt.Sprintf("%s %s 已存在", localizeErrName(errName), value))
	}
	ErrResourceNotFound = func(errName ErrName) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s not found", errName),
			fmt.Sprintf("%s 不存在", localizeErrName(errName)))
	}
	ErrContainResource = func(src, dest ErrName, srcValue, destValue string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s contains %s %s", src, dest, srcValue, destValue),
			fmt.Sprintf("%s %s 包含了 %s %s", localizeErrName(src), localizeErrName(dest), srcValue, destValue))
	}
	ErrBeenUsed = func(errName ErrName, value string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s has been used", errName, value),
			fmt.Sprintf("%s %s 已被使用", localizeErrName(errName), value))
	}
	ErrMissingParams = func(errName ErrName, value string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s params %s is missing", errName, value),
			fmt.Sprintf("%s 参数 %s 缺失", localizeErrName(errName), value))
	}
	ErrInvalidParams = func(errName ErrName, value interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %v is invalid", errName, value),
			fmt.Sprintf("%s %v 不合法", localizeErrName(errName), value))
	}
	ErrOnlySupport = func(errName ErrName, value interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("unknown %s, only support %v", errName, value),
			fmt.Sprintf("不支持 %s, 仅支持 %v", localizeErrName(errName), value))
	}
	ErrEnableResource = func(src, target ErrName) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s didn't enable %s", src, target),
			fmt.Sprintf("%s 未开启 %s", localizeErrName(src), localizeErrName(target)))
	}
	ErrInvalidFormat = func(errName ErrName, opt ErrName) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s action %s is invalid format", errName, opt),
			fmt.Sprintf("%s 操作 %s 请求参数格式不正确", localizeErrName(errName), localizeErrName(opt)))
	}
	ErrUnknownOpt = func(errName ErrName, opt interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s action %v is unknown", errName, opt),
			fmt.Sprintf("%s 操作 %v 无法识别", localizeErrName(errName), opt))
	}
	ErrDBError = func(errName ErrName, target, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s failed:%s", errName, target, errMsg),
			fmt.Sprintf("%s %s 失败: %s", localizeErrName(errName),
				localizeErrName(ErrName(target)), errMsg))
	}
	ErrNetworkError = func(target ErrName, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("get %s failed: %s", target, errMsg),
			fmt.Sprintf("获取 %s 失败: %s", localizeErrName(target), errMsg))
	}
	ErrExportTmp = func(target ErrName, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("export template %s failed:%s", target, errMsg),
			fmt.Sprintf("导出模板 %s 失败: %s", localizeErrName(target), errMsg))
	}
	ErrExport = func(target ErrName, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("export %s failed:%s", target, errMsg),
			fmt.Sprintf("导出数据 %s 失败: %s", localizeErrName(target), errMsg))
	}
	ErrImport = func(target ErrName, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("import %s failed:%s", target, errMsg),
			fmt.Sprintf("导入数据 %s 失败: %s", localizeErrName(target), errMsg))
	}
	ErrInvalidTableHeader = func() *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("failed to parse file: invalid table header"),
			fmt.Sprintf("解析导入文件失败: 表头格式不正确"),
		)
	}
	ErrImportExceedMaxCount = func(target ErrName, maxCount int) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("import %s exceeds max count: %d", target, maxCount),
			fmt.Sprintf("导入数据 %s 超过最大限制: %d", localizeErrName(target), maxCount))
	}
	ErrExceedMaxCount = func(target ErrName, maxCount int) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s exceeds max count: %d", target, maxCount),
			fmt.Sprintf("%s 超过最大限制: %d", localizeErrName(target), maxCount))
	}
	ErrExceedResourceMaxCount = func(target, targetResource ErrName, maxCount int) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s of %s exceeds max count: %d", target, targetResource, maxCount),
			fmt.Sprintf("%s%s超过最大限制: %d", localizeErrName(target), localizeErrName(targetResource), maxCount))
	}
	ErrFileIsEmpty = func(file string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("file %s is empty", file),
			fmt.Sprintf("文件 %s 为空", file))
	}
	ErrReadFile = func(file string, errMsg string) *goresterr.ErrorMessage {
		zhErrMsg := errMsg
		if strings.Contains(errMsg, "only support format of XLSX") {
			zhErrMsg = "仅支持 XLSX 格式的 Excel 文件"
		}
		return goresterr.NewErrorMessage(
			fmt.Sprintf("read file %s failed:%s", file, errMsg),
			fmt.Sprintf("读取文件 %s 失败: %s", file, zhErrMsg))
	}
	ErrParseHeader = func(file string, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("parse file %s header failed:%s", file, errMsg),
			fmt.Sprintf("解析文件 %s 表头失败: %s", file, errMsg))
	}
	ErrMissingMandatory = func(line int, fields []string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("line %d missing mandatory fields: %q", line, fields),
			fmt.Sprintf("第 %d 行缺少必填项: %q", line, fields))
	}
	ErrParseFailed = func(target ErrName, errMsg string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("parse %s failed: %s", target, errMsg),
			fmt.Sprintf("解析 %s 失败: %s", localizeErrName(target), errMsg))
	}
	ErrReadOnly = func(target string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s is read only", target),
			fmt.Sprintf("%s 是只读的", target))
	}
	ErrHasBeenAllocated = func(target ErrName, resource string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s has been allocated", target, resource),
			fmt.Sprintf("%s %s 已分配", localizeErrName(target), resource))
	}
	ErrIPHasBeenAllocated = func(target ErrName, resource string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s has been allocate IP", target, resource),
			fmt.Sprintf("%s %s 已分配IP", localizeErrName(target), resource))
	}
	ErrUsed = func(src, dct ErrName, srcValue, dctValue interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %v is used by %s %v", src, srcValue, dct, dctValue),
			fmt.Sprintf("%s %v 已被%s %v 使用", localizeErrName(src), srcValue, localizeErrName(dct), dctValue))
	}
	ErrUsedBy = func(src, dct ErrName, dctValue string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s is used by %s %s", src, dct, dctValue),
			fmt.Sprintf("%s 已被 %s %s 使用", localizeErrName(src), localizeErrName(dct), dctValue))
	}
	ErrUnsupported = func(opt string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s is unsupported", opt),
			fmt.Sprintf("%s 不支持", opt),
		)
	}
	ErrExpect = func(target ErrName, want, got interface{}) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s: expected is %v, but got %v", target, want, got),
			fmt.Sprintf("%s: 期待的值是 %v, 但输入的是 %v", localizeErrName(target), want, got),
		)
	}
	ErrExceedLimit = func(limit int) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("exceeds max limit %d", limit),
			fmt.Sprintf("超过最大限制 %d", limit),
		)
	}
	ErrParseCIDR = func(prefix string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("parse CIDR prefix failed: %s", prefix),
			fmt.Sprintf("解析CIDR前缀失败：%s", prefix),
		)
	}
	ErrInvalidAddress = func(address string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("invalid address: %s", address),
			fmt.Sprintf("无效的地址: %s", address),
		)
	}
	ErrNotBelongTo = func(source, target ErrName, sourceValue, targetValue string) *goresterr.ErrorMessage {
		return goresterr.NewErrorMessage(
			fmt.Sprintf("%s %s isn't contained by %s %s", source, sourceValue, target, targetValue),
			fmt.Sprintf("%s %s 不属于 %s %s", localizeErrName(source), sourceValue, localizeErrName(target), targetValue),
		)
	}
)

func HandleAPIError(code goresterr.ErrorCode, err error) *goresterr.APIError {
	if errMsg := new(goresterr.ErrorMessage); errors.As(err, &errMsg) {
		return goresterr.NewAPIError(code, *errMsg)
	} else {
		return goresterr.NewAPIError(code, goresterr.ErrorMessage{MessageEN: err.Error(), MessageCN: err.Error()})
	}
}

func TryGetErrorCNMsg(err error) string {
	if err == nil {
		return ""
	}
	em, ok := err.(*goresterr.ErrorMessage)
	if ok {
		return em.ErrorCN()
	}
	return err.Error()
}
