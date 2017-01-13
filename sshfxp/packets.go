package sshfxp

import (
	"bytes"
	"encoding/binary"
	"io"
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

type Packet struct {
	Length  uint32
	Type    byte
	Payload []byte
}

func (p *Packet) Read(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &p.Length); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &p.Type); err != nil {
		return err
	}

	p.Payload = make([]byte, p.Length-1)
	if err := binary.Read(r, binary.LittleEndian, &p.Payload); err != nil {
		return err
	}

	return nil
}

type Reader interface {
	Read(io.Reader) error
}

func (p *Packet) Decode() (interface{}, error) {
	var o Reader

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
	case TypeExtendedReply:
	}

	buf := bytes.NewBuffer(p.Payload)

	if err := o.(GetMeta).Meta().ReadMeta(buf); err != nil {
		return nil, err
	}

	if err := o.Read(buf); err != nil {
		return nil, err
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

func (i Init) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, i.Version)
	// TODO: write extensions
}

func (i *Init) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &i.Version)
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

func (a Attr) Write(w io.Writer) error {
	write := func(x interface{}) error {
		return binary.Write(w, binary.LittleEndian, x)
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

	if err := write(a.ExtendedCount); err != nil {
		return err
	}

	// TODO: write extened
	return nil
}

func (a *Attr) Read(r io.Reader) error {
	read := func(x interface{}) error {
		return binary.Read(r, binary.LittleEndian, x)
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

type Meta struct {
	ID uint32
}

func (m *Meta) Meta() *Meta {
	return m
}

type GetMeta interface {
	Meta() *Meta
}

func (m Meta) WriteMeta(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, m.ID)
}

func (m *Meta) ReadMeta(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &m.ID)
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
	Meta

	Filename   string
	PFlags     uint32
	Attributes Attr
}

func (o Open) Write(w io.Writer) error {
	if err := writeString(w, o.Filename); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, o.PFlags); err != nil {
		return err
	}

	return o.Attributes.Write(w)
}

func (o *Open) Read(r io.Reader) error {
	if err := readString(r, &o.Filename); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &o.PFlags); err != nil {
		return err
	}

	return o.Attributes.Read(r)
}

type Close struct {
	Meta
	Handle string
}

func (c Close) Write(w io.Writer) error {
	return writeString(w, c.Handle)
}

func (c *Close) Read(r io.Reader) error {
	return readString(r, &c.Handle)
}

type Read struct {
	Meta
	Handle string
	Offset uint64
	Length uint32
}

func (_r Read) Write(w io.Writer) error {
	if err := writeString(w, _r.Handle); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, _r.Offset); err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, _r.Length)
}

func (_r *Read) Read(r io.Reader) error {
	if err := readString(r, &_r.Handle); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &_r.Offset); err != nil {
		return err
	}

	return binary.Read(r, binary.LittleEndian, &_r.Length)
}

type Write struct {
	Meta

	Handle string
	Offset uint64
	Length uint32

	Data []byte
}

func (_w Write) Write(w io.Writer) error {
	if err := writeString(w, _w.Handle); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, _w.Offset); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, _w.Length); err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, _w.Data)
}

func (_w Write) Read(r io.Reader) error {
	if err := readString(r, &_w.Handle); err != nil {
		return err
	}

	if err := binary.Read(r, binary.LittleEndian, &_w.Offset); err != nil {
		return nil
	}

	if err := binary.Read(r, binary.LittleEndian, &_w.Length); err != nil {
		return nil
	}

	_w.Data = make([]byte, _w.Length)
	return binary.Read(r, binary.LittleEndian, &_w.Data)
}

type Remove struct {
	Meta

	File string
}

func (rm Remove) Write(w io.Writer) error {
	return writeString(w, rm.File)
}

func (rm *Remove) Read(r io.Reader) error {
	return readString(r, &rm.File)
}

type Rename struct {
	Meta

	OldPath string
	NewPath string
}

func (rn Rename) Write(w io.Writer) error {
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
	Meta

	Path string
	Attr Attr
}

func (mk MkDir) Write(w io.Writer) error {
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
	Meta

	Path string
}

func (rm RmDir) Write(w io.Writer) error {
	return writeString(w, rm.Path)
}

func (rm *RmDir) Read(r io.Reader) error {
	return readString(r, &rm.Path)
}

type OpenDir struct {
	Meta

	Path string
}

func (o OpenDir) Write(w io.Writer) error {
	return writeString(w, o.Path)
}

func (o *OpenDir) Read(r io.Reader) error {
	return readString(r, &o.Path)
}

type ReadDir struct {
	Meta

	Handle string
}

func (o ReadDir) Write(w io.Writer) error {
	return writeString(w, o.Handle)
}

func (o *ReadDir) Read(r io.Reader) error {
	return readString(r, &o.Handle)
}

type Stat struct {
	Meta

	Handle string
}

func (s Stat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *Stat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type LStat struct {
	Meta

	Handle string
}

func (s LStat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *LStat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type FStat struct {
	Meta

	Handle string
}

func (s FStat) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *FStat) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type SetStat struct {
	Meta

	Path string
	Attr Attr
}

func (ss SetStat) Write(w io.Writer) error {
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
	Meta

	Path string
	Attr Attr
}

func (ss FSetStat) Write(w io.Writer) error {
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
	Meta

	Path string
}

func (s ReadLink) Write(w io.Writer) error {
	return writeString(w, s.Path)
}

func (s *ReadLink) Read(r io.Reader) error {
	return readString(r, &s.Path)
}

type Symlink struct {
	Meta

	LinkPath   string
	TargetPath string
}

func (s Symlink) Write(w io.Writer) error {
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
	Meta

	Path string
}

func (s RealPath) Write(w io.Writer) error {
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
	Meta

	Error    uint32
	Message  string
	Language string
}

func (s Status) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, s.Error); err != nil {
		return err
	}

	if err := writeString(w, s.Message); err != nil {
		return err
	}

	return writeString(w, s.Language)
}

func (s *Status) Read(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &s.Error); err != nil {
		return err
	}

	if err := readString(r, &s.Message); err != nil {
		return err
	}

	return readString(r, &s.Language)
}

type Handle struct {
	Meta

	Handle string
}

func (s Handle) Write(w io.Writer) error {
	return writeString(w, s.Handle)
}

func (s *Handle) Read(r io.Reader) error {
	return readString(r, &s.Handle)
}

type Data struct {
	Meta

	Data string
}

func (s Data) Write(w io.Writer) error {
	return writeString(w, s.Data)
}

func (s *Data) Read(r io.Reader) error {
	return readString(r, &s.Data)
}

type Name struct {
	Meta

	Count uint32
	Names struct {
		Filename string
		Longname string
		Attr     Attr
	}
}

type Attrs struct {
	Meta

	Attr Attr
}

type Version struct {
	Version    uint32
	Extensions []struct {
		Name string
		Data string
	}
}

func (v Version) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, v.Version)
	// TODO: write extensions
}

func (v *Version) Read(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &v.Version)
}

//
// internals and helper functions
//

func readString(r io.Reader, v *string) error {
	var length uint32

	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return err
	}

	buf := make([]byte, length)
	if err := binary.Read(r, binary.LittleEndian, &buf); err != nil {
		return err
	}

	*v = string(buf)

	return nil
}

func writeString(w io.Writer, s string) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(s))); err != nil {
		return err
	}

	if n, err := w.Write([]byte(s)); n != len(s) && err != nil {
		return err
	}

	return nil
}
