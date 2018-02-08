// Package goquirc provides an encapsulation of quirc library
// for fast qrcode processing
package goquirc

// #cgo LDFLAGS: -Lquirc -lquirc -lm
// #cgo CFLAGS: -Iquirc/lib -O3 -DQUIRC_MAX_REGIONS=65534 -fPIC
// #include <quirc.h>
// #include <stdio.h>
import "C"
import (
	"errors"
	"unsafe"
)

// Processing represents all informations needed by quirc to fully work
type Processing struct {
	qrStruct *C.struct_quirc
	code     C.struct_quirc_code
	data     C.struct_quirc_data
}

// Position describes a location in the input image buffer
type Position struct {
	X int
	Y int
}

// QRcode represents all informations about a qrcode
type QRcode struct {
	Corners       [4]Position
	Size          int
	Version       int
	ECCLevel      int
	Mask          int
	DataType      int
	Payload       string
	PayloadLength int
}

// Result contains all informations after a reveal process
type Result struct {
	Found  int
	Usable int
	Code   []QRcode
}

// Version provides current version of quirc
func (qr *Processing) Version() string {
	return C.GoString(C.quirc_version())
}

// Create allocates memory for library usage
func (qr *Processing) Create() error {
	if qr.qrStruct = C.quirc_new(); qr.qrStruct == nil {
		return errors.New("Failed to allocate memory")
	}
	return nil
}

// Destroy frees memory after library usage
func (qr *Processing) Destroy() {
	C.quirc_destroy(qr.qrStruct)
}

// Resize allocates memory for source image buffer
func (qr *Processing) Resize(w int, h int) error {
	if C.quirc_resize(qr.qrStruct, C.int(w), C.int(h)) == -1 {
		return errors.New("Failed to allocate video memory")
	}
	return nil
}

// Count returns the count of all Processings detected
func (qr *Processing) Count() int {
	return int(C.quirc_count(qr.qrStruct))
}

// Extract allows to work on a specific processing
func (qr *Processing) Extract(index int) {
	C.quirc_extract(qr.qrStruct, C.int(index), &qr.code)
}

// Decode gives informations from previously extracted processing
func (qr *Processing) Decode() error {
	if decodeError := C.quirc_decode(&qr.code, &qr.data); decodeError != C.QUIRC_SUCCESS {
		return errors.New(C.GoString(C.quirc_strerror(decodeError)))
	}
	return nil
}

// Load permits to load a byte array (source image) for further detection work
func (qr *Processing) Load(image *[]byte) {
	var w C.int
	var h C.int

	data := C.quirc_begin(qr.qrStruct, &w, &h)

	indexableData := (*[1 << 30]C.uint8_t)(unsafe.Pointer(data))

	var i C.int
	var imageSize C.int
	imageSize = w*h - 1
	for i = 0; i < imageSize; i++ {
		(*indexableData)[i] = *(*C.uint8_t)(unsafe.Pointer(&(*image)[i]))
	}
}

// End announces detection end
func (qr *Processing) End() {
	C.quirc_end(qr.qrStruct)
}

// Reveal allows to count all found processings by providing a source image with
// its dimensions and returns an error if an allocation went wrong
func (qr *Processing) Reveal(image *[]byte, w int, h int) (Result, error) {
	var result Result
	var err error

	if err = qr.Create(); err != nil {
		return result, err
	}
	defer qr.Destroy()

	if err = qr.Resize(w, h); err != nil {
		return result, err
	}

	qr.Load(image)
	qr.End()

	result.Found = qr.Count()
	result.Usable = result.Found
	for i := 0; i < result.Found; i++ {
		qr.Extract(i)
		if err = qr.Decode(); err == nil {
			result.Code = append(result.Code, QRcode{
				Corners: [4]Position{
					Position{
						(int)(qr.code.corners[0].x),
						(int)(qr.code.corners[0].y),
					},
					Position{
						(int)(qr.code.corners[1].x),
						(int)(qr.code.corners[1].y),
					},
					Position{
						(int)(qr.code.corners[2].x),
						(int)(qr.code.corners[2].y),
					},
					Position{
						(int)(qr.code.corners[3].x),
						(int)(qr.code.corners[3].y),
					}},
				DataType:      (int)(qr.data.data_type),
				ECCLevel:      (int)(qr.data.ecc_level),
				Mask:          (int)(qr.data.mask),
				Payload:       C.GoString((*C.char)(unsafe.Pointer(&qr.data.payload[0]))),
				PayloadLength: len(C.GoString((*C.char)(unsafe.Pointer(&qr.data.payload[0])))),
				Size:          (int)(qr.code.size),
				Version:       (int)(qr.data.version)})
		} else {
			result.Usable--
		}
	}

	return result, nil
}
