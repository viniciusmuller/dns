package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Header struct {
	id          int16
	flags       int16
	questions   int16
	answers     int16
	authorities int16
	additionals int16
}

type Query struct {
	name  string
	type_ int16
	class int16
}

func (h Header) Encode() bytes.Buffer {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, h.id)
	binary.Write(&b, binary.BigEndian, h.flags)
	binary.Write(&b, binary.BigEndian, h.questions)
	binary.Write(&b, binary.BigEndian, h.answers)
	binary.Write(&b, binary.BigEndian, h.authorities)
	binary.Write(&b, binary.BigEndian, h.additionals)
	return b
}

func (q Query) Encode() bytes.Buffer {
	var b bytes.Buffer

	encodedDomain := EncodeDomain(q.name)
	b.Write(encodedDomain.Bytes())

	binary.Write(&b, binary.BigEndian, q.type_)
	binary.Write(&b, binary.BigEndian, q.class)
	return b
}

type DNSRecord struct {
	name  string
	type_ int16
	class int16
	ttl   int32
	data  bytes.Buffer
}

func (d DNSRecord) GetIP() string {
	var length int16
	var s1, s2, s3, s4 uint8

	binary.Read(&d.data, binary.BigEndian, &length)
	binary.Read(&d.data, binary.BigEndian, &s1)
	binary.Read(&d.data, binary.BigEndian, &s2)
	binary.Read(&d.data, binary.BigEndian, &s3)
	binary.Read(&d.data, binary.BigEndian, &s4)

	return fmt.Sprintf("%d.%d.%d.%d", s1, s2, s3, s4)
}

func ParseDNSResponse(b *bytes.Buffer) DNSRecord {
	_ = ParseHeader(b)
	_ = ParseQuery(b)
	r := ParseRecord(b)

	fmt.Printf("domain resolved to: %s\n", r.GetIP())

	return r
}

func ParseRecord(b *bytes.Buffer) DNSRecord {
	var data bytes.Buffer

	var type_, class, length int16
	var ttl int32

	binary.Read(b, binary.BigEndian, &type_)
	binary.Read(b, binary.BigEndian, &class)
	binary.Read(b, binary.BigEndian, &ttl)
	binary.Read(b, binary.BigEndian, &length)

	data.ReadFrom(b)

	return DNSRecord{
		type_: type_,
		class: class,
		ttl: ttl,
		data: data,
	}
}

func ParseHeader(b *bytes.Buffer) Header {
	var id, flags, questions, answers, authorities, additionals int16

	binary.Read(b, binary.BigEndian, &id)
	binary.Read(b, binary.BigEndian, &flags)
	binary.Read(b, binary.BigEndian, &questions)
	binary.Read(b, binary.BigEndian, &answers)
	binary.Read(b, binary.BigEndian, &authorities)
	binary.Read(b, binary.BigEndian, &additionals)

	return Header{id: id,
		flags:       flags,
		questions:   questions,
		answers:     answers,
		authorities: authorities,
		additionals: additionals,
	}
}

func ParseDomain(b *bytes.Buffer) string {
	var result bytes.Buffer
	var segmentLen int8 = -1

	for segmentLen != 0 {
		binary.Read(b, binary.BigEndian, &segmentLen)

		for i := 0; i < int(segmentLen); i++ {
			var r byte

			r, err := b.ReadByte()
			if err != nil {
				fmt.Printf("could not read rune: %v\n", err)
			}

			result.WriteRune(rune(r))
		}

		// TODO: separate with dots
	}

	return result.String()
}

func ParseQuery(b *bytes.Buffer) Query {
	var type_, class int16

	domain := ParseDomain(b)
	binary.Read(b, binary.BigEndian, &type_)
	binary.Read(b, binary.BigEndian, &class)

	return Query{
		name: domain,
		type_: type_,
		class: class,
	}
}

func EncodeDomain(domain string) bytes.Buffer {
	var b bytes.Buffer

	for _, part := range strings.Split(domain, ".") {
		binary.Write(&b, binary.BigEndian, int8(len(part)))
		b.WriteString(part)
	}

	binary.Write(&b, binary.BigEndian, int8(0))
	return b
}

// TODO: use "enums" and add other record types, etc
var TYPE_A int16 = 1

func BuildQuery(domain string, recordType int16) bytes.Buffer {
	var RECURSION_DESIRED int16 = 1 << 8
	var CLASS_IN int16 = 1

	flags := RECURSION_DESIRED

	header := Header{
		id:          15,
		flags:       flags,
		questions:   1,
		answers:     0,
		authorities: 0,
		additionals: 0,
	}

	query := Query{
		name:  domain,
		type_: recordType,
		class: CLASS_IN,
	}
	qbuffer := query.Encode()

	hbuffer := header.Encode()
	hbuffer.Write(qbuffer.Bytes())
	return hbuffer
}

func main() {
	var target string

	if len(os.Args) > 1 {
		target = os.Args[1]
	} else {
		target = "www.example.com"
	}

	dnsquery := BuildQuery(target, TYPE_A)

	conn, err := net.DialTimeout("udp", "8.8.8.8:53", time.Second*3)
	if err != nil {
		log.Fatalf("Could not open UDP socket: %v", err)
	}

	_, err = conn.Write(dnsquery.Bytes())
	if err != nil {
		log.Fatalf("failed to write dns query: %v", err)
	}

	recv := make([]byte, 1024)

	_, err = conn.Read(recv)
	if err != nil {
		log.Fatalf("failed to read response: %v", err)
	}

	ParseDNSResponse(bytes.NewBuffer(recv))

	conn.Close()
}
