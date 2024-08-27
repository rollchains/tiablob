// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: rollchains/tiablob/v1/genesis.proto

package tiablob

import (
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	celestia "github.com/rollchains/celestia-da-light-client"
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

// GenesisState defines the rollchain module's genesis state.
type GenesisState struct {
	Validators []Validator `protobuf:"bytes,1,rep,name=validators,proto3" json:"validators"`
	// the height of the last block that was proven to be posted to Celestia.
	// increment only, never skipping heights.
	ProvenHeight         int64                  `protobuf:"varint,2,opt,name=proven_height,json=provenHeight,proto3" json:"proven_height,omitempty"`
	CelestiaGenesisState *celestia.GenesisState `protobuf:"bytes,3,opt,name=celestia_genesis_state,json=celestiaGenesisState,proto3" json:"celestia_genesis_state,omitempty"`
	PendingBlocks        []*BlockWithExpiration `protobuf:"bytes,4,rep,name=pending_blocks,json=pendingBlocks,proto3" json:"pending_blocks,omitempty"`
}

func (m *GenesisState) Reset()         { *m = GenesisState{} }
func (m *GenesisState) String() string { return proto.CompactTextString(m) }
func (*GenesisState) ProtoMessage()    {}
func (*GenesisState) Descriptor() ([]byte, []int) {
	return fileDescriptor_f2bd6ea5db040ca7, []int{0}
}
func (m *GenesisState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GenesisState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GenesisState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GenesisState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GenesisState.Merge(m, src)
}
func (m *GenesisState) XXX_Size() int {
	return m.Size()
}
func (m *GenesisState) XXX_DiscardUnknown() {
	xxx_messageInfo_GenesisState.DiscardUnknown(m)
}

var xxx_messageInfo_GenesisState proto.InternalMessageInfo

func (m *GenesisState) GetValidators() []Validator {
	if m != nil {
		return m.Validators
	}
	return nil
}

func (m *GenesisState) GetProvenHeight() int64 {
	if m != nil {
		return m.ProvenHeight
	}
	return 0
}

func (m *GenesisState) GetCelestiaGenesisState() *celestia.GenesisState {
	if m != nil {
		return m.CelestiaGenesisState
	}
	return nil
}

func (m *GenesisState) GetPendingBlocks() []*BlockWithExpiration {
	if m != nil {
		return m.PendingBlocks
	}
	return nil
}

func init() {
	proto.RegisterType((*GenesisState)(nil), "rollchains.tiablob.v1.GenesisState")
}

func init() {
	proto.RegisterFile("rollchains/tiablob/v1/genesis.proto", fileDescriptor_f2bd6ea5db040ca7)
}

var fileDescriptor_f2bd6ea5db040ca7 = []byte{
	// 343 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x91, 0xcd, 0x4a, 0xc3, 0x40,
	0x10, 0x80, 0x93, 0xb6, 0x78, 0xd8, 0xb6, 0x1e, 0x42, 0x95, 0x50, 0x30, 0x46, 0x8b, 0x50, 0x3c,
	0x6c, 0x68, 0x3d, 0x88, 0x1e, 0x0b, 0xfe, 0x5c, 0x8d, 0xa0, 0xe0, 0x25, 0x6c, 0xd2, 0x25, 0x59,
	0x5c, 0x77, 0x62, 0x76, 0x1b, 0xf4, 0x2d, 0x7c, 0x23, 0xaf, 0x3d, 0xf6, 0xe8, 0x49, 0xa4, 0x7d,
	0x11, 0xc9, 0x5f, 0xad, 0x18, 0x6f, 0xc3, 0xcc, 0x37, 0xb3, 0xf3, 0xed, 0xa0, 0x41, 0x02, 0x9c,
	0x07, 0x11, 0x61, 0x42, 0x3a, 0x8a, 0x11, 0x9f, 0x83, 0xef, 0xa4, 0x23, 0x27, 0xa4, 0x82, 0x4a,
	0x26, 0x71, 0x9c, 0x80, 0x02, 0x63, 0xe7, 0x07, 0xc2, 0x25, 0x84, 0xd3, 0x51, 0xbf, 0x17, 0x42,
	0x08, 0x39, 0xe1, 0x64, 0x51, 0x01, 0xf7, 0x0f, 0xea, 0x27, 0x3e, 0xcf, 0x68, 0xf2, 0x5a, 0x22,
	0x47, 0xf5, 0x48, 0x4a, 0x38, 0x9b, 0x12, 0x05, 0x49, 0x89, 0x9d, 0xd7, 0x60, 0x9c, 0x85, 0x91,
	0x0a, 0x38, 0xa3, 0x42, 0x49, 0x27, 0xa0, 0x9c, 0x4a, 0xc5, 0x48, 0xd6, 0x5c, 0xc5, 0x45, 0xef,
	0xe1, 0x7b, 0x03, 0x75, 0xae, 0x0a, 0x89, 0x5b, 0x45, 0x14, 0x35, 0x2e, 0x11, 0x5a, 0xcf, 0x97,
	0xa6, 0x6e, 0x37, 0x87, 0xed, 0xb1, 0x8d, 0x6b, 0xc5, 0xf0, 0x5d, 0x05, 0x4e, 0x5a, 0xf3, 0xcf,
	0x7d, 0xcd, 0xdd, 0xe8, 0x34, 0x06, 0xa8, 0x1b, 0x27, 0x90, 0x52, 0xe1, 0x45, 0x34, 0xdb, 0xc5,
	0x6c, 0xd8, 0xfa, 0xb0, 0xe9, 0x76, 0x8a, 0xe4, 0x75, 0x9e, 0x33, 0x00, 0xed, 0x56, 0xfb, 0x78,
	0xe5, 0x57, 0x7a, 0x32, 0x5b, 0xc3, 0x6c, 0xda, 0xfa, 0xb0, 0x3d, 0x3e, 0xab, 0x7b, 0x78, 0x53,
	0x0d, 0xaf, 0x75, 0xd2, 0x11, 0xde, 0xf4, 0x70, 0x7b, 0x55, 0xe5, 0x97, 0xdd, 0x0d, 0xda, 0x8e,
	0xa9, 0x98, 0x32, 0x11, 0x7a, 0x3e, 0x87, 0xe0, 0x51, 0x9a, 0xad, 0xdc, 0xf0, 0xf8, 0x1f, 0xc3,
	0x49, 0x06, 0xdd, 0x33, 0x15, 0x5d, 0xbc, 0xc4, 0x2c, 0x21, 0x8a, 0x81, 0x70, 0xbb, 0xe5, 0x84,
	0xbc, 0x26, 0x27, 0xa7, 0xf3, 0xa5, 0xa5, 0x2f, 0x96, 0x96, 0xfe, 0xb5, 0xb4, 0xf4, 0xb7, 0x95,
	0xa5, 0x2d, 0x56, 0x96, 0xf6, 0xb1, 0xb2, 0xb4, 0x87, 0xbd, 0x90, 0xa9, 0x68, 0xe6, 0xe3, 0x00,
	0x9e, 0x9c, 0xbf, 0x27, 0xf2, 0xb7, 0xf2, 0x0b, 0x9c, 0x7c, 0x07, 0x00, 0x00, 0xff, 0xff, 0x10,
	0x73, 0x2e, 0x22, 0x5b, 0x02, 0x00, 0x00,
}

func (m *GenesisState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GenesisState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GenesisState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.PendingBlocks) > 0 {
		for iNdEx := len(m.PendingBlocks) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.PendingBlocks[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x22
		}
	}
	if m.CelestiaGenesisState != nil {
		{
			size, err := m.CelestiaGenesisState.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintGenesis(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if m.ProvenHeight != 0 {
		i = encodeVarintGenesis(dAtA, i, uint64(m.ProvenHeight))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Validators) > 0 {
		for iNdEx := len(m.Validators) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Validators[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintGenesis(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintGenesis(dAtA []byte, offset int, v uint64) int {
	offset -= sovGenesis(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GenesisState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Validators) > 0 {
		for _, e := range m.Validators {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	if m.ProvenHeight != 0 {
		n += 1 + sovGenesis(uint64(m.ProvenHeight))
	}
	if m.CelestiaGenesisState != nil {
		l = m.CelestiaGenesisState.Size()
		n += 1 + l + sovGenesis(uint64(l))
	}
	if len(m.PendingBlocks) > 0 {
		for _, e := range m.PendingBlocks {
			l = e.Size()
			n += 1 + l + sovGenesis(uint64(l))
		}
	}
	return n
}

func sovGenesis(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozGenesis(x uint64) (n int) {
	return sovGenesis(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GenesisState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowGenesis
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
			return fmt.Errorf("proto: GenesisState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GenesisState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Validators", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Validators = append(m.Validators, Validator{})
			if err := m.Validators[len(m.Validators)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProvenHeight", wireType)
			}
			m.ProvenHeight = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProvenHeight |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CelestiaGenesisState", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CelestiaGenesisState == nil {
				m.CelestiaGenesisState = &celestia.GenesisState{}
			}
			if err := m.CelestiaGenesisState.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
					return ErrIntOverflowGenesis
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
				return ErrInvalidLengthGenesis
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthGenesis
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PendingBlocks = append(m.PendingBlocks, &BlockWithExpiration{})
			if err := m.PendingBlocks[len(m.PendingBlocks)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipGenesis(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthGenesis
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
func skipGenesis(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
					return 0, ErrIntOverflowGenesis
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
				return 0, ErrInvalidLengthGenesis
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupGenesis
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthGenesis
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthGenesis        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowGenesis          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupGenesis = fmt.Errorf("proto: unexpected end of group")
)
