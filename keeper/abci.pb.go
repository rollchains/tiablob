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
	CreateClient  *celestia.CreateClient `protobuf:"bytes,1,opt,name=create_client,json=createClient,proto3" json:"create_client,omitempty"`
	Headers       []*celestia.Header     `protobuf:"bytes,2,rep,name=headers,proto3" json:"headers,omitempty"`
	Proofs        []*celestia.BlobProof  `protobuf:"bytes,3,rep,name=proofs,proto3" json:"proofs,omitempty"`
	PendingBlocks PendingBlocks          `protobuf:"bytes,4,opt,name=pending_blocks,json=pendingBlocks,proto3" json:"pending_blocks"`
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

func (m *InjectedData) GetPendingBlocks() PendingBlocks {
	if m != nil {
		return m.PendingBlocks
	}
	return PendingBlocks{}
}

type PendingBlocks struct {
	BlockHeights []int64 `protobuf:"varint,1,rep,packed,name=block_heights,json=blockHeights,proto3" json:"block_heights,omitempty"`
}

func (m *PendingBlocks) Reset()         { *m = PendingBlocks{} }
func (m *PendingBlocks) String() string { return proto.CompactTextString(m) }
func (*PendingBlocks) ProtoMessage()    {}
func (*PendingBlocks) Descriptor() ([]byte, []int) {
	return fileDescriptor_8c7ccc9d39695dd7, []int{1}
}
func (m *PendingBlocks) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *PendingBlocks) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_PendingBlocks.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *PendingBlocks) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PendingBlocks.Merge(m, src)
}
func (m *PendingBlocks) XXX_Size() int {
	return m.Size()
}
func (m *PendingBlocks) XXX_DiscardUnknown() {
	xxx_messageInfo_PendingBlocks.DiscardUnknown(m)
}

var xxx_messageInfo_PendingBlocks proto.InternalMessageInfo

func (m *PendingBlocks) GetBlockHeights() []int64 {
	if m != nil {
		return m.BlockHeights
	}
	return nil
}

func init() {
	proto.RegisterType((*InjectedData)(nil), "rollchains.tiablob.v1.InjectedData")
	proto.RegisterType((*PendingBlocks)(nil), "rollchains.tiablob.v1.PendingBlocks")
}

func init() { proto.RegisterFile("rollchains/tiablob/v1/abci.proto", fileDescriptor_8c7ccc9d39695dd7) }

var fileDescriptor_8c7ccc9d39695dd7 = []byte{
	// 350 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x91, 0xb1, 0x4f, 0xfa, 0x40,
	0x14, 0xc7, 0x5b, 0x4a, 0xf8, 0x25, 0x07, 0xfd, 0x0d, 0x8d, 0x26, 0x0d, 0x43, 0x6d, 0x90, 0x81,
	0xe9, 0x1a, 0xc0, 0x98, 0xe8, 0xe0, 0x50, 0x1c, 0x70, 0xd0, 0x60, 0x47, 0x07, 0xc9, 0xf5, 0x78,
	0xb6, 0x27, 0x67, 0xaf, 0x69, 0x4f, 0xfe, 0x0e, 0xff, 0x2c, 0x46, 0x46, 0x27, 0x43, 0xe0, 0x1f,
	0x31, 0xbd, 0x82, 0x62, 0xd2, 0x85, 0xed, 0xf5, 0xf5, 0xf3, 0xfd, 0xbc, 0x77, 0x79, 0xc8, 0xcd,
	0x04, 0xe7, 0x34, 0x26, 0x2c, 0xc9, 0x3d, 0xc9, 0x48, 0xc8, 0x45, 0xe8, 0x2d, 0xfa, 0x1e, 0x09,
	0x29, 0xc3, 0x69, 0x26, 0xa4, 0xb0, 0x4e, 0x7f, 0x09, 0xbc, 0x23, 0xf0, 0xa2, 0xdf, 0x3e, 0x89,
	0x44, 0x24, 0x14, 0xe1, 0x15, 0x55, 0x09, 0xb7, 0xaf, 0x2b, 0x74, 0x9c, 0x45, 0xb1, 0xa4, 0x9c,
	0x41, 0x22, 0x73, 0x8f, 0x02, 0x87, 0x5c, 0x32, 0x52, 0x0c, 0xd9, 0xd7, 0x65, 0xb6, 0xb3, 0xae,
	0xa1, 0xd6, 0x5d, 0xf2, 0x0a, 0x54, 0xc2, 0xec, 0x96, 0x48, 0x62, 0x3d, 0x23, 0x93, 0x66, 0x40,
	0x24, 0x4c, 0xcb, 0xb4, 0xad, 0xbb, 0x7a, 0xaf, 0x39, 0xb8, 0xc2, 0x15, 0x1b, 0x1d, 0x0e, 0xc1,
	0x3f, 0xe2, 0x45, 0x1f, 0x8f, 0x94, 0x61, 0xa4, 0xfe, 0x04, 0x2d, 0x7a, 0xf0, 0x65, 0xdd, 0xa3,
	0x7f, 0x31, 0x90, 0x19, 0x64, 0xb9, 0x5d, 0x73, 0x8d, 0x5e, 0x73, 0x30, 0x3c, 0xca, 0x3c, 0x56,
	0xd9, 0x60, 0xef, 0xb0, 0x1e, 0x50, 0x23, 0xcd, 0x84, 0x78, 0xc9, 0x6d, 0x43, 0xd9, 0x2e, 0x8f,
	0xb2, 0xf9, 0x5c, 0x84, 0x93, 0x22, 0x1e, 0xec, 0x2c, 0xd6, 0x23, 0xfa, 0x9f, 0x42, 0x32, 0x63,
	0x49, 0x34, 0x0d, 0xb9, 0xa0, 0xf3, 0xdc, 0xae, 0xab, 0xf7, 0x77, 0x71, 0xe5, 0x45, 0xf0, 0xa4,
	0x84, 0x7d, 0xc5, 0xfa, 0xf5, 0xe5, 0xd7, 0x99, 0x16, 0x98, 0xe9, 0x61, 0xb3, 0x73, 0x81, 0xcc,
	0x3f, 0x94, 0x75, 0x8e, 0x4c, 0xe5, 0x9e, 0xc6, 0x50, 0x2c, 0x96, 0xdb, 0xba, 0x6b, 0xf4, 0x8c,
	0xa0, 0xa5, 0x9a, 0xe3, 0xb2, 0xe7, 0xdf, 0x2c, 0x37, 0x8e, 0xbe, 0xda, 0x38, 0xfa, 0x7a, 0xe3,
	0xe8, 0x1f, 0x5b, 0x47, 0x5b, 0x6d, 0x1d, 0xed, 0x73, 0xeb, 0x68, 0x4f, 0xdd, 0x88, 0xc9, 0xf8,
	0x3d, 0xc4, 0x54, 0xbc, 0x79, 0x15, 0x97, 0x9f, 0x03, 0xa4, 0x90, 0x85, 0x0d, 0x75, 0xdf, 0xe1,
	0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0x00, 0xf9, 0x96, 0xba, 0x6c, 0x02, 0x00, 0x00,
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
	{
		size, err := m.PendingBlocks.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintAbci(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0x22
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

func (m *PendingBlocks) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PendingBlocks) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *PendingBlocks) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.BlockHeights) > 0 {
		dAtA4 := make([]byte, len(m.BlockHeights)*10)
		var j3 int
		for _, num1 := range m.BlockHeights {
			num := uint64(num1)
			for num >= 1<<7 {
				dAtA4[j3] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j3++
			}
			dAtA4[j3] = uint8(num)
			j3++
		}
		i -= j3
		copy(dAtA[i:], dAtA4[:j3])
		i = encodeVarintAbci(dAtA, i, uint64(j3))
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
	l = m.PendingBlocks.Size()
	n += 1 + l + sovAbci(uint64(l))
	return n
}

func (m *PendingBlocks) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.BlockHeights) > 0 {
		l = 0
		for _, e := range m.BlockHeights {
			l += sovAbci(uint64(e))
		}
		n += 1 + sovAbci(uint64(l)) + l
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
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PendingBlocks", wireType)
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
			if err := m.PendingBlocks.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
func (m *PendingBlocks) Unmarshal(dAtA []byte) error {
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
			return fmt.Errorf("proto: PendingBlocks: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PendingBlocks: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType == 0 {
				var v int64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowAbci
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= int64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.BlockHeights = append(m.BlockHeights, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowAbci
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthAbci
				}
				postIndex := iNdEx + packedLen
				if postIndex < 0 {
					return ErrInvalidLengthAbci
				}
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				var elementCount int
				var count int
				for _, integer := range dAtA[iNdEx:postIndex] {
					if integer < 128 {
						count++
					}
				}
				elementCount = count
				if elementCount != 0 && len(m.BlockHeights) == 0 {
					m.BlockHeights = make([]int64, 0, elementCount)
				}
				for iNdEx < postIndex {
					var v int64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowAbci
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= int64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.BlockHeights = append(m.BlockHeights, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field BlockHeights", wireType)
			}
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
