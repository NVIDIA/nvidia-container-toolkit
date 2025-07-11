package pciids

import (
	"bufio"
	"bytes"
	_ "embed" // Fallback is the embedded pci.ids db file
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// token what the Lexer retruns.
type token int

const (
	// ILLEGAL a token which the Lexer does not understand.
	ILLEGAL token = iota
	// EOF end of file.
	EOF
	// WS whitespace.
	WS
	// NEWLINE '\n'.
	NEWLINE
	// COMMENT '# something'.
	COMMENT
	// VENDOR PCI vendor.
	VENDOR
	// SUBVENDOR PCI subvendor.
	SUBVENDOR
	// DEVICE PCI device.
	DEVICE
	// CLASS PCI class.
	CLASS
	// SUBCLASS PCI subclass.
	SUBCLASS
	// PROGIF PCI programming interface.
	PROGIF
)

// literal values from the Lexer.
type literal struct {
	ID      string
	name    string
	SubName string
}

// scanner a lexical scanner.
type scanner struct {
	r        *bufio.Reader
	isVendor bool
}

// newScanner well a new scanner ...
func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r)}
}

// Since the pci.ids is line base we're consuming a whole line rather then only
// a single rune/char.
func (s *scanner) readline() []byte {
	ln, err := s.r.ReadBytes('\n')
	if err == io.EOF {
		return []byte{'E', 'O', 'F'}
	}
	if err != nil {
		fmt.Printf("ReadBytes failed with %v", err)
		return []byte{}
	}
	return ln
}

func scanClass(line []byte) (token, literal) {
	class := string(line[1:])
	return CLASS, scanEntry([]byte(class), 2)
}

func scanSubVendor(line []byte) (token, literal) {
	trim0 := strings.TrimSpace(string(line))
	subv := string(trim0[:4])
	trim1 := strings.TrimSpace(trim0[4:])
	subd := string(trim1[:4])
	subn := strings.TrimSpace(trim1[4:])

	return SUBVENDOR, literal{subv, subd, subn}
}

func scanEntry(line []byte, offset uint) literal {
	trim := strings.TrimSpace(string(line))
	id := string(trim[:offset])
	name := strings.TrimSpace(trim[offset:])

	return literal{id, name, ""}
}

func isLeadingOneTab(ln []byte) bool  { return (ln[0] == '\t') && (ln[1] != '\t') }
func isLeadingTwoTabs(ln []byte) bool { return (ln[0] == '\t') && (ln[1] == '\t') }

func isHexDigit(ln []byte) bool  { return (ln[0] >= '0' && ln[0] <= '9') }
func isHexLetter(ln []byte) bool { return (ln[0] >= 'a' && ln[0] <= 'f') }

func isVendor(ln []byte) bool    { return isHexDigit(ln) || isHexLetter(ln) }
func isEOF(ln []byte) bool       { return (ln[0] == 'E' && ln[1] == 'O' && ln[2] == 'F') }
func isComment(ln []byte) bool   { return (ln[0] == '#') }
func isSubVendor(ln []byte) bool { return isLeadingTwoTabs(ln) }
func isDevice(ln []byte) bool    { return isLeadingOneTab(ln) }
func isNewline(ln []byte) bool   { return (ln[0] == '\n') }

// List of known device classes, subclasses and programming interfaces.
func isClass(ln []byte) bool    { return (ln[0] == 'C') }
func isProgIf(ln []byte) bool   { return isLeadingTwoTabs(ln) }
func isSubClass(ln []byte) bool { return isLeadingOneTab(ln) }

// unread places the previously read rune back on the reader.
func (s *scanner) unread() { _ = s.r.UnreadRune() }

// scan returns the next token and literal value.
func (s *scanner) scan() (tok token, lit literal) {

	line := s.readline()

	if isEOF(line) {
		return EOF, literal{}
	}

	if isNewline(line) {
		return NEWLINE, literal{ID: string('\n')}
	}

	if isComment(line) {
		return COMMENT, literal{ID: string(line)}
	}

	// vendors
	if isVendor(line) {
		s.isVendor = true
		return VENDOR, scanEntry(line, 4)
	}

	if isSubVendor(line) && s.isVendor {
		return scanSubVendor(line)
	}

	if isDevice(line) && s.isVendor {
		return DEVICE, scanEntry(line, 4)
	}

	// classes
	if isClass(line) {
		s.isVendor = false
		return scanClass(line)
	}
	if isProgIf(line) && !s.isVendor {
		return PROGIF, scanEntry(line, 2)
	}

	if isSubClass(line) && !s.isVendor {
		return SUBCLASS, scanEntry(line, 2)
	}

	return ILLEGAL, literal{ID: string(line)}
}

// parser reads the tokens returned by the Lexer and constructs the AST.
type parser struct {
	s   *scanner
	buf struct {
		tok token
		lit literal
		n   int
	}
}

// Various locations of pci.ids for different distributions. These may be more
// up to date then the embedded pci.ids db.
var defaultPCIdbPaths = []string{
	"/usr/share/misc/pci.ids",   // Ubuntu
	"/usr/local/share/pci.ids",  // RHEL like with manual update
	"/usr/share/hwdata/pci.ids", // RHEL like
	"/usr/share/pci.ids",        // SUSE
}

// This is a fallback if all of the locations fail
//
//go:embed default_pci.ids
var defaultPCIdb []byte

// NewDB Parse the PCI DB in its default locations or use the default
// builtin pci.ids db.
func NewDB(opts ...Option) Interface {
	db := &pcidb{}
	for _, opt := range opts {
		opt(db)
	}

	pcidbs := defaultPCIdbPaths
	if db.path != "" {
		pcidbs = append([]string{db.path}, defaultPCIdbPaths...)
	}

	return newParser(pcidbs).parse()
}

// Option defines a function for passing options to the NewDB() call.
type Option func(*pcidb)

// WithFilePath provides an Option to set the file path
// for the pciids database used by pciids interface.
// The file path provided takes precedence over all other
// paths.
func WithFilePath(path string) Option {
	return func(db *pcidb) {
		db.path = path
	}
}

// newParser will attempt to read the db pci.ids from well known places or fall
// back to an internal db.
func newParser(pcidbs []string) *parser {

	for _, db := range pcidbs {
		file, err := os.ReadFile(db)
		if err != nil {
			continue
		}
		return newParserFromReader(bufio.NewReader(bytes.NewReader(file)))

	}
	// We're using go embed above to have the byte array
	// correctly initialized with the internal shipped db
	// if we cannot find an up to date in the filesystem.
	return newParserFromReader(bufio.NewReader(bytes.NewReader(defaultPCIdb)))
}

func newParserFromReader(r *bufio.Reader) *parser {
	return &parser{s: newScanner(r)}
}

func (p *parser) scan() (tok token, lit literal) {

	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}
	tok, lit = p.s.scan()
	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *parser) unscan() { p.buf.n = 1 }

var _ Interface = (*pcidb)(nil)

// Interface returns textual description of specific attributes of PCI devices.
type Interface interface {
	GetDeviceName(uint16, uint16) (string, error)
	GetClassName(uint32) (string, error)
}

// GetDeviceName return the textual description of the PCI device.
func (d *pcidb) GetDeviceName(vendorID uint16, deviceID uint16) (string, error) {
	vendor, ok := d.vendors[vendorID]
	if !ok {
		return "", fmt.Errorf("failed to find vendor with id '%x'", vendorID)
	}

	device, ok := vendor.devices[deviceID]
	if !ok {
		return "", fmt.Errorf("failed to find device with id '%x'", deviceID)
	}

	return device.name, nil
}

// GetClassName resturn the textual description of the PCI device class.
func (d *pcidb) GetClassName(classID uint32) (string, error) {
	class, ok := d.classes[classID]
	if !ok {
		return "", fmt.Errorf("failed to find class with id '%x'", classID)
	}
	return class.name, nil
}

// pcidb  The complete set of PCI vendors and  PCI classes.
type pcidb struct {
	vendors map[uint16]vendor
	classes map[uint32]class
	path    string
}

// vendor PCI vendors/devices/subVendors/SubDevices.
type vendor struct {
	name    string
	devices map[uint16]device
}

// subVendor PCI subVendor.
type subVendor struct {
	SubDevices map[uint16]SubDevice
}

// SubDevice PCI SubDevice.
type SubDevice struct {
	name string
}

// device PCI device.
type device struct {
	name       string
	subVendors map[uint16]subVendor
}

// class PCI classes/subClasses/Programming Interfaces.
type class struct {
	name       string
	subClasses map[uint32]subClass
}

// subClass PCI subClass.
type subClass struct {
	name    string
	progIfs map[uint8]progIf
}

// progIf PCI Programming Interface.
type progIf struct {
	name string
}

// parse parses a PCI IDS entry.
func (p *parser) parse() Interface {

	db := &pcidb{
		vendors: map[uint16]vendor{},
		classes: map[uint32]class{},
	}

	// Used for housekeeping, breadcrumb for aggregated types.
	var hkVendor vendor
	var hkDevice device

	var hkClass class
	var hkSubClass subClass

	var hkFullID uint32 = 0
	var hkFullName [2]string

	for {
		tok, lit := p.scan()

		// We're ignoring COMMENT, NEWLINE.
		// An EOF will break the loop.
		if tok == EOF {
			break
		}

		// PCI vendors -------------------------------------------------
		if tok == VENDOR {
			id, _ := strconv.ParseUint(lit.ID, 16, 16)
			db.vendors[uint16(id)] = vendor{
				name:    lit.name,
				devices: map[uint16]device{},
			}
			hkVendor = db.vendors[uint16(id)]
		}

		if tok == DEVICE {
			id, _ := strconv.ParseUint(lit.ID, 16, 16)
			hkVendor.devices[uint16(id)] = device{
				name:       lit.name,
				subVendors: map[uint16]subVendor{},
			}
			hkDevice = hkVendor.devices[uint16(id)]
		}

		if tok == SUBVENDOR {
			id, _ := strconv.ParseUint(lit.ID, 16, 16)
			hkDevice.subVendors[uint16(id)] = subVendor{
				SubDevices: map[uint16]SubDevice{},
			}
			subvendor := hkDevice.subVendors[uint16(id)]
			subid, _ := strconv.ParseUint(lit.name, 16, 16)
			subvendor.SubDevices[uint16(subid)] = SubDevice{
				name: lit.SubName,
			}
		}

		// PCI classes -------------------------------------------------
		if tok == CLASS {
			id, _ := strconv.ParseUint(lit.ID, 16, 32)
			db.classes[uint32(id)] = class{
				name:       lit.name,
				subClasses: map[uint32]subClass{},
			}
			hkClass = db.classes[uint32(id)]

			hkFullID = uint32(id) << 16
			hkFullID &= 0xFFFF0000
			hkFullName[0] = fmt.Sprintf("%s (%02x)", lit.name, id)
		}

		if tok == SUBCLASS {
			id, _ := strconv.ParseUint(lit.ID, 16, 8)
			hkClass.subClasses[uint32(id)] = subClass{
				name:    lit.name,
				progIfs: map[uint8]progIf{},
			}
			hkSubClass = hkClass.subClasses[uint32(id)]

			// Clear the last detected subclass.
			hkFullID &= 0xFFFF0000
			hkFullID |= uint32(id) << 8
			// Clear the last detected prog iface.
			hkFullID &= 0xFFFFFF00
			hkFullName[1] = fmt.Sprintf("%s (%02x)", lit.name, id)

			db.classes[uint32(hkFullID)] = class{
				name: hkFullName[0] + " | " + hkFullName[1],
			}
		}

		if tok == PROGIF {
			id, _ := strconv.ParseUint(lit.ID, 16, 8)
			hkSubClass.progIfs[uint8(id)] = progIf{
				name: lit.name,
			}

			finalID := hkFullID | uint32(id)

			name := fmt.Sprintf("%s (%02x)", lit.name, id)
			finalName := hkFullName[0] + " | " + hkFullName[1] + " | " + name

			db.classes[finalID] = class{
				name: finalName,
			}
		}

		if tok == ILLEGAL {
			fmt.Printf("warning: illegal token %s %s cannot parse PCI IDS, database may be incomplete ", lit.ID, lit.name)
		}
	}
	return db
}
