// Copyright 2018 Darell Tan. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the README.

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

type Guid struct {
	A    uint32
	B, C uint16
	D    [2]byte
	E    [6]byte
}

func (g Guid) String() string {
	return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		g.A, g.B, g.C, g.D[0], g.D[1],
		g.E[0], g.E[1], g.E[2], g.E[3], g.E[4], g.E[5])
}

type RegionDesc struct {
	Name   string
	Offset int64
	Size   int64
}

type VolumeHeader struct {
	Jmp               [3]byte
	Signature         [8]byte
	SectorSize        uint16
	SectorsPerCluster uint8
	ReservedClusters  uint16

	// unimportant fields
	_ [1 + 2 + 2 + 1 + 2 + 2 + 2 + 4 + 4]byte
	_ [4]byte

	NumSectors      uint64
	MftStartCluster uint64
	MetadataLcn     uint64

	_ [96]byte

	Guid        Guid
	InfoOffsets [3]uint64
	EOWOffsets  [2]uint64
}

type ValidationHeader struct {
	Size    uint16
	Version uint16
	Crc32   uint32
}

type InfoStructHeader struct {
	Signature [8]byte
	Size      uint16
	Version   uint16
}

type InfoStruct struct {
	InfoStructHeader

	_ [2 + 2]byte

	VolumeSize uint64

	ConvertSize         uint32
	HeaderSectors       uint32
	InfoOffsets         [3]uint64
	HeaderSectorsOffset uint64
}

func (s *InfoStruct) Read(r io.ReadSeeker) (size int64, err error) {
	var hdr InfoStructHeader
	size = -1

	err = binary.Read(r, binary.LittleEndian, &hdr)
	if err != nil {
		return
	}

	if !VerifySignature(hdr.Signature) {
		err = fmt.Errorf("invalid signature %q", hdr.Signature)
		return
	}

	size = int64(hdr.Size)
	switch hdr.Version {
	case 1:
		// no op

	case 2:
		size *= 16

	default:
		err = fmt.Errorf("unknown version %x")
		return
	}

	if size < 64 {
		err = fmt.Errorf("size too small")
		return
	}

	// rewind and read struct in full
	r.Seek(int64(-binary.Size(hdr)), 1)

	buf := make([]byte, size)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return
	}

	var validation ValidationHeader
	err = binary.Read(r, binary.LittleEndian, &validation)
	if err != nil {
		err = fmt.Errorf("cannot read validation header: %+v", err)
		return
	}

	// verify CRC
	checksum := crc32.ChecksumIEEE(buf)
	if checksum != validation.Crc32 {
		err = fmt.Errorf("validation checksum mismatch: stored %08x, computed %08x",
			validation.Crc32, checksum)
		return
	}

	size += int64(validation.Size)

	// parse whatever we read & verified
	r2 := bytes.NewReader(buf)
	err = binary.Read(r2, binary.LittleEndian, s)
	return
}

func VerifySignature(b [8]byte) bool { return string(b[:]) == "-FVE-FS-" }

const INFO_GUID string = "4967D63B-2E29-4AD8-8399-F6A339E3D001"

func fatal(format string, a ...interface{}) {
	if len(format) > 0 && format[len(format)-1:] != "\n" {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: blwipe [flags] <bitlocker-vol.img>\n\n")
	flag.PrintDefaults()
}

func main() {
	offset := flag.Int64("offset", 0, "offset into volume")
	verbose := flag.Bool("v", false, "show more information")
	doWipe := flag.Bool("wipe", false, "wipes cleartext data")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if *offset < 0 {
		fatal("offset cannot be negative")
	}

	f, err := os.OpenFile(flag.Arg(0), os.O_RDWR, 0644)
	if err != nil {
		fatal("can't open file: %s", err)
	}
	defer f.Close()

	f.Seek(*offset, 0)
	hdr := VolumeHeader{}
	err = binary.Read(f, binary.LittleEndian, &hdr)
	if err != nil {
		fatal("can't read header: %s", err)
	}

	// validate headers
	if !VerifySignature(hdr.Signature) {
		fatal("invalid volume header signature %q", hdr.Signature)
	}

	if hdr.SectorSize < 512 {
		fatal("weird sector size: %d", hdr.SectorSize)
	}

	if hdr.Guid.String() != INFO_GUID {
		fatal("unsupported GUID %v", hdr.Guid)
	}

	for i := 0; i < len(hdr.InfoOffsets); i++ {
		fmt.Printf("metadata offset %d: 0x%08x\n", i, hdr.InfoOffsets[i])
	}

	if *verbose {
		fmt.Printf("volume header:\n%+v\n", &hdr)
	}

	var validInfoSize int64
	var validInfoOffsets [3]int64

	// check info structs
	info := InfoStruct{}
	for i := 0; i < len(hdr.InfoOffsets); i++ {
		f.Seek(*offset+int64(hdr.InfoOffsets[i]), 0)

		infoSize, err := info.Read(f)
		if err != nil {
			fmt.Printf("can't parse metadata block %d: %+v\n", i, err)
			continue
		}

		// record valid data here
		validInfoSize = infoSize
		for idx, off := range info.InfoOffsets {
			validInfoOffsets[idx] = int64(off)
		}

		// round up to sector size
		infoSize = (infoSize + int64(hdr.SectorSize) - 1) & ^(int64(hdr.SectorSize) - 1)

		fmt.Printf("metadata block %d (size %d):", i, infoSize)
		if *verbose {
			fmt.Printf("\n%+v\n", &info)
		} else {
			fmt.Printf(" parsed OK\n")
		}
	}

	if validInfoSize == 0 {
		fatal("invalid or no metadata blocks found!")
	}

	if *doWipe {
		eraseRegions := []RegionDesc{
			{"volume header", 0, int64(hdr.SectorSize)},
			{"metadata block 0", validInfoOffsets[0], validInfoSize},
			{"metadata block 1", validInfoOffsets[1], validInfoSize},
			{"metadata block 2", validInfoOffsets[2], validInfoSize},
		}

		for _, region := range eraseRegions {
			eraseBuf := make([]byte, region.Size)
			_, err = rand.Read(eraseBuf)
			if err != nil {
				fatal("unable to generate rand bytes: %v", err)
				break
			}

			fmt.Printf("overwriting %s at offset 0x%x size %d...\n",
				region.Name, region.Offset, region.Size)

			f.Seek(*offset+region.Offset, 0)
			_, err = f.Write(eraseBuf)
			if err != nil {
				fmt.Printf("unable to write region: %v\n", err)
				continue
			}
		}
	}
}
