// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.15.8
// source: thresholds.proto

package alarm

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RegisterThreshold struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BaseThreshold *BaseThreshold `protobuf:"bytes,1,opt,name=base_threshold,json=baseThreshold,proto3" json:"base_threshold,omitempty"`
	Value         uint64         `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
	SendMail      bool           `protobuf:"varint,3,opt,name=send_mail,json=sendMail,proto3" json:"send_mail,omitempty"`
}

func (x *RegisterThreshold) Reset() {
	*x = RegisterThreshold{}
	if protoimpl.UnsafeEnabled {
		mi := &file_thresholds_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RegisterThreshold) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterThreshold) ProtoMessage() {}

func (x *RegisterThreshold) ProtoReflect() protoreflect.Message {
	mi := &file_thresholds_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterThreshold.ProtoReflect.Descriptor instead.
func (*RegisterThreshold) Descriptor() ([]byte, []int) {
	return file_thresholds_proto_rawDescGZIP(), []int{0}
}

func (x *RegisterThreshold) GetBaseThreshold() *BaseThreshold {
	if x != nil {
		return x.BaseThreshold
	}
	return nil
}

func (x *RegisterThreshold) GetValue() uint64 {
	if x != nil {
		return x.Value
	}
	return 0
}

func (x *RegisterThreshold) GetSendMail() bool {
	if x != nil {
		return x.SendMail
	}
	return false
}

type UpdateThreshold struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     ThresholdName `protobuf:"varint,1,opt,name=name,proto3,enum=proto.ThresholdName" json:"name,omitempty"`
	Value    uint64        `protobuf:"varint,2,opt,name=value,proto3" json:"value,omitempty"`
	SendMail bool          `protobuf:"varint,3,opt,name=send_mail,json=sendMail,proto3" json:"send_mail,omitempty"`
}

func (x *UpdateThreshold) Reset() {
	*x = UpdateThreshold{}
	if protoimpl.UnsafeEnabled {
		mi := &file_thresholds_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateThreshold) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateThreshold) ProtoMessage() {}

func (x *UpdateThreshold) ProtoReflect() protoreflect.Message {
	mi := &file_thresholds_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateThreshold.ProtoReflect.Descriptor instead.
func (*UpdateThreshold) Descriptor() ([]byte, []int) {
	return file_thresholds_proto_rawDescGZIP(), []int{1}
}

func (x *UpdateThreshold) GetName() ThresholdName {
	if x != nil {
		return x.Name
	}
	return ThresholdName_cpuUsedRatio
}

func (x *UpdateThreshold) GetValue() uint64 {
	if x != nil {
		return x.Value
	}
	return 0
}

func (x *UpdateThreshold) GetSendMail() bool {
	if x != nil {
		return x.SendMail
	}
	return false
}

type DeRegisterThreshold struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name ThresholdName `protobuf:"varint,1,opt,name=name,proto3,enum=proto.ThresholdName" json:"name,omitempty"`
}

func (x *DeRegisterThreshold) Reset() {
	*x = DeRegisterThreshold{}
	if protoimpl.UnsafeEnabled {
		mi := &file_thresholds_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeRegisterThreshold) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeRegisterThreshold) ProtoMessage() {}

func (x *DeRegisterThreshold) ProtoReflect() protoreflect.Message {
	mi := &file_thresholds_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeRegisterThreshold.ProtoReflect.Descriptor instead.
func (*DeRegisterThreshold) Descriptor() ([]byte, []int) {
	return file_thresholds_proto_rawDescGZIP(), []int{2}
}

func (x *DeRegisterThreshold) GetName() ThresholdName {
	if x != nil {
		return x.Name
	}
	return ThresholdName_cpuUsedRatio
}

var File_thresholds_proto protoreflect.FileDescriptor

var file_thresholds_proto_rawDesc = []byte{
	0x0a, 0x10, 0x74, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0b, 0x61, 0x6c, 0x61, 0x72, 0x6d,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x83, 0x01, 0x0a, 0x11, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x65, 0x72, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12, 0x3b, 0x0a, 0x0e,
	0x62, 0x61, 0x73, 0x65, 0x5f, 0x74, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x61, 0x73,
	0x65, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x52, 0x0d, 0x62, 0x61, 0x73, 0x65,
	0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12,
	0x1b, 0x0a, 0x09, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x08, 0x73, 0x65, 0x6e, 0x64, 0x4d, 0x61, 0x69, 0x6c, 0x22, 0x6e, 0x0a, 0x0f,
	0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x12,
	0x28, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x14, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x4e,
	0x61, 0x6d, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12,
	0x1b, 0x0a, 0x09, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x08, 0x73, 0x65, 0x6e, 0x64, 0x4d, 0x61, 0x69, 0x6c, 0x22, 0x3f, 0x0a, 0x13,
	0x44, 0x65, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x65, 0x72, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68,
	0x6f, 0x6c, 0x64, 0x12, 0x28, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68,
	0x6f, 0x6c, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x42, 0x08, 0x5a,
	0x06, 0x61, 0x6c, 0x61, 0x72, 0x6d, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_thresholds_proto_rawDescOnce sync.Once
	file_thresholds_proto_rawDescData = file_thresholds_proto_rawDesc
)

func file_thresholds_proto_rawDescGZIP() []byte {
	file_thresholds_proto_rawDescOnce.Do(func() {
		file_thresholds_proto_rawDescData = protoimpl.X.CompressGZIP(file_thresholds_proto_rawDescData)
	})
	return file_thresholds_proto_rawDescData
}

var file_thresholds_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_thresholds_proto_goTypes = []interface{}{
	(*RegisterThreshold)(nil),   // 0: proto.RegisterThreshold
	(*UpdateThreshold)(nil),     // 1: proto.UpdateThreshold
	(*DeRegisterThreshold)(nil), // 2: proto.DeRegisterThreshold
	(*BaseThreshold)(nil),       // 3: proto.BaseThreshold
	(ThresholdName)(0),          // 4: proto.ThresholdName
}
var file_thresholds_proto_depIdxs = []int32{
	3, // 0: proto.RegisterThreshold.base_threshold:type_name -> proto.BaseThreshold
	4, // 1: proto.UpdateThreshold.name:type_name -> proto.ThresholdName
	4, // 2: proto.DeRegisterThreshold.name:type_name -> proto.ThresholdName
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_thresholds_proto_init() }
func file_thresholds_proto_init() {
	if File_thresholds_proto != nil {
		return
	}
	file_alarm_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_thresholds_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RegisterThreshold); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_thresholds_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpdateThreshold); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_thresholds_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeRegisterThreshold); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_thresholds_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_thresholds_proto_goTypes,
		DependencyIndexes: file_thresholds_proto_depIdxs,
		MessageInfos:      file_thresholds_proto_msgTypes,
	}.Build()
	File_thresholds_proto = out.File
	file_thresholds_proto_rawDesc = nil
	file_thresholds_proto_goTypes = nil
	file_thresholds_proto_depIdxs = nil
}
