package sshfxp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
)

const (
	TypeInit          = 1
	TypeVersion       = 2
	TypeOpen          = 3
	TypeClose         = 4
	TypeRead          = 5
	TypeWrite         = 6
	TypeLStat         = 7
	TypeFStat         = 8
	TypeSetStat       = 9
	TypeFSetStat      = 10
	TypeOpenDir       = 11
	TypeReadDir       = 12
	TypeRemove        = 13
	TypeMkDir         = 14
	TypeRmDir         = 15
	TypeRealPath      = 16
	TypeStat          = 17
	TypeRename        = 18
	TypeReadlink      = 19
	TypeSymlink       = 20
	TypeStatus        = 101
	TypeHandle        = 102
	TypeData          = 103
	TypeName          = 104
	TypeAttr          = 105
	TypeExtended      = 200
	TypeExtendedReply = 201
)

func TypeID(x interface{}) byte {
	switch x.(type) {
	case *Init:
		return TypeInit
	case *Version:
		return TypeVersion
	case *Open:
		return TypeOpen
	case *Close:
		return TypeClose
	case *Read:
		return TypeRead
	case *Write:
		return TypeWrite
	case *LStat:
		return TypeLStat
	case *FStat:
		return TypeFStat
	case *SetStat:
		return TypeSetStat
	case *FSetStat:
		return TypeFSetStat
	case *OpenDir:
		return TypeOpenDir
	case *ReadDir:
		return TypeReadDir
	case *Remove:
		return TypeRemove
	case *MkDir:
		return TypeMkDir
	case *RmDir:
		return TypeRmDir
	case *RealPath:
		return TypeRealPath
	case *Stat:
		return TypeStat
	case *Rename:
		return TypeRename
	case *ReadLink:
		return TypeReadlink
	case *Symlink:
		return TypeSymlink
	case *Status:
		return TypeStatus
	case *Handle:
		return TypeHandle
	case *Data:
		return TypeData
	case *Name:
		return TypeName
	default:
		panic(fmt.Sprintf("unknown type: %v", x))
	}
	return 0
}

type Packet struct {
	Length  uint32
	Type    byte
	Payload []byte
}

func (p *Packet) Read(r io.Reader) error {
	read := func(x interface{}) error {
		return binary.Read(r, binary.BigEndian, x)
	}

	if err := read(&p.Length); err != nil {
		return err
	}

	if err := read(&p.Type); err != nil {
		return err
	}

	p.Payload = make([]byte, p.Length-1)
	if err := read(&p.Payload); err != nil {
		return err
	}

	return nil
}

func (p *Packet) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Grow(int(p.Length) + 4)

	write := func(x interface{}) error {
		return binary.Write(buf, binary.BigEndian, x)
	}

	if err := write(p.Length); err != nil {
		return nil, err
	}

	if err := write(p.Type); err != nil {
		return nil, err
	}

	if err := write(p.Payload); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (p *Packet) Encode(x interface{}) error {
	buf := new(bytes.Buffer)

	if header, ok := x.(Header); ok {
		if err := binary.Write(buf, binary.BigEndian, header.GetID()); err != nil {
			return err
		}
	}

	if writer, ok := x.(Writer); !ok {
		return fmt.Errorf("invalid parameter: %#v does not implement sshfxp.Writer", x)
	} else {
		if err := writer.Write(buf); err != nil {
			return err
		}
	}

	data := buf.Bytes()
	p.Length = uint32(len(data) + 1)
	p.Payload = data
	p.Type = TypeID(x)

	return nil
}

type Writer interface {
	Write(io.Writer) error
}

type Reader interface {
	Read(io.Reader) error
}

func (p *Packet) Decode() (Message, error) {
	var o Message

	switch p.Type {
	case TypeInit:
		o = &Init{}
	case TypeVersion:
		o = &Version{}
	case TypeOpen:
		o = &Open{}
	case TypeClose:
		o = &Close{}
	case TypeRead:
		o = &Read{}
	case TypeWrite:
		o = &Write{}
	case TypeLStat:
		o = &LStat{}
	case TypeFStat:
		o = &FStat{}
	case TypeSetStat:
		o = &SetStat{}
	case TypeFSetStat:
		o = &FSetStat{}
	case TypeOpenDir:
		o = &OpenDir{}
	case TypeReadDir:
		o = &ReadDir{}
	case TypeRemove:
		o = &Remove{}
	case TypeMkDir:
		o = &MkDir{}
	case TypeRmDir:
		o = &RmDir{}
	case TypeRealPath:
		o = &RealPath{}
	case TypeStat:
		o = &Stat{}
	case TypeRename:
		o = &Rename{}
	case TypeReadlink:
		o = &ReadLink{}
	case TypeSymlink:
		o = &Symlink{}
	case TypeStatus:
		o = &Status{}
	case TypeHandle:
		o = &Handle{}
	case TypeData:
		o = &Data{}
	case TypeName:
		o = &Name{}
	case TypeAttr:
		o = &Attrs{}
	case TypeExtended:
		// TODO
		return nil, fmt.Errorf("not yet implemented")
	case TypeExtendedReply:
		// TODO
		return nil, fmt.Errorf("not yet implemented")
	default:
		return nil, fmt.Errorf("not yet implemented")
	}

	if int(p.Length) != len(p.Payload)+1 /* byte for type */ {
		return nil, fmt.Errorf("invalid packet length: expected %d got %d", p.Length, len(p.Payload)+1)
	}

	buf := bytes.NewBuffer(p.Payload)
	if header, ok := o.(Header); ok {
		var id uint32

		if err := binary.Read(buf, binary.BigEndian, &id); err != nil {
			return nil, fmt.Errorf("failed to read id: %s", err)
		}

		header.SetID(id)
	}

	if err := o.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to read body: %s", err)
	}

	return o, nil
}

//
// Client sent requests
//

// When the file transfer protocol starts, it first sends a SSH_FXP_INIT
// (including its version number) packet to the server.  The server
// responds with a SSH_FXP_VERSION packet, supplying the lowest of its
// own and the client's version number.  Both parties should from then
// on adhere to particular version of the protocol.
type Init struct {
	Version    uint32
	Extensions []struct {
		Name string
		Data string
	}
}

func (i *Init) Write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, i.Version)
	// TODO: write extensions
}

func (i *Init) Read(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &i.Version)
}

const (
	AttrSize        = 0x00000001
	AttrUidGid      = 0x00000002
	AttrPermissions = 0x00000004
	AttrAcModTime   = 0x00000008
	AttrExtended    = 0x80000000
)

type Attr struct {
	Flags         uint32
	Size          uint64
	UID           uint32
	GID           uint32
	Permissions   uint32
	ATime         uint32
	MTime         uint32
	ExtendedCount uint32
	Extended      []struct {
		Type string
		Data string
	}
}

func (a *Attr) Write(w io.Writer) error {
	write := func(x interface{}) error {
		return binary.Write(w, binary.BigEndian, x)
	}

	if err := write(a.Flags); err != nil {
		return err
	}

	if err := write(a.Size); err != nil {
		return err
	}

	if err := write(a.UID); err != nil {
		return err
	}

	if err := write(a.GID); err != nil {
		return err
	}

	if err := write(a.Permissions); err != nil {
		return err
	}

	if err := write(a.ATime); err != nil {
		return err
	}

	if err := write(a.MTime); err != nil {
		return err
	}

	// TODO: Extended is currently not supported
	a.ExtendedCount = 0
	if err := write(a.ExtendedCount); err != nil {
		return err
	}
	return nil
}

func (a *Attr) Read(r io.Reader) error {
	read := func(x interface{}) error {
		return binary.Read(r, binary.BigEndian, x)
	}

	if err := read(&a.Flags); err != nil {
		return err
	}

	if err := read(&a.Size); err != nil {
		return err
	}

	if err := read(&a.UID); err != nil {
		return err
	}

	if err := read(&a.GID); err != nil {
		return err
	}

	if err := read(&a.Permissions); err != nil {
		return err
	}

	if err := read(&a.ATime); err != nil {
		return err
	}

	if err := read(&a.MTime); err != nil {
		return err
	}

	if err := read(&a.ExtendedCount); err != nil {
		return err
	}
	// TODO read extened
	return nil
}

type Header interface {
	SetID(uint32)
	GetID() uint32
}

type Message interface {
	Writer

	Reader
}

const (
	// Open the file for reading.
	OpenRead = 0x00000001

	// Open the file for writing.  If both this and OpenRead are
	// specified, the file is opened for both reading and writing.
	OpenWrite = 0x00000002

	// Force all writes to append data at the end of the file.
	OpenAppend = 0x00000004

	// If this flag is specified, then a new file will be created if one
	// does not already exist (if OpenTruncate is specified, the new file will
	// be truncated to zero length if it previously exists).
	OpenCreate = 0x00000008

	// Forces an existing file with the same name to be truncated to zero
	// length when creating a file by specifying OpenCreate.
	// OpenCreate MUST also be specified if this flag is used.
	OpenTruncate = 0x00000010

	// Causes the request to fail if the named file already exists.
	// OpenCreate MUST also be specified if this flag is used.
	OpenExcl = 0x00000020
)

type Open struct {
	ID uint32

	Filename   string
	PFlags     uint32
	Attributes Attr
}

func (x *Open) SetID(id uint32) {
	x.ID = id
}

func (x *Open) GetID() uint32 {
	return x.ID
}

func (o *Open) Write(w io.Writer) error {
	if err := writeString(w, o.Filename); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, o.PFlags); err != nil {
		return err
	}

	return o.Attributes.Write(w)
}

func (o *Open) Read(r io.Reader) error {
	if err := readString(r, &o.Filename); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &o.PFlags); err != nil {
		return err
	}

	return o.Attributes.Read(r)
}

type Close struct {
	ID uint32

	Handle string
}

func (x *Close) SetID(id uint32) {
	x.ID = id
}

func (x *Close) GetID() uint32 {
	return x.ID
}

func (c *Close) Write(w io.Writer) error {
	return writeString(w, c.Handle)
}

func (c *Close) Read(r io.Reader) error {
	return readString(r, &c.Handle)
}

type Read struct {
	ID     uint32
	Handle string
	Offset uint64
	Length uint32
}

func (x *Read) SetID(id uint32) {
	x.ID = id
}

func (x *Read) GetID() uint32 {
	return x.ID
}

func (_r *Read) Write(w io.Writer) error {
	if err := writeString(w, _r.Handle); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, _r.Offset); err != nil {
		return err
	}

	return binary.Write(w, binary.BigEndian, _r.Length)
}

func (_r *Read) Read(r io.Reader) error {
	if err := readString(r, &_r.Handle); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &_r.Offset); err != nil {
		return err
	}

	return binary.Read(r, binary.BigEndian, &_r.Length)
}

type Write struct {
	ID uint32

	Handle string
	Offset uint64
	Length uint32

	Data []byte
}

func (x *Write) SetID(id uint32) {
	x.ID = id
}

func (x *Write) GetID() uint32 {
	return x.ID
}

func (_w *Write) Write(w io.Writer) error {
	if err := writeString(w, _w.Handle); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, _w.Offset); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, _w.Length); err != nil {
		return err
	}

	return binary.Write(w, binary.BigEndian, _w.Data)
}

func (_w Write) Read(r io.Reader) error {
	if err := readString(r, &_w.Handle); err != nil {
		return err
	}

	if err := binary.Read(r, binary.BigEndian, &_w.Offset); err != nil {
		return nil
	}

	if err := binary.Read(r, binary.BigEndian, &_w.Length); err != nil {
		return nil
	}

	_w.Data = make([]byte, _w.Length)
	return binary.Read(r, binary.BigEndian, &_w.Data)
}

type Remove struct {
	ID uint32

	File string
}

func (x *Remove) SetID(id uint32) {
	x.ID = id
}

func (x *Remove) GetID() uint32 {
	return x.ID
}

func (rm *Remove) Write(w io.Writer) error {
	return writeString(w, rm.File)
}

func (rm *Remove) Read(r io.Reader) error {
	return readString(r, &rm.File)
}

type Rename struct {
	ID uint32

	OldPath string
	NewPath string
}

func (x *Rename) SetID(id uint32) {
	x.ID = id
}

func (x *Rename) GetID() uint32 {
	return x.ID
}

func (rn *Rename) Write(w io.Writer) error {
	if err := writeString(w, rn.OldPath); err != nil {
		return err
	}

	return writeString(w, rn.NewPath)
}

func (rn *Rename) Read(r io.Reader) error {
	if err := readString(r, &rn.OldPath); err != nil {
		return err
	}

	return readString(r, &rn.NewPath)
}

type MkDir struct {
	ID uint32

	Path string
	Attr Attr
}

func (x *MkDir) SetID(id uint32) {
	x.ID = id
}

func (x *MkDir) GetID() uint32 {
	return x.ID
}

func (mk *MkDir) Write(w io.Writer) error {
	if err := writeString(w, mk.Path); err != nil {
		return err
	}

	return mk.Attr.Write(w)
}

func (mk *MkDir) Read(r io.Reader) error {
	if err := readString(r, &mk.Path); err != nil {
		return err
	}

	return mk.Attr.Read(r)
}

type RmDir struct {
	ID uint32

	Path string
}

func (x *RmDir) SetID(id uint32) {
	x.ID = id
}

func (x *RmDir) GetID() uint32 {
	return x.ID
}

func (rm *RmDir) Write(w io.Writer) error {
	return writeString(w, rm.Path)
}

func (rm *RmDir) Read(r io.Reader) error {
	return readString(r, &rm.Path)
}

type OpenDir struct {
	ID uint32

	Path string
}

func (x *OpenDir) SetID(id uint32) {
	x.ID = id
}

func (x *OpenDir) GetID() uint32 {
	return x.ID
}

func (o *OpenDir) Write(w io.Writer) error {
	return writeString(w, o.Path)
}

func (o *OpenDir) Read(r io.Reader) error {
	return readString(r, &o.Path)
}

type ReadDir struct {
	ID uint32

	Handle string
}

func (x *ReadDir) SetID(id uint32) {
	x.ID = id
}

func (x *ReadDir) GetID() uint32 {
	return x.ID
}

func (o *ReadDir) Write(w io.Writer) error {
	return writeString(w, o.Handle)
}

func (o *ReadDir) Read(r io.Reader) error {
	return readString(r, &o.Handle)
}

type Stat struct {
	ID uint32

	Handle string
}

func (x *Stat) SetID(id uint32) {
	x.ID = id
}

func (x *Stat) GetID() uint32 {
	return x.ID
}

func (s *Stat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *Stat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type LStat struct {
	ID uint32

	Handle string
}

func (x *LStat) SetID(id uint32) {
	x.ID = id
}

func (x *LStat) GetID() uint32 {
	return x.ID
}

func (s *LStat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *LStat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type FStat struct {
	ID uint32

	Handle string
}

func (x *FStat) SetID(id uint32) {
	x.ID = id
}

func (x *FStat) GetID() uint32 {
	return x.ID
}

func (s *FStat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *FStat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type SetStat struct {
	ID uint32

	Path string
	Attr Attr
}

func (x *SetStat) SetID(id uint32) {
	x.ID = id
}

func (x *SetStat) GetID() uint32 {
	return x.ID
}

func (ss *SetStat) Write(w io.Writer) error {
	if err := writeString(w, ss.Path); err != nil {
		return err
	}

	return ss.Attr.Write(w)
}

func (ss *SetStat) Read(r io.Reader) error {
	if err := readString(r, &ss.Path); err != nil {
		return err
	}

	return ss.Attr.Read(r)
}

type FSetStat struct {
	ID uint32

	Path string
	Attr Attr
}

func (x *FSetStat) SetID(id uint32) {
	x.ID = id
}

func (x *FSetStat) GetID() uint32 {
	return x.ID
}

func (ss *FSetStat) Write(w io.Writer) error {
	if err := writeString(w, ss.Path); err != nil {
		return err
	}

	return ss.Attr.Write(w)
}

func (ss *FSetStat) Read(r io.Reader) error {
	if err := readString(r, &ss.Path); err != nil {
		return err
	}

	return ss.Attr.Read(r)
}

type ReadLink struct {
	ID uint32

	Path string
}

func (x *ReadLink) SetID(id uint32) {
	x.ID = id
}

func (x *ReadLink) GetID() uint32 {
	return x.ID
}

func (s *ReadLink) Write(w io.Writer) error {
	return writeString(w, s.Path)
}

func (s *ReadLink) Read(r io.Reader) error {
	return readString(r, &s.Path)
}

type Symlink struct {
	ID uint32

	LinkPath   string
	TargetPath string
}

func (x *Symlink) SetID(id uint32) {
	x.ID = id
}

func (x *Symlink) GetID() uint32 {
	return x.ID
}

func (s *Symlink) Write(w io.Writer) error {
	if err := writeString(w, s.LinkPath); err != nil {
		return err
	}
	return writeString(w, s.TargetPath)
}

func (s *Symlink) Read(r io.Reader) error {
	if err := readString(r, &s.LinkPath); err != nil {
		return err
	}
	return readString(r, &s.TargetPath)
}

type RealPath struct {
	ID uint32

	Path string
}

func (x *RealPath) SetID(id uint32) {
	x.ID = id
}

func (x *RealPath) GetID() uint32 {
	return x.ID
}

func (s *RealPath) Write(w io.Writer) error {
	return writeString(w, s.Path)
}

func (s *RealPath) Read(r io.Reader) error {
	return readString(r, &s.Path)
}

//
// Server sent response messages
//

const (
	StatusOK = iota
	StatusEOF
	StatusNoSuchFile
	StatusPermissionDenied
	StatusFailure
	StatusBadMessage
	StatusNoConnection
	StatusConnectionLost
	StatusOpUnsupported
)

type Status struct {
	ID uint32

	Error    uint32
	Message  string
	Language string
}

func (x *Status) SetID(id uint32) {
	x.ID = id
}

func (x *Status) GetID() uint32 {
	return x.ID
}

func (s *Status) Write(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, s.Error); err != nil {
		return err
	}

	if err := writeString(w, s.Message); err != nil {
		return err
	}

	return writeString(w, s.Language)
}

func (s *Status) Read(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &s.Error); err != nil {
		return err
	}

	if err := readString(r, &s.Message); err != nil {
		return err
	}

	return readString(r, &s.Language)
}

type Handle struct {
	ID uint32

	Handle string
}

func (x *Handle) SetID(id uint32) {
	x.ID = id
}

func (x *Handle) GetID() uint32 {
	return x.ID
}

func (s *Handle) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *Handle) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type Data struct {
	ID uint32

	Data string
}

func (x *Data) SetID(id uint32) {
	x.ID = id
}

func (x *Data) GetID() uint32 {
	return x.ID
}

func (s *Data) Write(w io.Writer) error {
	return writeString(w, s.Data)
}

func (s *Data) Read(r io.Reader) error {
	return readString(r, &s.Data)
}

type Name struct {
	ID uint32

	Count uint32
	Names []struct {
		Filename string
		Longname string
		Attr     Attr
	}
}

func (n *Name) Write(w io.Writer) error {
	n.Count = uint32(len(n.Names))

	if err := binary.Write(w, binary.BigEndian, n.Count); err != nil {
		return err
	}

	for _, name := range n.Names {
		if err := writeString(w, name.Filename); err != nil {
			return err
		}

		if err := writeString(w, name.Longname); err != nil {
			return err
		}

		if err := name.Attr.Write(w); err != nil {
			return err
		}
	}

	return fmt.Errorf("Not yet implemented")
}

func (n Name) Read(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &n.Count); err != nil {
		return err
	}

	for i := 0; i < int(n.Count); i++ {
		var filename string
		var longname string
		var attr Attr

		if err := readString(r, &filename); err != nil {
			return err
		}

		if err := readString(r, &longname); err != nil {
			return err
		}

		if err := attr.Read(r); err != nil {
			return err
		}

		n.Names = append(n.Names, struct {
			Filename string
			Longname string
			Attr     Attr
		}{filename, longname, attr})

	}

	return fmt.Errorf("Not yet implemented")
}

type Attrs struct {
	ID uint32

	Attr Attr
}

func (a *Attrs) Write(w io.Writer) error {
	return a.Attr.Write(w)
}

func (a *Attrs) Read(r io.Reader) error {
	return a.Attr.Read(r)
}

type Version struct {
	Version    uint32
	Extensions []struct {
		Name string
		Data string
	}
}

func (v *Version) Write(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, v.Version)
	// TODO: write extensions
}

func (v *Version) Read(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, &v.Version)
}

//
// internals and helper functions
//

func readString(r io.Reader, v *string) error {
	var length uint32

	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return err
	}

	logrus.Infof("reading string with %d byte", length)

	buf := make([]byte, length)
	if err := binary.Read(r, binary.BigEndian, &buf); err != nil {
		return err
	}

	*v = string(buf)

	return nil
}

func writeString(w io.Writer, s string) error {
	if err := binary.Write(w, binary.BigEndian, uint32(len(s))); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, []byte(s)); err != nil {
		return err
	}

	return nil
}
