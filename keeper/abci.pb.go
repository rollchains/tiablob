// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: rollchains/tiablob/v1/abci.proto

package keeper

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	celestia "github.com/rollchains/tiablob/lightclients/celestia"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type InjectedData struct {
	CreateClient *celestia.CreateClient `protobuf:"bytes,1,opt,name=create_client,json=createClient,proto3" json:"create_client,omitempty"`
	Headers      []*celestia.Header     `protobuf:"bytes,2,rep,name=headers,proto3" json:"headers,omitempty"`
	Proofs       []*celestia.BlobProof  `protobuf:"bytes,3,rep,name=proofs,proto3" json:"proofs,omitempty"`
}

func (m *InjectedData) Reset()         { *m = InjectedData{} }
func (m *InjectedData) String() string { return proto.CompactTextString(m) }
func (*InjectedData) ProtoMessage()    {}
func (*InjectedData) Descriptor() ([]byte, []int) {
	return fileDescriptor_8c7ccc9d39695dd7, []int{0}
}
func (m *InjectedData) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *InjectedData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_InjectedData.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *InjectedData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_InjectedData.Merge(m, src)
}
func (m *InjectedData) XXX_Size() int {
	return m.Size()
}
func (m *InjectedData) XXX_DiscardUnknown() {
	xxx_messageInfo_InjectedData.DiscardUnknown(m)
}

var xxx_messageInfo_InjectedData proto.InternalMessageInfo

func (m *InjectedData) GetCreateClient() *celestia.CreateClient {
	if m != nil {
		return m.CreateClient
	}
	return nil
}

func (m *InjectedData) GetHeaders() []*celestia.Header {
	if m != nil {
		return m.Headers
	}
	return nil
}

func (m *InjectedData) GetProofs() []*celestia.BlobProof {
	if m != nil {
		return m.Proofs
	}
	return nil
}

func init() {
	proto.RegisterType((*InjectedData)(nil), "rollchains.tiablob.v1.InjectedData")
}

func init() { proto.RegisterFile("rollchains/tiablob/v1/abci.proto", fileDescriptor_8c7ccc9d39695dd7) }

var fileDescriptor_8c7ccc9d39695dd7 = []byte{
	// 283 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x28, 0xca, 0xcf, 0xc9,
	0x49, 0xce, 0x48, 0xcc, 0xcc, 0x2b, 0xd6, 0x2f, 0xc9, 0x4c, 0x4c, 0xca, 0xc9, 0x4f, 0xd2, 0x2f,
	0x33, 0xd4, 0x4f, 0x4c, 0x4a, 0xce, 0xd4, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x45, 0xa8,
	0xd0, 0x83, 0xaa, 0xd0, 0x2b, 0x33, 0x94, 0x12, 0x49, 0xcf, 0x4f, 0xcf, 0x07, 0xab, 0xd0, 0x07,
	0xb1, 0x20, 0x8a, 0xa5, 0xac, 0xb0, 0x18, 0x97, 0x93, 0x99, 0x9e, 0x51, 0x92, 0x9c, 0x93, 0x99,
	0x9a, 0x57, 0x52, 0xac, 0x9f, 0x9c, 0x9a, 0x93, 0x5a, 0x5c, 0x92, 0x99, 0x08, 0xb2, 0x04, 0xc6,
	0x86, 0xe8, 0x55, 0xea, 0x65, 0xe2, 0xe2, 0xf1, 0xcc, 0xcb, 0x4a, 0x4d, 0x2e, 0x49, 0x4d, 0x71,
	0x49, 0x2c, 0x49, 0x14, 0x8a, 0xe3, 0xe2, 0x4d, 0x2e, 0x4a, 0x4d, 0x2c, 0x49, 0x8d, 0x87, 0xe8,
	0x96, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x36, 0xb2, 0xd4, 0xc3, 0xe2, 0x22, 0x64, 0x4b, 0xf4, 0xe0,
	0x06, 0x97, 0x19, 0xea, 0x39, 0x83, 0x4d, 0x70, 0x06, 0xcb, 0x04, 0xf1, 0x24, 0x23, 0xf1, 0x84,
	0x7c, 0xb9, 0xd8, 0x33, 0x52, 0x13, 0x53, 0x52, 0x8b, 0x8a, 0x25, 0x98, 0x14, 0x98, 0x35, 0xb8,
	0x8d, 0x8c, 0x49, 0x32, 0xd9, 0x03, 0xac, 0x37, 0x08, 0x66, 0x86, 0x90, 0x1f, 0x17, 0x5b, 0x41,
	0x51, 0x7e, 0x7e, 0x5a, 0xb1, 0x04, 0x33, 0xd8, 0x34, 0x33, 0x92, 0x4c, 0x73, 0xca, 0xc9, 0x4f,
	0x0a, 0x00, 0x69, 0x0f, 0x82, 0x9a, 0xe2, 0x64, 0x77, 0xe2, 0x91, 0x1c, 0xe3, 0x85, 0x47, 0x72,
	0x8c, 0x0f, 0x1e, 0xc9, 0x31, 0x4e, 0x78, 0x2c, 0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7,
	0x72, 0x0c, 0x51, 0x2a, 0xe9, 0x99, 0x25, 0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0x58,
	0x02, 0x3c, 0x3b, 0x35, 0xb5, 0x20, 0xb5, 0x28, 0x89, 0x0d, 0x1c, 0xac, 0xc6, 0x80, 0x00, 0x00,
	0x00, 0xff, 0xff, 0xb7, 0xf1, 0xac, 0xbc, 0xe3, 0x01, 0x00, 0x00,
}

func (m *InjectedData) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *InjectedData) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *InjectedData) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Proofs) > 0 {
		for iNdEx := len(m.Proofs) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Proofs[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintAbci(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.Headers) > 0 {
		for iNdEx := len(m.Headers) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Headers[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintAbci(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if m.CreateClient != nil {
		{
			size, err := m.CreateClient.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintAbci(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintAbci(dAtA []byte, offset int, v uint64) int {
	offset -= sovAbci(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *InjectedData) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.CreateClient != nil {
		l = m.CreateClient.Size()
		n += 1 + l + sovAbci(uint64(l))
	}
	if len(m.Headers) > 0 {
		for _, e := range m.Headers {
			l = e.Size()
			n += 1 + l + sovAbci(uint64(l))
		}
	}
	if len(m.Proofs) > 0 {
		for _, e := range m.Proofs {
			l = e.Size()
			n += 1 + l + sovAbci(uint64(l))
		}
	}
	return n
}

func sovAbci(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozAbci(x uint64) (n int) {
	return sovAbci(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *InjectedData) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAbci
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: InjectedData: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: InjectedData: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreateClient", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAbci
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAbci
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAbci
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreateClient == nil {
				m.CreateClient = &celestia.CreateClient{}
			}
			if err := m.CreateClient.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Headers", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAbci
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAbci
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAbci
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Headers = append(m.Headers, &celestia.Header{})
			if err := m.Headers[len(m.Headers)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Proofs", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAbci
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAbci
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthAbci
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Proofs = append(m.Proofs, &celestia.BlobProof{})
			if err := m.Proofs[len(m.Proofs)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAbci(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthAbci
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipAbci(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAbci
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAbci
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAbci
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthAbci
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupAbci
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthAbci
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthAbci        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAbci          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupAbci = fmt.Errorf("proto: unexpected end of group")
)
