package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
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
	name  bytes.Buffer
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
	b.Write(q.name.Bytes())
	binary.Write(&b, binary.BigEndian, q.type_)
	binary.Write(&b, binary.BigEndian, q.class)
	return b
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
		name:  EncodeDomain(domain),
		type_: recordType,
		class: CLASS_IN,
	}
	qbuffer := query.Encode()

	hbuffer := header.Encode()
	hbuffer.Write(qbuffer.Bytes())
	return hbuffer
}

func main() {
	dnsquery := BuildQuery("www.example.com", TYPE_A)

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

	fmt.Printf("got response: % x\n", recv[:50])

	conn.Close()
}
