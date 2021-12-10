// Copyright 2021 PairMesh, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stun generates STUN request packets and parses response packets.
package stun

import (
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"net"
)

// IANA Considerations
// https://datatracker.ietf.org/doc/html/rfc5389#section-18
const (
	attrNumSoftware      = 0x8022
	attrNumFingerprint   = 0x8028
	attrMappedAddress    = 0x0001
	attrXorMappedAddress = 0x0020

	// This alternative attribute type is not
	// mentioned in the RFC, but the shift into
	// the "comprehension-optional" range seems
	// like an easy mistake for a server to make.
	// And servers appear to send it.
	attrXorMappedAddressAlt = 0x8020

	software       = "meshnode" // notably: 8 bytes long, so no padding
	bindingRequest = "\x00\x01"
	magicCookie    = "\x21\x12\xa4\x42"
	lenFingerprint = 8 // 2+byte header + 2-byte length + 4-byte crc32
	headerLen      = 20
)

// TxID is a transaction ID.
type TxID [12]byte

// NewTxID returns a new random TxID.
func NewTxID() TxID {
	var tx TxID
	if _, err := crand.Read(tx[:]); err != nil {
		panic(err)
	}
	return tx
}

// Request generates a binding request STUN packet.
// The transaction ID, tID, should be a random sequence of bytes.
func Request(tID TxID) []byte {
	// STUN header, RFC5389 Section 6.
	// https://datatracker.ietf.org/doc/html/rfc5389#section-6
	const lenAttrSoftware = 4 + len(software)
	b := make([]byte, 0, headerLen+lenAttrSoftware+lenFingerprint)
	b = append(b, bindingRequest...)
	b = appendU16(b, uint16(lenAttrSoftware+lenFingerprint)) // number of bytes following header
	b = append(b, magicCookie...)
	b = append(b, tID[:]...)

	// Attribute SOFTWARE, RFC5389 Section 15.10.
	// https://datatracker.ietf.org/doc/html/rfc5389#section-15.10
	// Type-Length-Value
	b = appendU16(b, attrNumSoftware)
	b = appendU16(b, uint16(len(software)))
	b = append(b, software...)

	// Attribute FINGERPRINT, RFC5389 Section 15.5.
	// https://datatracker.ietf.org/doc/html/rfc5389#section-15.5
	fp := fingerPrint(b)
	b = appendU16(b, attrNumFingerprint)
	b = appendU16(b, 4)
	b = appendU32(b, fp)

	return b
}

func fingerPrint(b []byte) uint32 { return crc32.ChecksumIEEE(b) ^ 0x5354554e }

func appendU16(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// ParseBindingRequest parses a STUN binding request.
//
// It returns an error unless it advertises that it came from
// PairMesh.
func ParseBindingRequest(b []byte) (TxID, error) {
	if !Is(b) {
		return TxID{}, ErrNotSTUN
	}
	if string(b[:len(bindingRequest)]) != bindingRequest {
		return TxID{}, ErrNotBindingRequest
	}
	var txID TxID
	copy(txID[:], b[8:8+len(txID)])
	var softwareOK bool
	var lastAttr uint16
	var gotFP uint32
	if err := foreachAttr(b[headerLen:], func(attrType uint16, a []byte) error {
		lastAttr = attrType
		if attrType == attrNumSoftware && string(a) == software {
			softwareOK = true
		}
		if attrType == attrNumFingerprint && len(a) == 4 {
			gotFP = binary.BigEndian.Uint32(a)
		}
		return nil
	}); err != nil {
		return TxID{}, err
	}
	if !softwareOK {
		return TxID{}, ErrWrongSoftware
	}
	if lastAttr != attrNumFingerprint {
		return TxID{}, ErrNoFingerprint
	}
	wantFP := fingerPrint(b[:len(b)-lenFingerprint])
	if gotFP != wantFP {
		return TxID{}, ErrWrongFingerprint
	}
	return txID, nil
}

var (
	ErrNotSTUN            = errors.New("response is not a STUN packet")
	ErrNotSuccessResponse = errors.New("STUN packet is not a response")
	ErrMalformedAttrs     = errors.New("STUN response has malformed attributes")
	ErrNotBindingRequest  = errors.New("STUN request not a binding request")
	ErrWrongSoftware      = errors.New("STUN request came from non-PairMesh software")
	ErrNoFingerprint      = errors.New("STUN request didn't end in fingerprint")
	ErrWrongFingerprint   = errors.New("STUN request had bogus fingerprint")
)

func foreachAttr(b []byte, fn func(attrType uint16, a []byte) error) error {
	for len(b) > 0 {
		if len(b) < 4 {
			return ErrMalformedAttrs
		}
		attrType := binary.BigEndian.Uint16(b[:2])
		attrLen := int(binary.BigEndian.Uint16(b[2:4]))
		attrLenWithPad := (attrLen + 3) &^ 3
		b = b[4:]
		if attrLenWithPad > len(b) {
			return ErrMalformedAttrs
		}
		if err := fn(attrType, b[:attrLen]); err != nil {
			return err
		}
		b = b[attrLenWithPad:]
	}
	return nil
}

// Response generates a binding response.
func Response(txID TxID, ip net.IP, port uint16) []byte {
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	var fam byte
	switch len(ip) {
	case net.IPv4len:
		fam = 1
	case net.IPv6len:
		fam = 2
	default:
		return nil
	}
	attrsLen := 8 + len(ip)
	b := make([]byte, 0, headerLen+attrsLen)

	// Header
	b = append(b, 0x01, 0x01) // success
	b = appendU16(b, uint16(attrsLen))
	b = append(b, magicCookie...)
	b = append(b, txID[:]...)

	// Attributes (well, one)
	b = appendU16(b, attrXorMappedAddress)
	b = appendU16(b, uint16(4+len(ip)))
	b = append(b,
		0, // unused byte
		fam)
	b = appendU16(b, port^0x2112) // first half of magicCookie
	for i, o := range []byte(ip) {
		if i < 4 {
			b = append(b, o^magicCookie[i])
		} else {
			b = append(b, o^txID[i-len(magicCookie)])
		}
	}
	return b
}

// ParseResponse parses a successful binding response STUN packet.
// The IP address is extracted from the XOR-MAPPED-ADDRESS attribute.
// The returned addr slice is owned by the caller and does not alias b.
func ParseResponse(b []byte) (txID TxID, addr []byte, port uint16, err error) {
	if !Is(b) {
		return txID, nil, 0, ErrNotSTUN
	}

	copy(txID[:], b[8:8+len(txID)])
	if b[0] != 0x01 || b[1] != 0x01 {
		return txID, nil, 0, ErrNotSuccessResponse
	}

	attrsLen := int(binary.BigEndian.Uint16(b[2:4]))
	b = b[headerLen:] // remove STUN header
	if attrsLen > len(b) {
		return txID, nil, 0, ErrMalformedAttrs

	} else if len(b) > attrsLen {
		b = b[:attrsLen] // trim trailing packet bytes
	}

	var addr6, fallbackAddr, fallbackAddr6 []byte
	var port6, fallbackPort, fallbackPort6 uint16

	// Read through the attributes.
	// The the addr+port reported by XOR-MAPPED-ADDRESS
	// as the canonical value. If the attribute is not
	// present but the STUN server responds with
	// MAPPED-ADDRESS we fall back to it.
	if err := foreachAttr(b, func(attrType uint16, attr []byte) error {
		switch attrType {
		case attrXorMappedAddress, attrXorMappedAddressAlt:
			a, p, err := xorMappedAddress(txID, attr)
			if err != nil {
				return err
			}
			if len(a) == 16 {
				addr6, port6 = a, p

			} else {
				addr, port = a, p
			}

		case attrMappedAddress:
			a, p, err := mappedAddress(attr)
			if err != nil {
				return ErrMalformedAttrs
			}
			if len(a) == 16 {
				fallbackAddr6, fallbackPort6 = a, p
			} else {
				fallbackAddr, fallbackPort = a, p
			}
		}
		return nil

	}); err != nil {
		return TxID{}, nil, 0, err
	}

	if addr != nil {
		return txID, addr, port, nil
	}
	if fallbackAddr != nil {
		return txID, append([]byte{}, fallbackAddr...), fallbackPort, nil
	}
	if addr6 != nil {
		return txID, addr6, port6, nil
	}
	if fallbackAddr6 != nil {
		return txID, append([]byte{}, fallbackAddr6...), fallbackPort6, nil
	}
	return txID, nil, 0, ErrMalformedAttrs
}

func xorMappedAddress(tID TxID, b []byte) (addr []byte, port uint16, err error) {
	// XOR-MAPPED-ADDRESS attribute, RFC5389 Section 15.2
	// https://datatracker.ietf.org/doc/html/rfc5389#section-15.2
	if len(b) < 4 {
		return nil, 0, ErrMalformedAttrs
	}
	xorPort := binary.BigEndian.Uint16(b[2:4])
	addrField := b[4:]
	port = xorPort ^ 0x2112 // first half of magicCookie

	addrLen := familyAddrLen(b[1])
	if addrLen == 0 {
		return nil, 0, ErrMalformedAttrs
	}
	if len(addrField) < addrLen {
		return nil, 0, ErrMalformedAttrs
	}
	xorAddr := addrField[:addrLen]
	addr = make([]byte, addrLen)
	for i := range xorAddr {
		if i < len(magicCookie) {
			addr[i] = xorAddr[i] ^ magicCookie[i]

		} else {
			addr[i] = xorAddr[i] ^ tID[i-len(magicCookie)]
		}
	}

	return addr, port, nil
}

func familyAddrLen(fam byte) int {
	switch fam {
	case 0x01: // IPv4
		return net.IPv4len

	case 0x02: // IPv6
		return net.IPv6len

	default:
		return 0
	}
}

func mappedAddress(b []byte) (addr []byte, port uint16, err error) {
	if len(b) < 4 {
		return nil, 0, ErrMalformedAttrs
	}
	port = uint16(b[2])<<8 | uint16(b[3])
	addrField := b[4:]
	addrLen := familyAddrLen(b[1])
	if addrLen == 0 {
		return nil, 0, ErrMalformedAttrs
	}
	if len(addrField) < addrLen {
		return nil, 0, ErrMalformedAttrs
	}
	return append([]byte(nil), addrField[:addrLen]...), port, nil
}

// Is reports whether b is a STUN message.
func Is(b []byte) bool {
	return len(b) >= headerLen &&
		b[0]&0b11000000 == 0 && // top two bits must be zero
		string(b[4:8]) == magicCookie
}
