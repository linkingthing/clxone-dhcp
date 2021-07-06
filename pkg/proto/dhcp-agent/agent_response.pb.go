// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.15.1
// source: agent_response.proto

package dhcp_agent

import (
	proto "github.com/golang/protobuf/proto"
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

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type AgentResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Resource      string `protobuf:"bytes,1,opt,name=resource,proto3" json:"resource,omitempty"`
	Method        string `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	Node          string `protobuf:"bytes,3,opt,name=node,proto3" json:"node,omitempty"`
	NodeType      string `protobuf:"bytes,4,opt,name=node_type,json=nodeType,proto3" json:"node_type,omitempty"`
	Succeed       bool   `protobuf:"varint,5,opt,name=succeed,proto3" json:"succeed,omitempty"`
	ErrorMessage  string `protobuf:"bytes,6,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	CmdMessage    string `protobuf:"bytes,7,opt,name=cmd_message,json=cmdMessage,proto3" json:"cmd_message,omitempty"`
	OperationTime string `protobuf:"bytes,8,opt,name=operation_time,json=operationTime,proto3" json:"operation_time,omitempty"`
}

func (x *AgentResponse) Reset() {
	*x = AgentResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_agent_response_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AgentResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AgentResponse) ProtoMessage() {}

func (x *AgentResponse) ProtoReflect() protoreflect.Message {
	mi := &file_agent_response_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AgentResponse.ProtoReflect.Descriptor instead.
func (*AgentResponse) Descriptor() ([]byte, []int) {
	return file_agent_response_proto_rawDescGZIP(), []int{0}
}

func (x *AgentResponse) GetResource() string {
	if x != nil {
		return x.Resource
	}
	return ""
}

func (x *AgentResponse) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *AgentResponse) GetNode() string {
	if x != nil {
		return x.Node
	}
	return ""
}

func (x *AgentResponse) GetNodeType() string {
	if x != nil {
		return x.NodeType
	}
	return ""
}

func (x *AgentResponse) GetSucceed() bool {
	if x != nil {
		return x.Succeed
	}
	return false
}

func (x *AgentResponse) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

func (x *AgentResponse) GetCmdMessage() string {
	if x != nil {
		return x.CmdMessage
	}
	return ""
}

func (x *AgentResponse) GetOperationTime() string {
	if x != nil {
		return x.OperationTime
	}
	return ""
}

var File_agent_response_proto protoreflect.FileDescriptor

var file_agent_response_proto_rawDesc = []byte{
	0x0a, 0x14, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xfb, 0x01,
	0x0a, 0x0d, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x1a, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6d,
	0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x04, 0x6e, 0x6f, 0x64, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x6e, 0x6f, 0x64, 0x65, 0x5f,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6e, 0x6f, 0x64, 0x65,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x65, 0x64, 0x12, 0x23,
	0x0a, 0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6d, 0x64, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6d, 0x64, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6f, 0x70,
	0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_agent_response_proto_rawDescOnce sync.Once
	file_agent_response_proto_rawDescData = file_agent_response_proto_rawDesc
)

func file_agent_response_proto_rawDescGZIP() []byte {
	file_agent_response_proto_rawDescOnce.Do(func() {
		file_agent_response_proto_rawDescData = protoimpl.X.CompressGZIP(file_agent_response_proto_rawDescData)
	})
	return file_agent_response_proto_rawDescData
}

var file_agent_response_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_agent_response_proto_goTypes = []interface{}{
	(*AgentResponse)(nil), // 0: proto.AgentResponse
}
var file_agent_response_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_agent_response_proto_init() }
func file_agent_response_proto_init() {
	if File_agent_response_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_agent_response_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AgentResponse); i {
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
			RawDescriptor: file_agent_response_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_agent_response_proto_goTypes,
		DependencyIndexes: file_agent_response_proto_depIdxs,
		MessageInfos:      file_agent_response_proto_msgTypes,
	}.Build()
	File_agent_response_proto = out.File
	file_agent_response_proto_rawDesc = nil
	file_agent_response_proto_goTypes = nil
	file_agent_response_proto_depIdxs = nil
}
