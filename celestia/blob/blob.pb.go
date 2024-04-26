// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: celestia/core/v1/blob/blob.proto

package blob

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	proto "github.com/cosmos/gogoproto/proto"
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

// Blob (named after binary large object) is a chunk of data submitted by a user
// to be published to the Celestia blockchain. The data of a Blob is published
// to a namespace and is encoded into shares based on the format specified by
// share_version.
type Blob struct {
	NamespaceId      []byte `protobuf:"bytes,1,opt,name=namespace_id,json=namespaceId,proto3" json:"namespace_id,omitempty"`
	Data             []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	ShareVersion     uint32 `protobuf:"varint,3,opt,name=share_version,json=shareVersion,proto3" json:"share_version,omitempty"`
	NamespaceVersion uint32 `protobuf:"varint,4,opt,name=namespace_version,json=namespaceVersion,proto3" json:"namespace_version,omitempty"`
}

func (m *Blob) Reset()         { *m = Blob{} }
func (m *Blob) String() string { return proto.CompactTextString(m) }
func (*Blob) ProtoMessage()    {}
func (*Blob) Descriptor() ([]byte, []int) {
	return fileDescriptor_c6da8a12c2dbf976, []int{0}
}
func (m *Blob) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Blob) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Blob.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Blob) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Blob.Merge(m, src)
}
func (m *Blob) XXX_Size() int {
	return m.Size()
}
func (m *Blob) XXX_DiscardUnknown() {
	xxx_messageInfo_Blob.DiscardUnknown(m)
}

var xxx_messageInfo_Blob proto.InternalMessageInfo

func (m *Blob) GetNamespaceId() []byte {
	if m != nil {
		return m.NamespaceId
	}
	return nil
}

func (m *Blob) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *Blob) GetShareVersion() uint32 {
	if m != nil {
		return m.ShareVersion
	}
	return 0
}

func (m *Blob) GetNamespaceVersion() uint32 {
	if m != nil {
		return m.NamespaceVersion
	}
	return 0
}

// BlobTx wraps an encoded sdk.Tx with a second field to contain blobs of data.
// The raw bytes of the blobs are not signed over, instead we verify each blob
// using the relevant MsgPayForBlobs that is signed over in the encoded sdk.Tx.
type BlobTx struct {
	Tx     []byte  `protobuf:"bytes,1,opt,name=tx,proto3" json:"tx,omitempty"`
	Blobs  []*Blob `protobuf:"bytes,2,rep,name=blobs,proto3" json:"blobs,omitempty"`
	TypeId string  `protobuf:"bytes,3,opt,name=type_id,json=typeId,proto3" json:"type_id,omitempty"`
}

func (m *BlobTx) Reset()         { *m = BlobTx{} }
func (m *BlobTx) String() string { return proto.CompactTextString(m) }
func (*BlobTx) ProtoMessage()    {}
func (*BlobTx) Descriptor() ([]byte, []int) {
	return fileDescriptor_c6da8a12c2dbf976, []int{1}
}
func (m *BlobTx) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *BlobTx) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_BlobTx.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *BlobTx) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BlobTx.Merge(m, src)
}
func (m *BlobTx) XXX_Size() int {
	return m.Size()
}
func (m *BlobTx) XXX_DiscardUnknown() {
	xxx_messageInfo_BlobTx.DiscardUnknown(m)
}

var xxx_messageInfo_BlobTx proto.InternalMessageInfo

func (m *BlobTx) GetTx() []byte {
	if m != nil {
		return m.Tx
	}
	return nil
}

func (m *BlobTx) GetBlobs() []*Blob {
	if m != nil {
		return m.Blobs
	}
	return nil
}

func (m *BlobTx) GetTypeId() string {
	if m != nil {
		return m.TypeId
	}
	return ""
}

func init() {
	proto.RegisterType((*Blob)(nil), "celestia.core.v1.blob.Blob")
	proto.RegisterType((*BlobTx)(nil), "celestia.core.v1.blob.BlobTx")
}

func init() { proto.RegisterFile("celestia/core/v1/blob/blob.proto", fileDescriptor_c6da8a12c2dbf976) }

var fileDescriptor_c6da8a12c2dbf976 = []byte{
	// 289 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0xc1, 0x4a, 0xc3, 0x30,
	0x1c, 0xc6, 0x97, 0xae, 0x56, 0xcc, 0x3a, 0xd1, 0x80, 0x58, 0x10, 0x42, 0x9d, 0x97, 0xc2, 0x20,
	0xa5, 0xfa, 0x06, 0x03, 0x0f, 0xbb, 0x16, 0xf1, 0xe0, 0x65, 0xa4, 0x6d, 0xb0, 0x81, 0xae, 0x29,
	0x49, 0x2c, 0xf5, 0x29, 0xf4, 0xb1, 0x3c, 0xee, 0xe8, 0x51, 0xda, 0x17, 0x91, 0x64, 0x5b, 0xbd,
	0x78, 0x09, 0xf9, 0x7f, 0xdf, 0x2f, 0xe1, 0xfb, 0x7f, 0x30, 0xcc, 0x59, 0xc5, 0x94, 0xe6, 0x34,
	0xce, 0x85, 0x64, 0x71, 0x9b, 0xc4, 0x59, 0x25, 0x32, 0x7b, 0x90, 0x46, 0x0a, 0x2d, 0xd0, 0xd5,
	0x91, 0x20, 0x86, 0x20, 0x6d, 0x42, 0x8c, 0xb9, 0xf8, 0x00, 0xd0, 0x5d, 0x55, 0x22, 0x43, 0xb7,
	0xd0, 0xaf, 0xe9, 0x96, 0xa9, 0x86, 0xe6, 0x6c, 0xc3, 0x8b, 0x00, 0x84, 0x20, 0xf2, 0xd3, 0xd9,
	0xa8, 0xad, 0x0b, 0x84, 0xa0, 0x5b, 0x50, 0x4d, 0x03, 0xc7, 0x5a, 0xf6, 0x8e, 0xee, 0xe0, 0x5c,
	0x95, 0x54, 0xb2, 0x4d, 0xcb, 0xa4, 0xe2, 0xa2, 0x0e, 0xa6, 0x21, 0x88, 0xe6, 0xa9, 0x6f, 0xc5,
	0xe7, 0xbd, 0x86, 0x96, 0xf0, 0xf2, 0xef, 0xef, 0x23, 0xe8, 0x5a, 0xf0, 0x62, 0x34, 0x0e, 0xf0,
	0xa2, 0x80, 0x9e, 0x09, 0xf4, 0xd4, 0xa1, 0x73, 0xe8, 0xe8, 0xee, 0x10, 0xc4, 0xd1, 0x1d, 0x4a,
	0xe0, 0x89, 0xc9, 0xac, 0x02, 0x27, 0x9c, 0x46, 0xb3, 0xfb, 0x1b, 0xf2, 0xef, 0x4a, 0xc4, 0xbc,
	0x4e, 0xf7, 0x24, 0xba, 0x86, 0xa7, 0xfa, 0xbd, 0xb1, 0x0b, 0x99, 0x60, 0x67, 0xa9, 0x67, 0xc6,
	0x75, 0xb1, 0x7a, 0xfc, 0xea, 0x31, 0xd8, 0xf5, 0x18, 0xfc, 0xf4, 0x18, 0x7c, 0x0e, 0x78, 0xb2,
	0x1b, 0xf0, 0xe4, 0x7b, 0xc0, 0x93, 0x97, 0xe5, 0x2b, 0xd7, 0xe5, 0x5b, 0x46, 0x72, 0xb1, 0x8d,
	0xa5, 0xa8, 0xaa, 0xbc, 0xa4, 0xbc, 0x56, 0xb1, 0xe6, 0xd4, 0x56, 0x3a, 0x16, 0x6d, 0xa6, 0xcc,
	0xb3, 0xe5, 0x3e, 0xfc, 0x06, 0x00, 0x00, 0xff, 0xff, 0x26, 0xc7, 0x14, 0x6f, 0x80, 0x01, 0x00,
	0x00,
}

func (m *Blob) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Blob) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Blob) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.NamespaceVersion != 0 {
		i = encodeVarintBlob(dAtA, i, uint64(m.NamespaceVersion))
		i--
		dAtA[i] = 0x20
	}
	if m.ShareVersion != 0 {
		i = encodeVarintBlob(dAtA, i, uint64(m.ShareVersion))
		i--
		dAtA[i] = 0x18
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintBlob(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.NamespaceId) > 0 {
		i -= len(m.NamespaceId)
		copy(dAtA[i:], m.NamespaceId)
		i = encodeVarintBlob(dAtA, i, uint64(len(m.NamespaceId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *BlobTx) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *BlobTx) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *BlobTx) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.TypeId) > 0 {
		i -= len(m.TypeId)
		copy(dAtA[i:], m.TypeId)
		i = encodeVarintBlob(dAtA, i, uint64(len(m.TypeId)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Blobs) > 0 {
		for iNdEx := len(m.Blobs) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Blobs[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintBlob(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Tx) > 0 {
		i -= len(m.Tx)
		copy(dAtA[i:], m.Tx)
		i = encodeVarintBlob(dAtA, i, uint64(len(m.Tx)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintBlob(dAtA []byte, offset int, v uint64) int {
	offset -= sovBlob(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Blob) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.NamespaceId)
	if l > 0 {
		n += 1 + l + sovBlob(uint64(l))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovBlob(uint64(l))
	}
	if m.ShareVersion != 0 {
		n += 1 + sovBlob(uint64(m.ShareVersion))
	}
	if m.NamespaceVersion != 0 {
		n += 1 + sovBlob(uint64(m.NamespaceVersion))
	}
	return n
}

func (m *BlobTx) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Tx)
	if l > 0 {
		n += 1 + l + sovBlob(uint64(l))
	}
	if len(m.Blobs) > 0 {
		for _, e := range m.Blobs {
			l = e.Size()
			n += 1 + l + sovBlob(uint64(l))
		}
	}
	l = len(m.TypeId)
	if l > 0 {
		n += 1 + l + sovBlob(uint64(l))
	}
	return n
}

func sovBlob(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozBlob(x uint64) (n int) {
	return sovBlob(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Blob) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlob
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
			return fmt.Errorf("proto: Blob: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Blob: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field NamespaceId", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.NamespaceId = append(m.NamespaceId[:0], dAtA[iNdEx:postIndex]...)
			if m.NamespaceId == nil {
				m.NamespaceId = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ShareVersion", wireType)
			}
			m.ShareVersion = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ShareVersion |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NamespaceVersion", wireType)
			}
			m.NamespaceVersion = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NamespaceVersion |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipBlob(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlob
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
func (m *BlobTx) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowBlob
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
			return fmt.Errorf("proto: BlobTx: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: BlobTx: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Tx", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Tx = append(m.Tx[:0], dAtA[iNdEx:postIndex]...)
			if m.Tx == nil {
				m.Tx = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Blobs", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
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
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Blobs = append(m.Blobs, &Blob{})
			if err := m.Blobs[len(m.Blobs)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TypeId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowBlob
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthBlob
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthBlob
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TypeId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipBlob(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthBlob
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
func skipBlob(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowBlob
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
					return 0, ErrIntOverflowBlob
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
					return 0, ErrIntOverflowBlob
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
				return 0, ErrInvalidLengthBlob
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupBlob
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthBlob
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthBlob        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowBlob          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupBlob = fmt.Errorf("proto: unexpected end of group")
)
